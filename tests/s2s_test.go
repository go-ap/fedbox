//go:build integration && s2s

package tests

import (
	"fmt"
	"net/http"
	"path"
	"testing"

	vocab "github.com/go-ap/activitypub"
)

func CreateS2SObject(actor *testAccount, object any) actS2SMock {
	id := "http://" + s2shost + "/" + path.Join("activities", fmt.Sprintf("%d", activityCount))
	var objectId string
	switch ob := object.(type) {
	case string:
		objectId = ob
	case *testAccount:
		objectId = ob.ID
	case vocab.Item:
		objectId = string(ob.GetID())
	}
	return actS2SMock{
		Id:       id,
		ActorID:  actor.ID,
		ObjectID: objectId,
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
						"mocks/s2s/activities/create-object.json",
						CreateS2SObject(defaultS2SAccount(), defaultS2SAccount()),
					),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(vocab.CreateType),
						act: &objectVal{
							id:                defaultS2SAccount().ID,
							typ:               string(vocab.PersonType),
							preferredUsername: "lou",
						},
						obj: &objectVal{
							id:                defaultS2SAccount().ID,
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
						"mocks/s2s/activities/create-object.json",
						CreateS2SObject(defaultS2SAccount(), loadMockFromDisk("mocks/s2s/objects/note-1.json", nil)),
					),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(vocab.CreateType),
						act: &objectVal{
							id:                defaultS2SAccount().ID,
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
						"mocks/s2s/activities/accept-follow.json",
						generatedAccept,
					),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(vocab.AcceptType),
						act: &objectVal{
							id:                defaultC2SAccount().ID,
							typ:               string(vocab.PersonType),
							preferredUsername: defaultC2SAccount().Handle,
						},
						obj: &objectVal{
							id:  generatedAccept.ObjectID,
							typ: string(vocab.FollowType),
							obj: &objectVal{
								id:  defaultC2SAccount().ID,
								typ: string(vocab.PersonType),
							},
							act: &objectVal{
								id: defaultS2SAccount().ID,
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
						id:        CollectionURL(FollowingURL(defaultS2SAccount())(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
						items:     map[string]*objectVal{},
					},
				},
			},
		},
	},
}

var followID = fmt.Sprintf("http://%s/follow/1", host)
var UndoTests = testPairs{
	{
		name:    "Follow Actor then Undo",
		configs: s2sConfigs,
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
			"mocks/c2s/actors/actor-extra.json",
			// S2S objects need to be present
			"mocks/s2s/actors/actor-666.json",
		},
		tests: []testPair{
			{
				name: "Follow Actor",
				req: testReq{
					met:     http.MethodPost,
					account: defaultC2SAccount(),
					urlFn:   OutboxURL(defaultC2SAccount()),
					bodyFn: loadMockJson(
						"mocks/c2s/activities/activity.json",
						&actS2SMock{
							Id:       followID,
							Type:     "Follow",
							ActorID:  defaultC2SAccount().ID,
							ObjectID: defaultS2SAccount().ID,
						},
					),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						id:  followID,
						typ: string(vocab.FollowType),
						act: &objectVal{
							id:                defaultC2SAccount().ID,
							typ:               string(vocab.PersonType),
							preferredUsername: "johndoe",
						},
						obj: &objectVal{
							id:                defaultS2SAccount().ID,
							typ:               string(vocab.PersonType),
							preferredUsername: "lou",
						},
					},
				},
			},
			{
				name: "Test follower doesn't exist in following collection",
				req: testReq{
					met:   http.MethodGet,
					urlFn: FollowingURL(defaultC2SAccount()),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(FollowingURL(defaultC2SAccount())(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 0,
					},
				},
			},
			{
				name: "Test Follow exists in outbox",
				req: testReq{
					met:   http.MethodGet,
					urlFn: OutboxURL(defaultC2SAccount()),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(OutboxURL(&defaultTestAccountC2S)(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
						items: map[string]*objectVal{
							followID: {
								id:  followID,
								typ: "Follow",
							},
						},
					},
				},
			},
			{
				name: "Test Follow exists in remote inbox",
				req: testReq{
					met:   http.MethodGet,
					urlFn: InboxURL(defaultS2SAccount()),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(InboxURL(defaultS2SAccount())(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
						items: map[string]*objectVal{
							followID: {
								id:  followID,
								typ: "Follow",
							},
						},
					},
				},
			},
			{
				// S2S Accept Follow
				name: "Accept the Follow",
				req: testReq{
					met:     http.MethodPost,
					account: defaultS2SAccount(),
					urlFn:   OutboxURL(defaultS2SAccount()),
					bodyFn: loadMockJson(
						"mocks/c2s/activities/activity.json",
						&actS2SMock{
							Type:     "Accept",
							ActorID:  defaultS2SAccount().ID,
							ObjectID: followID,
						},
					),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(vocab.AcceptType),
						act: &objectVal{
							typ: string(vocab.FollowType),
						},
					},
				},
			},
			{
				name: "Test Follower exists",
				req: testReq{
					met:   http.MethodGet,
					urlFn: FollowersURL(defaultS2SAccount()),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(FollowersURL(defaultS2SAccount())(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
						items: map[string]*objectVal{
							defaultC2SAccount().ID: {
								id: defaultC2SAccount().ID,
							},
						},
					},
				},
			},
			{
				name: "Undo the Follow",
				req: testReq{
					met:     http.MethodPost,
					account: defaultC2SAccount(),
					urlFn:   OutboxURL(defaultC2SAccount()),
					bodyFn: loadMockJson(
						"mocks/c2s/activities/activity.json",
						&actS2SMock{
							Type:     "Undo",
							ActorID:  defaultC2SAccount().ID,
							ObjectID: followID,
						},
					),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(vocab.FollowType),
						act: &objectVal{
							typ:               string(vocab.PersonType),
							preferredUsername: "johndoe",
						},
						obj: &objectVal{
							typ:               string(vocab.PersonType),
							preferredUsername: "lou",
						},
					},
				},
			},
		},
	},
}

func Test_S2SReceiveRequests(t *testing.T) {
	runTestSuites(t, S2SReceiveTests)
}

func Test_S2S_UndoRequests(t *testing.T) {
	runTestSuites(t, UndoTests)
}
