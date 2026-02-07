package integration

import (
	"context"
	"crypto/rand"
	"net/http"
	"testing"

	vocab "github.com/go-ap/activitypub"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/integration/internal/tests"
	ap "github.com/go-ap/fedbox/integration/internal/vocab"
	"golang.org/x/crypto/ed25519"
)

var (
	defaultPublicKey, defaultPrivateKey, _ = ed25519.GenerateKey(rand.Reader)

	defaultPassword = "asd"
	defaultC2SEnv   = map[string]string{
		"HTTP_PORT":    "4000",
		"SSH_PORT":     "4044",
		"HTTPS":        "false",
		"HOSTNAME":     "fedbox",
		"STORAGE_PATH": "/storage",
	}
)

var service = ap.Actor(
	ap.HasID("http://fedbox/"),
	ap.HasType(vocab.ServiceType),
	ap.HasPreferredUsername("self"),
	ap.HasName("FedBOX"),
	ap.HasAttributedTo("https://github.com/mariusor"),
	ap.HasAudience(vocab.PublicNS),
	ap.HasContext("https://github.com/go-ap/fedbox"),
	ap.HasSummary("Generic ActivityPub service"),
	ap.HasURL("http://fedbox/"),
	ap.HasStream("http://fedbox/actors"),
	ap.HasStream("http://fedbox/activities"),
	ap.HasStream("http://fedbox/objects"),
	ap.HasPublicKey(defaultPublicKey),
	ap.HasAuthEp("http://fedbox/oauth/authorize"),
	ap.HasTokenEp("http://fedbox/oauth/token"),
	ap.HasProxyURL("http://fedbox/proxyUrl"),
)

var admin1 = ap.Actor(
	ap.HasID("https://fedbox/actors/1"),
	ap.HasType(vocab.PersonType),
	ap.HasAttributedTo("https://fedbox"),
	ap.HasAudience(vocab.PublicNS),
	ap.HasGenerator("https://fedbox"),
	ap.HasURL("https://fedbox/actors/1"),
	ap.HasPreferredUsername("admin"),
	ap.HasTag(tag0),
	ap.HasAuthEp("https://fedbox/actors/1/oauth/authorize"),
	ap.HasTokenEp("https://fedbox/actors/1/oauth/token"),
	ap.HasSharedInbox("https://fedbox/inbox"),
	ap.HasProxyURL("https://fedbox/proxyUrl"),
)

var actor2 = ap.Actor(
	ap.HasID("https://fedbox/actors/2"),
	ap.HasType(vocab.PersonType),
	ap.HasAttributedTo("https://fedbox"),
	ap.HasAudience(vocab.PublicNS),
	ap.HasContent("Generated actor"),
	ap.HasSummary("Generated actor"),
	ap.HasGenerator("https://fedbox"),
	ap.HasURL("https://fedbox/actors/2"),
	ap.HasLiked(),
	ap.HasPreferredUsername("johndoe"),
	ap.HasPublished("2019-08-11T13:14:47.000000000+02:00"),
	ap.HasUpdated("2019-08-11T13:14:47.000000000+02:00"),
	ap.HasName("Johnathan Doe"),
	ap.HasSharedInbox("https://fedbox/inbox"),
	ap.HasAuthEp("https://fedbox/oauth/authorize"),
	ap.HasTokenEp("https://fedbox/oauth/token"),
	ap.HasSharedInbox("https://fedbox/inbox"),
)

var tag0 = ap.Object(
	ap.HasID("https://fedbox/objects/0"),
	ap.HasName("#sysop"),
	ap.HasAttributedTo("https://fedbox"),
	ap.HasTo(vocab.PublicNS),
)

var object1 = ap.Object(
	ap.HasID("https://fedbox/objects/1"),
	ap.HasType(vocab.NoteType),
	ap.HasContent("<p>Hello</p><code>FedBOX</code>!</p>\n"),
	ap.HasMediaType("text/html"),
	ap.HasPublished("2019-09-27T14:26:43.000000000Z"),
	ap.HasUpdated("2019-09-27T14:26:43.000000000Z"),
	ap.HasAttributedTo(admin1.ID),
	ap.HasSource("Hello `FedBOX`!", "text/markdown"),
	ap.HasTo("https://www.w3.org/ns/activitystreams#Public"),
)

func Test_Fetch(t *testing.T) {
	toRun := []tests.IOTest{
		{
			Name: "service",
			IN:   tests.IN(tests.WithURL("http://fedbox")),
			OUT:  tests.OUT(tests.HasCode(http.StatusOK), tests.HasItem(service)),
		},
		{
			Name: "actors/1",
			IN:   tests.IN(tests.WithURL("http://fedbox/actors/1")),
			OUT:  tests.OUT(tests.HasCode(http.StatusOK), tests.HasItem(admin1)),
		},
		{
			Name: "objects/0",
			IN:   tests.IN(tests.WithURL("http://fedbox/objects/0")),
			OUT:  tests.OUT(tests.HasCode(http.StatusOK), tests.HasItem(tag0)),
		},
		{
			Name: "objects/1",
			IN:   tests.IN(tests.WithURL("http://fedbox/objects/1")),
			OUT:  tests.OUT(tests.HasCode(http.StatusOK), tests.HasItem(object1)),
		},
		{
			Name: "actors/2",
			IN:   tests.IN(tests.WithURL("http://fedbox/actors/2")),
			OUT:  tests.OUT(tests.HasCode(http.StatusOK), tests.HasItem(actor2)),
		},
	}

	envType := c.ExtractEnvTagFromBuild()
	var c2sFedBOX = c.C2SfedBOX(
		c.WithEnv(defaultC2SEnv),
		c.WithArgs([]string{"--env", envType, "--bootstrap"}),
		c.WithImageName(fedBOXImageName),
		c.WithKey(defaultPrivateKey),
		c.WithUser(service.ID),
		c.WithPw(defaultPassword),
		c.WithItems(tag0, object1, admin1, actor2),
	)

	images := c.Suite{c2sFedBOX}

	ctx := context.Background()
	cont, err := c.Init(ctx, t, images...)
	if err != nil {
		t.Fatalf("unable to initialize containers: %s", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	for _, test := range toRun {
		t.Run(test.Name, test.Run(ctx, cont))
	}
}
