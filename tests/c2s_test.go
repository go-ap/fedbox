//go:build integration && c2s

package tests

import (
	"fmt"
	"net/http"
	"testing"

	vocab "github.com/go-ap/activitypub"
)

func CreateC2SObject(actor *testAccount, object vocab.Item) actC2SMock {
	return actC2SMock{
		ActorID: actor.ID,
		Object:  object,
	}
}

var ActorsCollectionTests = testPairs{
	{
		name:    "ActorsCollection",
		configs: c2sConfigs,
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
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
						id:        CollectionURL(ActorsURL(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 2,
						items: map[string]*objectVal{
							"e869bdca-dd5e-4de7-9c5d-37845eccc6a1": {
								id:      "http://127.0.0.1:9998/actors/e869bdca-dd5e-4de7-9c5d-37845eccc6a1",
								typ:     string(vocab.PersonType),
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
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
		},
		tests: []testPair{
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ActorsURL(), typeOf(vocab.PersonType)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActorsURL(), typeOf(vocab.PersonType), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
						items: map[string]*objectVal{
							"e869bdca-dd5e-4de7-9c5d-37845eccc6a1": {
								id:      "http://127.0.0.1:9998/actors/e869bdca-dd5e-4de7-9c5d-37845eccc6a1",
								typ:     string(vocab.PersonType),
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
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
		},
		tests: []testPair{
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ActorsURL(), typeOf(vocab.GroupType)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActorsURL(), typeOf(vocab.GroupType), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 0,
					},
				},
			},
		},
	},
	{
		name:    "ActorsCollectionTypeApplication",
		configs: c2sConfigs,
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
		},
		tests: []testPair{
			{
				mocks: []string{
					"mocks/c2s/actors/service.json",
				},
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ActorsURL(), typeOf(vocab.ApplicationType)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActorsURL(), typeOf(vocab.ApplicationType), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
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
			"mocks/c2s/actors/actor-johndoe.json",
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
						id:        CollectionURL(ActorsURL(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 21,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ActorsURL(), typeOf(vocab.ApplicationType)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActorsURL(), typeOf(vocab.ApplicationType), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 6,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ActorsURL(), typeOf(vocab.GroupType)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActorsURL(), typeOf(vocab.GroupType), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 5,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ActorsURL(), typeOf(vocab.PersonType)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActorsURL(), typeOf(vocab.PersonType), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 10,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ActorsURL(), typeOf(vocab.PersonType, vocab.GroupType)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActorsURL(), typeOf(vocab.PersonType, vocab.GroupType), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 15,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ActorsURL(), nameOf("element_a")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActorsURL(), nameOf("element_a"), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ActorsURL(), nameOf("element")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActorsURL(), nameOf("element"), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 9,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ActorsURL(), nameOf("element_a", "element_b")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActorsURL(), nameOf("element_a", "element_b"), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
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
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
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
						id:        CollectionURL(ActorsURL(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 3,
						items: map[string]*objectVal{
							"2": {
								id:  "http://127.0.0.1:9998/actors/2",
								typ: string(vocab.PersonType),
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
						id:        CollectionURL(ObjectsURL(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
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
					url: CollectionURL(ActivitiesURL()),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActivitiesURL(), firstPage()),
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
			"mocks/c2s/actors/actor-johndoe.json",
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
					url: CollectionURL(ActivitiesURL()),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActivitiesURL(), firstPage()),
						itemCount: 1,
					},
				},
			},
			{
				name: "Filter by actor IRI",
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ActivitiesURL(), actorIRI("http://127.0.0.1:9998/actors/2")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActivitiesURL(), actorIRI("http://127.0.0.1:9998/actors/2"), firstPage()),
						itemCount: 1,
					},
				},
			},
			{
				name: "Filter by fuzzy search on actor IRI",
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ActivitiesURL(), actorIRI("~/actors/2")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActivitiesURL(), actorIRI("~/actors/2"), firstPage()),
						itemCount: 1,
					},
				},
			},
			{
				name: "Filter by empty actor IRI",
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ActivitiesURL(), actorIRI("")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActivitiesURL(), actorIRI(""), firstPage()),
						itemCount: 0,
					},
				},
			},
			{
				name: "Filter by object IRI",
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ActivitiesURL(), objectIRI("http://127.0.0.1:9998/objects/1")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActivitiesURL(), objectIRI("http://127.0.0.1:9998/objects/1"), firstPage()),
						itemCount: 1,
					},
				},
			},
			{
				name: "Filter by fuzzy search on object IRI",
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ActivitiesURL(), objectIRI("~/objects/1")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActivitiesURL(), objectIRI("~/objects/1"), firstPage()),
						itemCount: 1,
					},
				},
			},
			{
				name: "Filter by empty object IRI",
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ActivitiesURL(), objectIRI("")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActivitiesURL(), objectIRI(""), firstPage()),
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
					url: CollectionURL(ActivitiesURL(), inReplyTo("!-")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActivitiesURL(), inReplyTo("!-"), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
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
					url: CollectionURL(ActivitiesURL(), inReplyTo("-")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActivitiesURL(), inReplyTo("-"), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
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
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
		},
		tests: []testPair{
			{
				req: testReq{
					account: &service,
					met:     http.MethodGet,
					url:     ObjectsURL(),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
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
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
			"mocks/c2s/objects/note-1.json",
			"mocks/c2s/objects/page-2.json",
			"mocks/c2s/objects/tombstone-3.json",
			"mocks/c2s/objects/place-4.json",
		},
		tests: []testPair{
			{
				req: testReq{
					account: &service,
					met:     http.MethodGet,
					url:     fmt.Sprintf("%s/objects", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 4,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ObjectsURL(), typeOf(vocab.TombstoneType)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), typeOf(vocab.TombstoneType), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ObjectsURL(), typeOf(vocab.PlaceType)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), typeOf(vocab.PlaceType), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ObjectsURL(), typeOf(vocab.NoteType)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), typeOf(vocab.NoteType), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ObjectsURL(), typeOf(vocab.PageType)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), typeOf(vocab.PageType), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ObjectsURL(), typeOf(vocab.PageType, vocab.PlaceType)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), typeOf(vocab.PageType, vocab.PlaceType), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 2,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ObjectsURL(), nameOf("You are here")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), nameOf("You are here"), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ObjectsURL(), nameOf("~You are")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), nameOf("~You are"), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ObjectsURL(), nameOf("~You", "~Humble")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), nameOf("~You", "~Humble"), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 2,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ObjectsURL(), urlOf("https://github.com/mariusor/mpris-scrobbler")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), urlOf("https://github.com/mariusor/mpris-scrobbler"), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ObjectsURL(), attrTo("http://127.0.0.1:9998/actors/2")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), attrTo("http://127.0.0.1:9998/actors/2"), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 3,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ObjectsURL(), inReplyTo("http://127.0.0.1:9998/objects/1")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), inReplyTo("http://127.0.0.1:9998/objects/1"), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 0,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ObjectsURL(), iri("!-")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), iri("!-"), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 4,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ObjectsURL(), iri("!"+apiURL)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), iri("!"+apiURL), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 4,
					},
				},
			},
			{
				name: "Filter by inReplyTo different than nil",
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ObjectsURL(), inReplyTo("!-")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), inReplyTo("!-"), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				name: "Filter by inReplyTo be nil",
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ObjectsURL(), inReplyTo("-")),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), inReplyTo("-"), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
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
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
		},
		tests: []testPair{
			{
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:      "http://127.0.0.1:9998/",
						typ:     string(vocab.ServiceType),
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

var CreateTests = testPairs{
	{
		name:    "CreateActor",
		configs: c2sConfigs,
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
		},
		tests: []testPair{
			{
				mocks: []string{
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
						typ: string(vocab.CreateType),
						act: &objectVal{
							typ:               string(vocab.PersonType),
							preferredUsername: "johndoe",
						},
						obj: &objectVal{
							typ:               string(vocab.PersonType),
							preferredUsername: "jennyjane",
							name:              "Jane Doe",
						},
					},
				},
			},
		},
	},
	{
		name:    "CreateArticle",
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
					bodyFn:  loadMockJson("mocks/c2s/activities/create-article.json", defaultC2SAccount()),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(vocab.CreateType),
						act: &objectVal{
							typ:               string(vocab.PersonType),
							preferredUsername: "johndoe",
						},
						obj: &objectVal{
							typ:     string(vocab.ArticleType),
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
						id:        CollectionURL(OutboxURL(defaultC2SAccount())(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: ActivitiesURL(),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActivitiesURL(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: CollectionURL(ObjectsURL()),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ObjectsURL(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: InboxURL(defaultC2SAccount()),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(InboxURL(defaultC2SAccount())(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: FollowingURL(defaultC2SAccount()),
				},
				res: testRes{code: http.StatusNotFound},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: FollowersURL(defaultC2SAccount()),
				},
				res: testRes{code: http.StatusNotFound},
			},
		},
	},
	{
		name:    "CreateActorAnonymously",
		configs: c2sConfigs,
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
		},
		tests: []testPair{
			{
				mocks: []string{
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
		// This builds a test for verifying how a FedBOX instance processes C2S activities that contain
		// recipients belonging to other federated services
		name:    "CreateNoteWithFederatedRecipient",
		configs: s2sConfigs,
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
		},
		tests: []testPair{
			{
				mocks: []string{
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
						typ: string(vocab.CreateType),
						act: &objectVal{
							id:                defaultC2SAccount().ID,
							typ:               string(vocab.PersonType),
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
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
		},
	},
	{
		name:    "CreateNoteWithMultipleInReplyTos",
		configs: c2sConfigs,
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
			"mocks/c2s/actors/application.json",
			"mocks/c2s/objects/page-2.json",
		},
		tests: []testPair{
			{
				req: testReq{
					met:     http.MethodPost,
					account: defaultC2SAccount(),
					urlFn:   OutboxURL(defaultC2SAccount()),
					bodyFn: loadMockJson(
						"mocks/c2s/activities/create-object.json",
						CreateC2SObject(defaultC2SAccount(), loadMockFromDisk("mocks/c2s/objects/note-1.json", nil)),
					),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(vocab.CreateType),
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
					urlFn: RepliesURL(loadMockFromDisk("mocks/c2s/objects/page-2.json", nil)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: RepliesURL(loadMockFromDisk("mocks/c2s/objects/note-1.json", nil)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 0,
					},
				},
			},
			{
				req: testReq{
					met:     http.MethodPost,
					account: defaultC2SAccount(),
					urlFn:   OutboxURL(defaultC2SAccount()),
					bodyFn: loadMockJson(
						"mocks/c2s/activities/create-object.json",
						CreateC2SObject(defaultC2SAccount(), loadMockFromDisk("mocks/c2s/objects/note-replyTo-1-and-2.json", nil)),
					),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(vocab.CreateType),
						obj: &objectVal{
							id:  loadMockFromDisk("mocks/c2s/objects/note-replyTo-1-and-2.json", nil).GetID().String(),
							typ: string(loadMockFromDisk("mocks/c2s/objects/note-replyTo-1-and-2.json", nil).GetType()),
						},
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: RepliesURL(loadMockFromDisk("mocks/c2s/objects/page-2.json", nil)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 2,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: RepliesURL(loadMockFromDisk("mocks/c2s/objects/note-1.json", nil)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: RepliesURL(loadMockFromDisk("mocks/c2s/objects/note-replyTo-1-and-2.json", nil)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 0,
					},
				},
			},
			{
				req: testReq{
					met:     http.MethodPost,
					account: defaultC2SAccount(),
					urlFn:   OutboxURL(defaultC2SAccount()),
					bodyFn: loadMockJson(
						"mocks/c2s/activities/create-object.json",
						CreateC2SObject(defaultC2SAccount(), loadMockFromDisk("mocks/c2s/objects/note-replyTo-1-2-5.json", nil)),
					),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(vocab.CreateType),
						obj: &objectVal{
							id:  loadMockFromDisk("mocks/c2s/objects/note-replyTo-1-2-5.json", nil).GetID().String(),
							typ: string(loadMockFromDisk("mocks/c2s/objects/note-replyTo-1-2-5.json", nil).GetType()),
						},
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: RepliesURL(loadMockFromDisk("mocks/c2s/objects/page-2.json", nil)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 3,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: RepliesURL(loadMockFromDisk("mocks/c2s/objects/note-1.json", nil)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 2,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: RepliesURL(loadMockFromDisk("mocks/c2s/objects/note-replyTo-1-and-2.json", nil)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: RepliesURL(loadMockFromDisk("mocks/c2s/objects/note-replyTo-1-2-5.json", nil)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 0,
					},
				},
			},
		},
	},
}

var UpdateTests = testPairs{
	{
		name:    "UpdateActor",
		configs: c2sConfigs,
		mocks: []string{
			"mocks/c2s/actors/service.json",
			"mocks/c2s/actors/actor-johndoe.json",
		},
		tests: []testPair{
			{
				mocks: []string{
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
						typ: string(vocab.UpdateType),
						act: &objectVal{
							id:  defaultC2SAccount().ID,
							typ: string(vocab.PersonType),
						},
						obj: &objectVal{
							id:                defaultC2SAccount().ID,
							name:              "Jane Doe",
							preferredUsername: "jennyjane",
							typ:               string(vocab.PersonType),
							inbox: &objectVal{
								id: fmt.Sprintf("%s/inbox", defaultC2SAccount().ID),
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
}

var DeleteTests = testPairs{
	{
		name:    "DeleteActor",
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
					bodyFn:  loadMockJson("mocks/c2s/activities/activity.json", actS2SMock{Type: "Delete", ActorID: defaultC2SAccount().ID, ObjectID: defaultC2SAccount().ID}),
				},
				res: testRes{
					code: http.StatusGone,
					val: &objectVal{
						typ: string(vocab.DeleteType),
						act: &objectVal{
							id:  defaultC2SAccount().ID,
							typ: string(vocab.TombstoneType),
						},
						obj: &objectVal{
							id:  defaultC2SAccount().ID,
							typ: string(vocab.TombstoneType),
						},
					},
				},
			},
		},
	},
}

var LikeTests = testPairs{
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
					bodyFn:  loadMockJson("mocks/c2s/activities/activity.json", &actS2SMock{Type: "Like", ActorID: defaultC2SAccount().ID, ObjectID: "http://127.0.0.1:9998/objects/41e7ec45-ff92-473a-b79d-974bf30a0aba"}),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(vocab.LikeType),
						act: &objectVal{
							typ:               string(vocab.PersonType),
							preferredUsername: "johndoe",
						},
						obj: &objectVal{
							typ: string(vocab.NoteType),
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
						id:        CollectionURL(OutboxURL(defaultC2SAccount())(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met: http.MethodGet,
					url: ActivitiesURL(),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(ActivitiesURL(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: InboxURL(defaultC2SAccount()),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(InboxURL(defaultC2SAccount())(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: LikedURL(defaultC2SAccount()),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        CollectionURL(LikedURL(defaultC2SAccount())(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: FollowingURL(defaultC2SAccount()),
				},
				res: testRes{code: http.StatusNotFound},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: FollowersURL(defaultC2SAccount()),
				},
				res: testRes{code: http.StatusNotFound},
			},
		},
	},
}

var FollowTests = testPairs{
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
					bodyFn:  loadMockJson("mocks/c2s/activities/activity.json", &actS2SMock{Type: "Follow", ActorID: defaultC2SAccount().ID, ObjectID: "http://127.0.0.1:9998/actors/58e877c7-067f-4842-960b-3896d76aa4ed"}),
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
						id:        CollectionURL(OutboxURL(defaultC2SAccount())(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: FollowingURL(defaultC2SAccount()),
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
						id:        CollectionURL(InboxURL(&extraAccount)(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
						itemCount: 1,
					},
				},
			},
		},
	},
}

var BlockTests = testPairs{
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
					bodyFn:  loadMockJson("mocks/c2s/activities/activity-private.json", &actS2SMock{Type: "Block", ActorID: defaultC2SAccount().ID, ObjectID: "http://127.0.0.1:9998/actors/58e877c7-067f-4842-960b-3896d76aa4ed"}),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(vocab.BlockType),
						act: &objectVal{
							typ:               string(vocab.PersonType),
							preferredUsername: "johndoe",
						},
						obj: &objectVal{
							typ:               string(vocab.PersonType),
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
						id:        CollectionURL(OutboxURL(defaultC2SAccount())(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
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
						id:        CollectionURL(InboxURL(&extraAccount)(), firstPage()),
						typ:       string(vocab.OrderedCollectionPageType),
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
							ActorID: defaultC2SAccount().ID,
						},
					),
				},
				res: testRes{
					code: http.StatusNoContent,
					val: &objectVal{
						typ: string(vocab.QuestionType),
						act: &objectVal{
							typ:               string(vocab.PersonType),
							preferredUsername: "johndoe",
						},
						obj: nil,
					},
				},
			},
		},
	},
	{
		name:    "QuestionWithOneOfAnswer",
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
						"mocks/c2s/activities/question-with-oneOf.json",
						&actS2SMock{
							ActorID: defaultC2SAccount().ID,
						},
					),
				},
				res: testRes{
					code: http.StatusNoContent,
					val: &objectVal{
						typ:  string(vocab.QuestionType),
						name: "Some question",
						act: &objectVal{
							typ:               string(vocab.PersonType),
							preferredUsername: "johndoe",
						},
						oneOf: []*objectVal{
							{name: "Answer 1"},
							{name: "Answer 2"},
						},
					},
				},
			},
		},
	},
	{
		name:    "QuestionWithAnyOfAnswer",
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
						"mocks/c2s/activities/question-with-anyOf.json",
						&actS2SMock{
							ActorID: defaultC2SAccount().ID,
						},
					),
				},
				res: testRes{
					code: http.StatusNoContent,
					val: &objectVal{
						typ:  string(vocab.QuestionType),
						name: "Some question",
						act: &objectVal{
							typ:               string(vocab.PersonType),
							preferredUsername: "johndoe",
						},
						anyOf: []*objectVal{
							{name: "Answer 1"},
							{name: "Answer 2"},
							{name: "Answer 3"},
						},
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

func Test_C2S_CreateRequests(t *testing.T) {
	runTestSuite(t, CreateTests)
}

func Test_C2S_UpdateRequests(t *testing.T) {
	runTestSuite(t, UpdateTests)
}

func Test_C2S_DeleteRequests(t *testing.T) {
	runTestSuite(t, DeleteTests)
}

func Test_C2S_LikeRequests(t *testing.T) {
	runTestSuite(t, LikeTests)
}

func Test_C2S_FollowRequests(t *testing.T) {
	runTestSuite(t, FollowTests)
}

func Test_C2S_BlockRequests(t *testing.T) {
	runTestSuite(t, BlockTests)
}

func Test_C2S_QuestionRequests(t *testing.T) {
	runTestSuite(t, QuestionTests)
}
