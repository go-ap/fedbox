//go:build integration && c2s
// +build integration,c2s

package tests

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	pub "github.com/go-ap/activitypub"
)

func ActorsURL() string {
	return ServiceActorsURL(&service)
}

func ObjectsURL() string {
	return ServiceObjectsURL(&service)
}

func CreateC2SObject(actor *testAccount, object pub.Item) actC2SMock {
	return actC2SMock{
		ActorId: actor.Id,
		Object:  object,
	}
}

var ActorsCollectionTests = testPairs{
	{
		name:    "ActorsCollection",
		configs: c2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/c2s/actors/service.json",
					"mocks/c2s/actors/actor-johndoe.json",
				},
				req: testReq{
					met: http.MethodGet,
					url: ActorsURL(),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        ActorsURL(),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 2,
						items: map[string]*objectVal{
							"e869bdca-dd5e-4de7-9c5d-37845eccc6a1": {
								id:      "http://127.0.0.1:9998/actors/e869bdca-dd5e-4de7-9c5d-37845eccc6a1",
								typ:     string(pub.PersonType),
								summary: "Generated actor",
								content: "Generated actor",
								url:     "http://127.0.0.1:9998/actors/e869bdca-dd5e-4de7-9c5d-37845eccc6a1",
								inbox: &objectVal{
									id: "http://127.0.0.1:9998/actors/e869bdca-dd5e-4de7-9c5d-37845eccc6a1/inbox",
								},
								outbox: &objectVal{
									id: "http://127.0.0.1:9998/actors/e869bdca-dd5e-4de7-9c5d-37845eccc6a1/outbox",
								},
								preferredUsername: "johndoe",
								name:              "Johnathan Doe",
							},
						},
					},
				},
			},
		},
	},
	{
		name:    "ActorsCollectionTypePerson",
		configs: c2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/c2s/actors/service.json",
					"mocks/c2s/actors/actor-johndoe.json",
				},
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s?type=%s", ActorsURL(), pub.PersonType),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s?type=%s", ActorsURL(), pub.PersonType),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
						items: map[string]*objectVal{
							"e869bdca-dd5e-4de7-9c5d-37845eccc6a1": {
								id:      "http://127.0.0.1:9998/actors/e869bdca-dd5e-4de7-9c5d-37845eccc6a1",
								typ:     string(pub.PersonType),
								summary: "Generated actor",
								content: "Generated actor",
								url:     "http://127.0.0.1:9998/actors/e869bdca-dd5e-4de7-9c5d-37845eccc6a1",
								inbox: &objectVal{
									id: "http://127.0.0.1:9998/actors/e869bdca-dd5e-4de7-9c5d-37845eccc6a1/inbox",
								},
								outbox: &objectVal{
									id: "http://127.0.0.1:9998/actors/e869bdca-dd5e-4de7-9c5d-37845eccc6a1/outbox",
								},
								preferredUsername: "johndoe",
								name:              "Johnathan Doe",
							},
						},
					},
				},
			},
		},
	},
	{
		name:    "ActorsCollectionTypeGroup",
		configs: c2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/c2s/actors/service.json",
					"mocks/c2s/actors/actor-johndoe.json",
				},
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s?type=%s", ActorsURL(), pub.GroupType),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s?type=%s", ActorsURL(), pub.GroupType),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 0,
					},
				},
			},
		},
	},
	{
		name:    "ActorsCollectionTypeApplication",
		configs: c2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/c2s/actors/service.json",
				},
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s?type=%s", ActorsURL(), pub.ApplicationType),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s?type=%s", ActorsURL(), pub.ApplicationType),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
		},
	},
	{
		name:    "A lot of actors",
		configs: c2sConfigs,
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-admin.json",
			"mocks/c2s/actors/actor-element_a.json",
			"mocks/c2s/actors/actor-element_b.json",
			"mocks/c2s/actors/actor-element_c.json",
			"mocks/c2s/actors/actor-element_d.json",
			"mocks/c2s/actors/actor-element_e.json",
			"mocks/c2s/actors/actor-element_f.json",
			"mocks/c2s/actors/actor-element_g.json",
			"mocks/c2s/actors/actor-element_h.json",
			"mocks/c2s/actors/actor-element_i.json",
			"mocks/c2s/actors/application-11.json",
			"mocks/c2s/actors/application-12.json",
			"mocks/c2s/actors/application-13.json",
			"mocks/c2s/actors/application-14.json",
			"mocks/c2s/actors/application-15.json",
			"mocks/c2s/actors/group-16.json",
			"mocks/c2s/actors/group-17.json",
			"mocks/c2s/actors/group-18.json",
			"mocks/c2s/actors/group-19.json",
			"mocks/c2s/actors/group-20.json",
		},
		tests: []testPair{
			{
				req: testReq{
					met: http.MethodGet,
					url: ActorsURL(),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        ActorsURL(),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 21,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s?type=%s", ActorsURL(), pub.ApplicationType),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s?type=%s", ActorsURL(), pub.ApplicationType),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 6,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s?type=%s", ActorsURL(), pub.GroupType),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s?type=%s", ActorsURL(), pub.GroupType),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 5,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s?type=%s", ActorsURL(), pub.PersonType),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s?type=%s", ActorsURL(), pub.PersonType),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 10,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s?type=%s&type=%s", ActorsURL(), pub.PersonType, pub.GroupType),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s?type=%s&type=%s", ActorsURL(), pub.PersonType, pub.GroupType),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 15,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s?name=%s", ActorsURL(), "element_a"),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s?name=%s", ActorsURL(), "element_a"),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s?name=~%s", ActorsURL(), "element"),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s?name=~%s", ActorsURL(), "element"),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 9,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s?name=%s&name=%s", ActorsURL(), "element_a", "element_b"),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s?name=%s&name=%s", ActorsURL(), "element_a", "element_b"),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 2,
					},
				},
			},
		},
	},
	{
		name:    "Actor with tag",
		configs: c2sConfigs,
		mocks: []string{
			"mocks/c2s/actors/actor-element_a.json",
			"mocks/c2s/objects/tag-mod.json",
		},
		tests: []testPair{
			{
				req: testReq{
					met: http.MethodGet,
					url: ActorsURL(),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        ActorsURL(),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 2,
						items: map[string]*objectVal{
							"2": {
								id:  "http://127.0.0.1:9998/actors/2",
								typ: string(pub.PersonType),
								tag: []*objectVal{
									{
										id:   "http://127.0.0.1:9998/objects/t1",
										typ:  "",
										name: "#mod",
									},
								},
							},
						},
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: ObjectsURL(),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        ObjectsURL(),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
						items: map[string]*objectVal{
							"t1": {
								id:   "http://127.0.0.1:9998/objects/t1",
								typ:  "",
								name: "#mod",
							},
						},
					},
				},
			},
		},
	},
}

