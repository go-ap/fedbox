package integration

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	vocab "github.com/go-ap/activitypub"
)

var httpClient = http.Client{
	Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
}

var service = actor(
	"http://fedbox/",
	hasType(vocab.ServiceType),
	hasName("self"),
	hasAttributedTo("https://github.com/mariusor"),
	hasAudience(vocab.PublicNS),
	hasContext("https://github.com/go-ap/fedbox"),
	hasSummary("Generic ActivityPub service"),
	hasURL("http://fedbox/"),
	hasStream("http://fedbox/actors"),
	hasStream("http://fedbox/activities"),
	hasStream("http://fedbox/objects"),
	hasPublicKey(defaultPublicKey),
	hasAuthEp("http://fedbox/oauth/authorize"),
	hasTokenEp("http://fedbox/oauth/token"),
)

var admin1 = actor(
	"https://fedbox/actors/1",
	hasType(vocab.PersonType),
	hasAttributedTo("https://fedbox"),
	hasAudience(vocab.PublicNS),
	hasGenerator("https://fedbox"),
	hasURL("https://fedbox/actors/1"),
	hasPreferredUsername("admin"),
	hasTag(tag0),
	hasAuthEp("https://fedbox/actors/1/oauth/authorize"),
	hasTokenEp("https://fedbox/actors/1/oauth/token"),
	hasSharedInbox("https://fedbox/inbox"),
)

var actor2 = actor(
	"https://fedbox/actors/2",
	hasType(vocab.PersonType),
	hasAttributedTo("https://fedbox"),
	hasAudience(vocab.PublicNS),
	hasContent("Generated actor"),
	hasSummary("Generated actor"),
	hasGenerator("https://fedbox"),
	hasURL("https://fedbox/actors/2"),
	hasLiked(),
	hasPreferredUsername("johndoe"),
	hasPublished("2019-08-11T13:14:47.000000000+02:00"),
	hasUpdated("2019-08-11T13:14:47.000000000+02:00"),
	hasName("Johnathan Doe"),
	hasSharedInbox("https://fedbox/inbox"),
	hasAuthEp("https://fedbox/oauth/authorize"),
	hasTokenEp("https://fedbox/oauth/token"),
	hasSharedInbox("https://fedbox/inbox"),
)

var tag0 = object(
	"https://fedbox/objects/0",
	hasName("#sysop"),
	hasAttributedTo("https://fedbox"),
	hasTo(vocab.PublicNS),
)

var object1 = object(
	"https://fedbox/objects/1",
	hasType(vocab.NoteType),
	hasContent("<p>Hello</p><code>FedBOX</code>!</p>\n"),
	hasMediaType("text/html"),
	hasPublished("2019-09-27T14:26:43.000000000Z"),
	hasUpdated("2019-09-27T14:26:43.000000000Z"),
	hasAttributedTo(admin1.ID),
	hasSource("Hello `FedBOX`!", "text/markdown"),
	hasTo("https://www.w3.org/ns/activitystreams#Public"),
)

func Test_Fetch(t *testing.T) {
	tests := []inOutTest{
		{
			name:   "service",
			input:  in(withURL("http://fedbox/")),
			output: out(hasCode(http.StatusOK), hasItem(service)),
		},
		{
			name:   "actors/1",
			input:  in(withURL("http://fedbox/actors/1")),
			output: out(hasCode(http.StatusOK), hasItem(admin1)),
		},
		{
			name:   "objects/0",
			input:  in(withURL("http://fedbox/objects/0")),
			output: out(hasCode(http.StatusOK), hasItem(tag0)),
		},
		{
			name:   "objects/1",
			input:  in(withURL("http://fedbox/objects/1")),
			output: out(hasCode(http.StatusOK), hasItem(object1)),
		},
		{
			name:   "actors/2",
			input:  in(withURL("http://fedbox/actors/2")),
			output: out(hasCode(http.StatusOK), hasItem(actor2)),
		},
	}

	ctx := context.Background()
	mocks, err := initMocks(ctx, t, suite{name: "fedbox"})
	if err != nil {
		t.Fatalf("unable to initialize containers: %s", err)
	}

	t.Cleanup(func() {
		mocks.cleanup(t)
	})

	for _, test := range tests {
		t.Run(test.name, test.run(ctx, mocks))
	}
}
