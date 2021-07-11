// +build integration,s2s

package tests

import (
	"fmt"
	"net/http"
	"path"
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/fedbox/internal/config"
)

var s2sConfigs = []config.Options{
	C2SConfig,
	S2SConfig,
}

var activityCount = 0

func CreateS2SObject(actor *testAccount, object interface{}) actMock {
	id := "http://" + s2shost + "/" + path.Join("activities", fmt.Sprintf("%d", activityCount))
	var objectId string
	switch ob := object.(type) {
	case string:
		objectId = ob
	case *testAccount:
		objectId = ob.Id
	case pub.Item:
		objectId = string(ob.GetID())
	}
	return actMock{
		Id: id,
		ActorId:  actor.Id,
		ObjectId: objectId,
	}
}

var S2STests = testPairs{
	{
		name:    "CreateActor",
		configs: s2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/service.json",
					"mocks/actor-johndoe.json",
					"mocks/application.json",
					"mocks/s2s/actors/actor-666.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: defaultS2SAccount(),
					urlFn:   InboxURL(defaultC2SAccount()),
					bodyFn:  loadMockJson(
						"mocks/s2s/activities/create-1.json",
						CreateS2SObject(defaultS2SAccount(), defaultS2SAccount()),
					),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(pub.CreateType),
						act: &objectVal{
							id:                defaultS2SAccount().Id,
							typ:               string(pub.PersonType),
							preferredUsername: "lou",
						},
						obj: &objectVal{
							id:                defaultS2SAccount().Id,
							typ:               string(pub.PersonType),
							preferredUsername: "lou",
							name:              "Loucien Cypher",
						},
					},
				},
			},
		},
	},
	{
		name:    "CreateArticle",
		configs: s2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/service.json",
					"mocks/actor-johndoe.json",
					"mocks/application.json",
					"mocks/s2s/actors/actor-666.json",
					"mocks/s2s/objects/note-1.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: defaultS2SAccount(),
					urlFn:   InboxURL(defaultC2SAccount()),
					bodyFn:  loadMockJson(
						"mocks/s2s/activities/create-1.json",
						CreateS2SObject(defaultS2SAccount(), loadMockFromDisk("mocks/s2s/objects/note-1.json", nil)),
					),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(pub.CreateType),
						act: &objectVal{
							id:                defaultS2SAccount().Id,
							typ:               string(pub.PersonType),
							preferredUsername: "lou",
							name:              "Loucien Cypher",
						},
						obj: &objectVal{
							id:                loadMockFromDisk("mocks/s2s/objects/note-1.json", nil).GetID().String(),
							typ:               string(loadMockFromDisk("mocks/s2s/objects/note-1.json", nil).GetType()),
						},
					},
				},
			},
		},
	},
}

func Test_S2SRequests(t *testing.T) {
	runTestSuite(t, S2STests)
}