var ActivitiesCollectionTests = testPairs{
	{
		name:    "ActivitiesCollection",
		configs: c2sConfigs,
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
		},
		tests: []testPair{
			{
				name: "empty-activities-collection",
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/activities", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						itemCount: 0,
					},
				},
			},
		},
	},
	{
		name:    "Create activities filtering",
		configs: c2sConfigs,
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-element_a.json",
			"mocks/c2s/objects/note-1.json",
			"mocks/c2s/activities/create-1.json",
		},
		tests: []testPair{
			{
				name:  "no filter",
				mocks: []string{},
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/activities", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						itemCount: 1,
					},
				},
			},
			{
				name: "Filter by actor IRI",
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/activities?actor.iri=http://127.0.0.1:9998/actors/2", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						itemCount: 1,
					},
				},
			},
			{
				name: "Filter by fuzzy search on actor IRI",
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/activities?actor.iri=~/actors/2", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						itemCount: 1,
					},
				},
			},
			{
				name: "Filter by empty actor IRI",
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/activities?actor.iri=", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						itemCount: 0,
					},
				},
			},
			{
				name: "Filter by object IRI",
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/activities?object.iri=http://127.0.0.1:9998/objects/1", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						itemCount: 1,
					},
				},
			},
			{
				name: "Filter by fuzzy search on object IRI",
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/activities?object.iri=~/objects/1", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						itemCount: 1,
					},
				},
			},
			{
				name: "Filter by empty object IRI",
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/activities?object.iri=", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						itemCount: 0,
					},
				},
			},
			{
				name: "Filter by object.inReplyTo different than nil",
				mocks: []string{
					"mocks/c2s/objects/page-2.json",
					"mocks/c2s/activities/create-2.json",
				},
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/activities?object.inReplyTo=!-", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/activities?object.inReplyTo=%%21-", apiURL),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				name: "Filter by object.inReplyTo be nil",
				mocks: []string{
					"mocks/c2s/objects/page-2.json",
					"mocks/c2s/activities/create-2.json",
				},
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/activities?object.inReplyTo=-", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/activities?object.inReplyTo=-", apiURL),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
		},
	},
}

