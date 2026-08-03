//go:build c2s

package integration

import (
	"context"
	"crypto"
	"crypto/rand"
	"fmt"
	"net/http"
	"sort"
	"testing"

	vocab "github.com/go-ap/activitypub"
	ap2 "github.com/go-ap/fedbox/activitypub"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/integration/internal/containers/fedbox"
	"github.com/go-ap/fedbox/integration/internal/tests"
	ap "github.com/go-ap/fedbox/integration/internal/vocab"
	"github.com/go-ap/filters"
	"github.com/go-ap/storage-conformance-suite/gen"
	"golang.org/x/crypto/ed25519"
)

func plausibleRandomObjects(pubKey crypto.PublicKey, cnt int) vocab.ItemCollection {
	service := root(c2sRootIRI, ap.HasPublicKey(pubKey))

	gen.DefaultHost = c2sRootIRI
	gen.SetItemID = func(it vocab.Item) {
		ap2.GenerateID(it, c2sRootIRI, service)
	}
	gen.Root = service

	randomObjects := make(vocab.ItemCollection, 0, cnt)
	for _, ob := range gen.PlausibleStorage(gen.Root, cnt) {
		if it, ok := ob.(vocab.Item); ok {
			randomObjects = append(randomObjects, it)
		}
	}

	return randomObjects
}

func addActorOutboxTest(items vocab.ItemCollection, actor vocab.Item, ff ...filters.Check) tests.HTTPTest {
	name := vocab.NameOf(actor)
	testName := fmt.Sprintf("%s outbox", name)
	if len(ff) > 0 {
		testName = fmt.Sprintf("%s outbox:%s", name, filters.Checks(ff).GoString())
	}

	// NOTE(marius): we filter the global items collection for what matches current actor and filters
	wantItems, _ := append(filters.Checks{filters.Actor(filters.SameID(actor.GetLink()))}, ff...).Run(items).(vocab.ItemCollection)
	paginatedURL := buildFilterURL(vocab.Outbox.IRI(actor), ff...)
	return tests.HTTPTest{
		Name: testName,
		Req:  tests.Request().URL(paginatedURL),
		Res: tests.Response().HasCode(http.StatusOK).
			ItemMatch(
				tests.HasID(vocab.IRI(paginatedURL)),
				tests.IsType(vocab.OrderedCollectionPageType),
				tests.HasTotalItems(len(wantItems)),
				tests.HasItems(wantItems...),
			),
	}
}

func actorTests(items vocab.ItemCollection, actors ...vocab.Item) []tests.RunnableTest {
	tests := make([]tests.RunnableTest, 0, len(actors))
	for _, act := range actors {
		tests = append(tests, addActorOutboxTest(items, act))
	}
	return tests
}

func paginateByIRIs(items vocab.ItemCollection, cnt int) vocab.IRIs {
	result := make(vocab.IRIs, 0, len(items)/cnt)
	for i := range items {
		if (i+1)%cnt == 0 {
			next := items[i]
			result = append(result, next.GetLink())
		}
	}
	return result
}

func addRootInboxTest(items vocab.ItemCollection, maxItems int, ff ...filters.Check) tests.HTTPTest {
	testName := fmt.Sprintf("root inbox:%s", filters.Checks(ff).GoString())
	// NOTE(marius): we filter the global items collection for what matches current actor and filters
	wantItems, _ := filters.PaginateCollection(items, ff...).(vocab.ItemCollection)
	paginatedURL := buildFilterURL(vocab.Inbox.IRI(c2sRootIRI), ff...)
	return tests.HTTPTest{
		Name: testName,
		Req:  tests.Request().URL(paginatedURL),
		Res: tests.Response().HasCode(http.StatusOK).
			ItemMatch(
				tests.HasID(vocab.IRI(paginatedURL)),
				tests.IsType(vocab.OrderedCollectionPageType),
				tests.HasTotalItems(maxItems),
				tests.HasItems(wantItems...),
			),
	}
}

func rootInboxPaginationTest(items []vocab.Item, cnt, maxItems int, ff ...filters.Check) []tests.RunnableTest {
	if len(items) == 0 || cnt == 0 {
		return nil
	}
	if cnt == 1 {
		return []tests.RunnableTest{addRootInboxTest(items, maxItems, ff...)}
	}
	tests := make([]tests.RunnableTest, 0, len(items)/cnt+1)
	tests = append(tests, addRootInboxTest(items, maxItems, append(ff, filters.WithMaxCount(cnt))...))
	for _, it := range paginateByIRIs(items, cnt) {
		tests = append(tests, addRootInboxTest(items, maxItems, append(ff, filters.After(filters.SameID(it.GetLink())), filters.WithMaxCount(cnt))...))
	}
	return tests
}

func rootInboxAdditionalFiltersTest(items vocab.ItemCollection, ff ...filters.Check) []tests.RunnableTest {
	wantItems, _ := filters.Checks(ff).Run(items).(vocab.ItemCollection)
	if len(wantItems) == 0 {
		return nil
	}
	return rootInboxPaginationTest(wantItems, len(wantItems)/2, len(items), ff...)
}

