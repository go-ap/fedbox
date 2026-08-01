//go:build c2s

package integration

import (
	"context"
	"crypto"
	"crypto/rand"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

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

func buildFilterURL(iri vocab.IRI, ff ...filters.Check) string {
	if len(ff) == 0 {
		return string(iri)
	}
	return string(iri) + "?" + filters.ToValues(ff...).Encode()
}

func addActorOutboxTest(items vocab.ItemCollection) func(actor vocab.Item, ff ...filters.Check) tests.HTTPTest {
	return func(actor vocab.Item, ff ...filters.Check) tests.HTTPTest {
		name := vocab.NameOf(actor)
		testName := fmt.Sprintf("%s outbox", name)
		if len(ff) > 0 {
			testName = fmt.Sprintf("%s outbox:%s", name, filters.Checks(ff).GoString())
		}

		// NOTE(marius): we filter the global items collection for what matches current actor and filters
		wantItems, _ := append(filters.Checks{filters.Actor(filters.SameID(actor.GetLink()))}, ff...).Run(items).(vocab.ItemCollection)
		return tests.HTTPTest{
			Name: testName,
			Req:  tests.Request().URL(buildFilterURL(vocab.Outbox.IRI(actor), ff...)),
			Res: tests.Response().HasCode(http.StatusOK).
				ItemMatch(
					tests.IsType(vocab.OrderedCollectionPageType),
					tests.HasTotalItems(len(wantItems)),
					tests.HasItems(wantItems...),
				),
		}
	}
}

func actorTests(items vocab.ItemCollection) func(actors ...vocab.Item) []tests.RunnableTest {
	return func(actors ...vocab.Item) []tests.RunnableTest {
		tests := make([]tests.RunnableTest, 0, len(actors))
		for _, act := range actors {
			tests = append(tests, addActorOutboxTest(items)(act))
		}
		return tests
	}
}

func paginateByIRIs(items vocab.ItemCollection, cnt int) vocab.IRIs {
	result := make(vocab.IRIs, 0, len(items)/cnt)
	for i, it := range items {
		if (i+1)%cnt == 0 {
			result = append(result, it.GetLink())
		}
	}
	return result
}

func addRootInboxTest(items vocab.ItemCollection) func(ff ...filters.Check) tests.HTTPTest {
	return func(ff ...filters.Check) tests.HTTPTest {
		testName := fmt.Sprintf("root inbox:%s", filters.Checks(ff).GoString())
		// NOTE(marius): we filter the global items collection for what matches current actor and filters
		wantItems, _ := filters.PaginateCollection(items, ff...).(vocab.ItemCollection)
		return tests.HTTPTest{
			Name: testName,
			Req:  tests.Request().URL(buildFilterURL(vocab.Inbox.IRI(c2sRootIRI), ff...)),
			Res: tests.Response().HasCode(http.StatusOK).
				ItemMatch(
					tests.IsType(vocab.OrderedCollectionPageType),
					tests.HasTotalItems(len(items)+1), // NOTE(marius): the Create Service activity is also here.
					tests.HasItems(wantItems...),
				),
		}
	}
}

func rootInboxPaginationTest(items []vocab.Item) func(cnt int) []tests.RunnableTest {
	slices.SortFunc(items, func(a vocab.Item, b vocab.Item) int {
		var t1 time.Time
		var t2 time.Time
		_ = vocab.OnObject(a, func(o *vocab.Object) error {
			t1 = o.Published
			return nil
		})
		_ = vocab.OnObject(b, func(o *vocab.Object) error {
			t2 = o.Published
			return nil
		})
		return int(t1.Sub(t2))
	})
	return func(cnt int) []tests.RunnableTest {
		tests := make([]tests.RunnableTest, 0, len(items)/cnt)
		for _, it := range paginateByIRIs(items, cnt) {
			tests = append(tests, addRootInboxTest(items)(filters.After(filters.SameID(it.GetLink())), filters.WithMaxCount(cnt)))
		}
		return tests
	}
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

	byAdminFilter := filters.Checks{filters.Actor(filters.SameID(c2sRootIRI))}
	adminItems, _ := byAdminFilter.Run(items).(vocab.ItemCollection)

	// NOTE(marius): we get only the actors created by the Service actor
	createActorsFilter := filters.Checks{
		filters.HasType(vocab.CreateType),
		filters.Actor(filters.SameID(c2sRootIRI)),
		filters.Object(filters.HasType(vocab.ActorTypes...)),
	}

	actorCreateItems, _ := createActorsFilter.Run(items).(vocab.ItemCollection)
	actorItems := make(vocab.ItemCollection, 0, len(actorCreateItems))
	for _, act := range actorCreateItems {
		_ = vocab.OnActivity(act, func(act *vocab.Activity) error {
			actorItems = append(actorItems, act.Object)
			return nil
		})
	}

	allOutboxItems := append(vocab.ItemCollection{RootCreate}, adminItems...)
	allInboxItems := append(vocab.ItemCollection{RootCreate}, items...)
	toRun := []tests.RunnableTest{
		tests.HTTPTest{
			Name: "root outbox",
			Req:  tests.Request().IRI(vocab.Outbox.IRI(c2sRootIRI)),
			Res: tests.Response().HasCode(http.StatusOK).
				ItemMatch(
					tests.IsType(vocab.OrderedCollectionPageType),
					tests.HasTotalItems(len(allOutboxItems)),
					tests.HasItems(allOutboxItems...),
				),
		},
		tests.HTTPTest{
			Name: "root inbox",
			Req:  tests.Request().IRI(vocab.Inbox.IRI(c2sRootIRI)),
			Res: tests.Response().HasCode(http.StatusOK).
				ItemMatch(
					tests.IsType(vocab.OrderedCollectionPageType),
					tests.HasTotalItems(len(allInboxItems)),
					tests.HasItems(allInboxItems...),
				),
		},
	}

	//toRun = append(toRun, rootInboxPaginationTest(0)...)
	//toRun = append(toRun, rootInboxPaginationTest(6)...)
	toRun = append(toRun, rootInboxPaginationTest(items)(10)...)
	toRun = append(toRun, actorTests(items)(actorItems...)...)

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(ctx, cont))
	}
}