var ObjectsCollectionTests = testPairs{
	{
		name:    "ObjectsCollection",
		configs: c2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/c2s/actors/service.json",
					"mocks/c2s/actors/actor-johndoe.json",
				},
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						typ:       string(pub.OrderedCollectionType),
						itemCount: 0,
					},
				},
			},
		},
	},
	{
		name:    "A lot of objects",
		configs: c2sConfigs,
		mocks: []string{
			"mocks/c2s/objects/note-1.json",
			"mocks/c2s/objects/page-2.json",
			"mocks/c2s/objects/tombstone-3.json",
			"mocks/c2s/objects/place-4.json",
		},
		tests: []testPair{
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects", apiURL),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 4,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects?type=%s", apiURL, pub.TombstoneType),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects?type=%s", apiURL, pub.TombstoneType),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects?type=%s", apiURL, pub.PlaceType),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects?type=%s", apiURL, pub.PlaceType),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects?type=%s", apiURL, pub.NoteType),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects?type=%s", apiURL, pub.NoteType),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects?type=%s", apiURL, pub.PageType),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects?type=%s", apiURL, pub.PageType),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects?type=%s&type=%s", apiURL, pub.PageType, pub.PlaceType),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects?type=%s&type=%s", apiURL, pub.PageType, pub.PlaceType),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 2,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects?name=%s", apiURL, url.QueryEscape("You are here")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects?name=%s", apiURL, url.QueryEscape("You are here")),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects?name=~%s", apiURL, url.QueryEscape("You are")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects?name=~%s", apiURL, url.QueryEscape("You are")),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects?name=~%s&name=~%s", apiURL, "You", "Humble"),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects?name=~%s&name=~%s", apiURL, "You", "Humble"),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 2,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects?url=%s", apiURL, url.QueryEscape("https://github.com/mariusor/mpris-scrobbler")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects?url=%s", apiURL, url.QueryEscape("https://github.com/mariusor/mpris-scrobbler")),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects?attributedTo=%s", apiURL, url.QueryEscape("http://127.0.0.1:9998/actors/2")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects?attributedTo=%s", apiURL, url.QueryEscape("http://127.0.0.1:9998/actors/2")),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 3,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects?inReplyTo=%s", apiURL, url.QueryEscape("http://127.0.0.1:9998/objects/1")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects?inReplyTo=%s", apiURL, url.QueryEscape("http://127.0.0.1:9998/objects/1")),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 0,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects?iri=!-", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects?iri=%%21-", apiURL),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 4,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects?iri=!", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects?iri=%%21", apiURL),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 4,
					},
				},
			},
			{
				name: "Filter by inReplyTo different than nil",
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects?inReplyTo=!-", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects?inReplyTo=%%21-", apiURL),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				name: "Filter by inReplyTo be nil",
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects?inReplyTo=-", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects?inReplyTo=-", apiURL),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 3,
					},
				},
			},
		},
	},
}