func Test_CollectionFilters(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	pw := rand.Text()[:8]

	items := plausibleRandomObjects(publicKey, 60)
	images := c.Suite(fedbox.New(
		fedbox.WithConfig(fedbox.ConfigFromBuildInfo(defaultC2SOptions)),
		fedbox.WithArgs([]string{"--bootstrap"}),
		fedbox.WithTestLogger(t, Verbose),
		fedbox.WithImageName(fedBOXImageName),
		fedbox.WithKey(privateKey),
		fedbox.WithPw(pw),
		fedbox.WithItems(items...),
	))

	ctx := context.Background()
	cont, err := c.Start(ctx, t, images...)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	// NOTE(marius): we load the actors created by the Service actor
	actorCreateItems, _ := filters.Checks{
		filters.HasType(vocab.CreateType),
		filters.Actor(filters.SameID(c2sRootIRI)),
		filters.Object(filters.HasType(vocab.ActorTypes...)),
	}.Run(items).(vocab.ItemCollection)
	actorItems := make(vocab.ItemCollection, 0, len(actorCreateItems))
	for _, act := range actorCreateItems {
		_ = vocab.OnActivity(act, func(act *vocab.Activity) error {
			actorItems = append(actorItems, act.Object)
			return nil
		})
	}
	// NOTE(marius): we append an activity matching the one generated by FedBOX to create
	// the root Service actor.
	items = append(items, RootCreate)

	// NOTE(marius): this sorting is important so our after/before activity IDs
	// match the tests we generate for pagination.
	sort.SliceStable(items, func(i int, j int) bool {
		return vocab.ItemOrderTimestamp(items[i], items[j])
	})

	// NOTE(marius): get the activities created by the root service actor
	allOutboxItems, _ := filters.Checks{
		filters.Actor(filters.SameID(c2sRootIRI)),
	}.Run(items).(vocab.ItemCollection)

	toRun := []tests.RunnableTest{
		tests.HTTPTest{
			Name: "root outbox",
			Req:  tests.Request().IRI(vocab.Outbox.IRI(c2sRootIRI)),
			Res: tests.Response().HasCode(http.StatusOK).
				ItemMatch(
					tests.HasID(vocab.Outbox.IRI(c2sRootIRI)+"?maxItems=100"),
					tests.IsType(vocab.OrderedCollectionPageType),
					tests.HasTotalItems(len(allOutboxItems)),
					tests.HasItems(allOutboxItems...),
				),
		},
		// NOTE(marius): all activities have also landed in the root actor's inbox
		tests.HTTPTest{
			Name: "root inbox",
			Req:  tests.Request().IRI(vocab.Inbox.IRI(c2sRootIRI)),
			Res: tests.Response().HasCode(http.StatusOK).
				ItemMatch(
					// NOTE(marius): automatic redirect from FedBOX to a collection page
					// limited to 100 items.
					tests.HasID(vocab.Inbox.IRI(c2sRootIRI)+"?maxItems=100"),
					tests.IsType(vocab.OrderedCollectionPageType),
					tests.HasTotalItems(len(items)),
					tests.HasItems(items...),
				),
		},
	}
	toRun = append(toRun, actorTests(items, actorItems...)...)

	// NOTE(marius): pagination tests
	toRun = append(toRun, rootInboxPaginationTest(items, len(items)/20, len(items))...)
	toRun = append(toRun, rootInboxPaginationTest(items, len(items)/10, len(items))...)
	toRun = append(toRun, rootInboxPaginationTest(items, len(items)/6, len(items))...)
	toRun = append(toRun, rootInboxPaginationTest(items, len(items)/2, len(items))...)
	// NOTE(marius): use a pagination count that's not divisible by the items slice length
	toRun = append(toRun, rootInboxPaginationTest(items, (len(items)/6)+1, len(items))...)
	// NOTE(marius): add some additional filters on the inbox
	toRun = append(toRun, rootInboxAdditionalFiltersTest(items, filters.HasType(vocab.CreateType))...)
	toRun = append(toRun, rootInboxAdditionalFiltersTest(items, filters.HasType(vocab.LikeType))...)
	toRun = append(toRun, rootInboxAdditionalFiltersTest(items, filters.Object(filters.HasType(vocab.LinkType)))...)
	toRun = append(toRun, rootInboxAdditionalFiltersTest(items, filters.Object(filters.HasType(vocab.ImageType)))...)
	toRun = append(toRun, rootInboxAdditionalFiltersTest(items, filters.Object(filters.HasType(vocab.AudioType)))...)
	toRun = append(toRun, rootInboxAdditionalFiltersTest(items, filters.HasType(vocab.LikeType, vocab.DislikeType), filters.Object(filters.HasType(vocab.ImageType, vocab.VideoType, vocab.AudioType)))...)
	toRun = append(toRun, rootInboxAdditionalFiltersTest(items, filters.HasType(vocab.IgnoreType), filters.Object(filters.HasType(vocab.ActorTypes...)))...)

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(ctx, cont))
	}
}
