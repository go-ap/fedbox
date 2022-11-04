//go:build integration && s2s

package tests

import (
	"fmt"
	"net/http"
	"path"
	"testing"

	vocab "github.com/go-ap/activitypub"
)

func CreateS2SObject(actor *testAccount, object interface{}) actS2SMock {
	id := "http://" + s2shost + "/" + path.Join("activities", fmt.Sprintf("%d", activityCount))
	var objectId string
	switch ob := object.(type) {
	case string:
		objectId = ob
	case *testAccount:
		objectId = ob.Id
	case vocab.Item:
		objectId = string(ob.GetID())
	}
	return actS2SMock{
		Id:       id,
		ActorId:  actor.Id,
		ObjectId: objectId,
	}
}

// Generate test Accept with C2S account as actor and the Follow request as object
var generatedAccept = CreateS2SObject(
	defaultC2SAccount(),
	loadMockFromDisk("mocks/s2s/activities/follow-666-johndoe.json", nil),
)

// S2SReceiveTests builds tests for verifying a FedBOX instance receives and processes correctly
// activities coming from federated requests.
var S2SReceiveTests = testPairs{
	{
		name:    "CreateActor",
		configs: s2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/c2s/actors/service.json",
					"mocks/c2s/actors/actor-johndoe.json",
					"mocks/c2s/actors/application.json",
					// S2S objects need to be present
					"mocks/s2s/activities/create-actor-666.json",
					"mocks/s2s/actors/actor-666.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: defaultS2SAccount(),
					urlFn:   InboxURL(defaultC2SAccount()),
					bodyFn: loadMockJson(
						"mocks/s2s/create-object.json",
						CreateS2SObject(defaultS2SAccount(), defaultS2SAccount()),
					),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(vocab.CreateType),
						act: &objectVal{
							id:                defaultS2SAccount().Id,
							typ:               string(vocab.PersonType),
							preferredUsername: "lou",
						},
						obj: &objectVal{
							id:                defaultS2SAccount().Id,
							typ:               string(vocab.PersonType),
							preferredUsername: "lou",
							name:              "Loucien Cypher",
						},
					},
				},
			},
		},
	},
	{
		name:    "CreateNote",
		configs: s2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/c2s/actors/service.json",
					"mocks/c2s/actors/actor-johndoe.json",
					"mocks/c2s/actors/application.json",
					// s2s entities that need to exist
					"mocks/s2s/actors/actor-666.json",
					"mocks/s2s/objects/note-1.json",
					"mocks/s2s/activities/create-note-1.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: defaultS2SAccount(),
					urlFn:   InboxURL(defaultC2SAccount()),
					bodyFn: loadMockJson(
						"mocks/s2s/create-object.json",
						CreateS2SObject(defaultS2SAccount(), loadMockFromDisk("mocks/s2s/objects/note-1.json", nil)),
					),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(vocab.CreateType),
						act: &objectVal{
							id:                defaultS2SAccount().Id,
							typ:               string(vocab.PersonType),
							preferredUsername: "lou",
							name:              "Loucien Cypher",
						},
						obj: &objectVal{
							id:  loadMockFromDisk("mocks/s2s/objects/note-1.json", nil).GetID().String(),
							typ: string(loadMockFromDisk("mocks/s2s/objects/note-1.json", nil).GetType()),
						},
					},
				},
			},
		},
	},
	{
		name:    "AcceptFollow",
		configs: s2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/c2s/actors/service.json",
					"mocks/c2s/actors/actor-johndoe.json",
					"mocks/c2s/actors/application.json",
					// s2s entities that need to exist
					"mocks/s2s/actors/actor-666.json",
					"mocks/s2s/activities/follow-666-johndoe.json",
					// This is used for validation
					"mocks/s2s/activities/accept-follow-666-johndoe.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: defaultC2SAccount(),
					urlFn:   InboxURL(defaultS2SAccount()),
					bodyFn: loadMockJson(
						"mocks/s2s/accept-follow.json",
						generatedAccept,
					),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(vocab.AcceptType),
						act: &objectVal{
							id:                defaultC2SAccount().Id,
							typ:               string(vocab.PersonType),
							preferredUsername: defaultC2SAccount().Handle,
						},
						obj: &objectVal{
							id:  generatedAccept.ObjectId,
							typ: string(vocab.FollowType),
							obj: &objectVal{
								id:  defaultC2SAccount().Id,
								typ: string(vocab.PersonType),
							},
							act: &objectVal{
								id: defaultS2SAccount().Id,
							},
						},
					},
				},
			},
			{
				// The followers collection doesn't really exist because we didn't mock it
				req: testReq{
					met:   http.MethodGet,
					urlFn: FollowersURL(defaultC2SAccount()),
				},
				res: testRes{
					code: http.StatusNotFound,
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: FollowingURL(defaultS2SAccount()),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						typ:       string(vocab.OrderedCollectionType),
						itemCount: 1,
						items:     map[string]*objectVal{},
					},
				},
			},
		},
	},
}

func Test_S2SReceiveRequests(t *testing.T) {
	runTestSuite(t, S2SReceiveTests)
}