var SingleItemLoadTests = testPairs{
	{
		name:    "SelfService",
		configs: c2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/c2s/actors/service.json",
				},
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:      "http://127.0.0.1:9998/",
						typ:     string(pub.ServiceType),
						summary: "Generic ActivityPub service",
						inbox: &objectVal{
							id: "http://127.0.0.1:9998/inbox",
						},
						name: "self",
					},
				},
			},
		},
	},
}

var C2STests = testPairs{
	{
		name:    "CreateActor",
		configs: c2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/c2s/actors/service.json",
					"mocks/c2s/actors/actor-johndoe.json",
					"mocks/c2s/actors/application.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: defaultC2SAccount(),
					urlFn:   OutboxURL(defaultC2SAccount()),
					bodyFn:  loadMockJson("mocks/c2s/activities/create-actor.json", defaultC2SAccount()),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(pub.CreateType),
						act: &objectVal{
							typ:               string(pub.PersonType),
							preferredUsername: "johndoe",
						},
						obj: &objectVal{
							typ:               string(pub.PersonType),
							preferredUsername: "jennyjane",
							name:              "Jane Doe",
						},
					},
				},
			},
		},
	},
	{
		name:    "CreateActorAnonymously",
		configs: c2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/c2s/actors/service.json",
					"mocks/c2s/actors/actor-johndoe.json",
					"mocks/c2s/actors/application.json",
				},
				req: testReq{
					met:    http.MethodPost,
					urlFn:  OutboxURL(defaultC2SAccount()),
					bodyFn: loadMockJson("mocks/c2s/activities/create-actor.json", defaultC2SAccount()),
				},
				res: testRes{
					code: http.StatusUnauthorized,
				},
			},
		},
	},
	{
		name:    "UpdateActor",
		configs: c2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/c2s/actors/service.json",
					"mocks/c2s/actors/actor-johndoe.json",
					"mocks/c2s/actors/application.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: defaultC2SAccount(),
					urlFn:   OutboxURL(defaultC2SAccount()),
					bodyFn:  loadMockJson("mocks/c2s/activities/update-actor.json", defaultC2SAccount()),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(pub.UpdateType),
						act: &objectVal{
							id:  defaultC2SAccount().Id,
							typ: string(pub.PersonType),
						},
						obj: &objectVal{
							id:                defaultC2SAccount().Id,
							name:              "Jane Doe",
							preferredUsername: "jennyjane",
							typ:               string(pub.PersonType),
							inbox: &objectVal{
								id: fmt.Sprintf("%s/inbox", defaultC2SAccount().Id),
							},
							outbox: &objectVal{
								id: OutboxURL(defaultC2SAccount())(),
							},
						},
					},
				},
			},
		},
	},
	{
		name:    "DeleteActor",
		configs: c2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/c2s/actors/service.json",
					"mocks/c2s/actors/actor-johndoe.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: defaultC2SAccount(),
					urlFn:   OutboxURL(defaultC2SAccount()),
					bodyFn:  loadMockJson("mocks/c2s/activities/activity.json", actS2SMock{Type: "Delete", ActorId: defaultC2SAccount().Id, ObjectId: defaultC2SAccount().Id}),
				},
				res: testRes{
					code: http.StatusGone,
					val: &objectVal{
						typ: string(pub.DeleteType),
						act: &objectVal{
							id:  defaultC2SAccount().Id,
							typ: string(pub.TombstoneType),
						},
						obj: &objectVal{
							id:  defaultC2SAccount().Id,
							typ: string(pub.TombstoneType),
						},
					},
				},
			},
		},
	},
	{
		name:    "CreateArticle",
		configs: c2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/c2s/actors/service.json",
					"mocks/c2s/actors/actor-johndoe.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: defaultC2SAccount(),
					urlFn:   OutboxURL(defaultC2SAccount()),
					bodyFn:  loadMockJson("mocks/c2s/activities/create-article.json", defaultC2SAccount()),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(pub.CreateType),
						act: &objectVal{
							typ:               string(pub.PersonType),
							preferredUsername: "johndoe",
						},
						obj: &objectVal{
							typ:     string(pub.ArticleType),
							content: "<p>Hello world</p>",
						},
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: OutboxURL(defaultC2SAccount()),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        OutboxURL(defaultC2SAccount())(),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/activities", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/activities", apiURL),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects", apiURL),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/inbox", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/inbox", apiURL),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: func() string { return fmt.Sprintf("%s/following", defaultC2SAccount().Id) },
				},
				res: testRes{code: http.StatusNotFound},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: func() string { return fmt.Sprintf("%s/followers", defaultC2SAccount().Id) },
				},
				res: testRes{code: http.StatusNotFound},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: func() string { return fmt.Sprintf("%s/liked", defaultC2SAccount().Id) },
				},
				res: testRes{code: http.StatusNotFound},
			},
		},
	},
	{
		name:    "LikeNote",
		configs: c2sConfigs,
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
			"mocks/c2s/objects/note.json",
		},
		tests: []testPair{
			{
				req: testReq{
					met:     http.MethodPost,
					account: defaultC2SAccount(),
					urlFn:   OutboxURL(defaultC2SAccount()),
					bodyFn:  loadMockJson("mocks/c2s/activities/activity.json", &actS2SMock{Type: "Like", ActorId: defaultC2SAccount().Id, ObjectId: "http://127.0.0.1:9998/objects/41e7ec45-ff92-473a-b79d-974bf30a0aba"}),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(pub.LikeType),
						act: &objectVal{
							typ:               string(pub.PersonType),
							preferredUsername: "johndoe",
						},
						obj: &objectVal{
							typ: string(pub.NoteType),
						},
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: OutboxURL(defaultC2SAccount()),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        OutboxURL(defaultC2SAccount())(),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/activities", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/activities", apiURL),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/inbox", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/inbox", apiURL),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/liked", defaultC2SAccount().Id),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/liked", defaultC2SAccount().Id),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: func() string { return fmt.Sprintf("%s/following", defaultC2SAccount().Id) },
				},
				res: testRes{code: http.StatusNotFound},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: func() string { return fmt.Sprintf("%s/followers", defaultC2SAccount().Id) },
				},
				res: testRes{code: http.StatusNotFound},
			},
		},
	},
	{
		name:    "FollowActor",
		configs: c2sConfigs,
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
			"mocks/c2s/actors/actor-extra.json",
		},
		tests: []testPair{
			{
				req: testReq{
					met:     http.MethodPost,
					account: defaultC2SAccount(),
					urlFn:   OutboxURL(defaultC2SAccount()),
					bodyFn:  loadMockJson("mocks/c2s/activities/activity.json", &actS2SMock{Type: "Follow", ActorId: defaultC2SAccount().Id, ObjectId: "http://127.0.0.1:9998/actors/58e877c7-067f-4842-960b-3896d76aa4ed"}),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(pub.FollowType),
						act: &objectVal{
							typ:               string(pub.PersonType),
							preferredUsername: "johndoe",
						},
						obj: &objectVal{
							typ:               string(pub.PersonType),
							preferredUsername: "extra",
						},
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: OutboxURL(defaultC2SAccount()),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        OutboxURL(defaultC2SAccount())(),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: func() string { return fmt.Sprintf("%s/following", defaultC2SAccount().Id) },
				},
				res: testRes{code: http.StatusNotFound},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: InboxURL(&extraAccount),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        InboxURL(&extraAccount)(),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
		},
	},
	{
		name:    "BlockActor",
		configs: c2sConfigs,
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
			"mocks/c2s/actors/actor-extra.json",
		},
		tests: []testPair{
			{
				req: testReq{
					met:     http.MethodPost,
					account: defaultC2SAccount(),
					urlFn:   OutboxURL(defaultC2SAccount()),
					bodyFn:  loadMockJson("mocks/c2s/activities/activity-private.json", &actS2SMock{Type: "Block", ActorId: defaultC2SAccount().Id, ObjectId: "http://127.0.0.1:9998/actors/58e877c7-067f-4842-960b-3896d76aa4ed"}),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(pub.BlockType),
						act: &objectVal{
							typ:               string(pub.PersonType),
							preferredUsername: "johndoe",
						},
						obj: &objectVal{
							typ:               string(pub.PersonType),
							preferredUsername: "extra",
						},
					},
				},
			},
			{
				req: testReq{
					met:     http.MethodGet,
					account: defaultC2SAccount(),
					urlFn:   OutboxURL(defaultC2SAccount()),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        OutboxURL(defaultC2SAccount())(),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met:     http.MethodGet,
					account: defaultC2SAccount(),
					urlFn:   InboxURL(&extraAccount),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        InboxURL(&extraAccount)(),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 0,
					},
				},
			},
		},
	},
}

