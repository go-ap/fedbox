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

// S2SReceiveTests builds tests for verifying a FedBOX instance receives and processes correctly
// activities coming from federated requests.
var S2SReceiveTests = testPairs{
	{
		name:    "CreateActor",
		configs: s2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/service.json",
					"mocks/actor-johndoe.json",
					"mocks/application.json",
					// S2S objects need to be present
					"mocks/s2s/activities/create-actor-666.json",
					"mocks/s2s/actors/actor-666.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: defaultS2SAccount(),
					urlFn:   InboxURL(defaultC2SAccount()),
					bodyFn:  loadMockJson(
						"mocks/s2s/create-object.json",
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
					// s2s entities that need to exist
					"mocks/s2s/actors/actor-666.json",
					"mocks/s2s/objects/note-1.json",
					"mocks/s2s/activities/create-note-1.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: defaultS2SAccount(),
					urlFn:   InboxURL(defaultC2SAccount()),
					bodyFn:  loadMockJson(
						"mocks/s2s/create-object.json",
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

// S2SSendTests builds tests for verifying how a FedBOX instance processes C2S activities that contain
// recipients belonging to other federated services
var S2SSendTests = testPairs{
}

func Test_S2SReceiveRequests(t *testing.T) {
	runTestSuite(t, S2SReceiveTests)
}

func Test_S2SSendRequests(t *testing.T) {
	runTestSuite(t, S2SSendTests)
}