var QuestionTests = testPairs{
	{
		name:    "SimpleQuestion",
		configs: c2sConfigs,
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
		},
		tests: []testPair{
			{
				req: testReq{
					met:     http.MethodPost,
					account: defaultC2SAccount(),
					urlFn:   OutboxURL(defaultC2SAccount()),
					bodyFn: loadMockJson(
						"mocks/c2s/activities/question.json",
						&actS2SMock{
							ActorId: defaultC2SAccount().Id,
						},
					),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(pub.QuestionType),
						act: &objectVal{
							typ:               string(pub.PersonType),
							preferredUsername: "johndoe",
						},
						obj: nil,
					},
				},
			},
		},
	},
}

// S2SSendTests builds tests for verifying how a FedBOX instance processes C2S activities that contain
// recipients belonging to other federated services
var S2SSendTests = testPairs{
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
				},
				req: testReq{
					met:     http.MethodPost,
					account: defaultC2SAccount(),
					urlFn:   OutboxURL(defaultC2SAccount()),
					bodyFn: loadMockJson(
						"mocks/c2s/activities/create-object-with-federated-cc.json",
						CreateC2SObject(defaultC2SAccount(), loadMockFromDisk("mocks/c2s/objects/note-1.json", nil)),
					),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(pub.CreateType),
						act: &objectVal{
							id:                defaultC2SAccount().Id,
							typ:               string(pub.PersonType),
							preferredUsername: "johndoe",
							name:              "Johnathan Doe",
						},
						obj: &objectVal{
							id:  loadMockFromDisk("mocks/c2s/objects/note-1.json", nil).GetID().String(),
							typ: string(loadMockFromDisk("mocks/c2s/objects/note-1.json", nil).GetType()),
						},
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: InboxURL(defaultS2SAccount()),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
		},
	},
}

func Test_SingleItemLoad(t *testing.T) {
	runTestSuite(t, SingleItemLoadTests)
}

func Test_ObjectsCollections(t *testing.T) {
	runTestSuite(t, ObjectsCollectionTests)
}

func Test_ActivitiesCollection(t *testing.T) {
	runTestSuite(t, ActivitiesCollectionTests)
}

func Test_ActorsCollection(t *testing.T) {
	runTestSuite(t, ActorsCollectionTests)
}

func Test_C2SRequests(t *testing.T) {
	runTestSuite(t, C2STests)
}

func Test_S2SSendRequests(t *testing.T) {
	runTestSuite(t, S2SSendTests)
}

func Test_QuestionRequests(t *testing.T) {
	runTestSuite(t, QuestionTests)
}
