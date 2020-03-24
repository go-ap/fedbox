package tests

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	"net/http"
	"testing"
)

var C2STests = []testSuite{
	{
		name:  "SelfService",
		mocks: nil,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/service.json",
				},
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/", apiURL),
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
	{
		name:  "ActorsCollection",
		mocks: nil,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/service.json",
					"mocks/actor-johndoe.json",
				},
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/actors", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/actors", apiURL),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 3,
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
								liked: &objectVal{
									id: "http://127.0.0.1:9998/actors/e869bdca-dd5e-4de7-9c5d-37845eccc6a1/liked",
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
		name:  "ActorsCollectionTypePerson",
		mocks: nil,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/service.json",
					"mocks/actor-johndoe.json",
				},
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/actors?type=%s", apiURL, pub.PersonType),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/actors?type=%s", apiURL, pub.PersonType),
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
								liked: &objectVal{
									id: "http://127.0.0.1:9998/actors/e869bdca-dd5e-4de7-9c5d-37845eccc6a1/liked",
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
		name:  "ActorsCollectionTypeGroup",
		mocks: nil,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/service.json",
					"mocks/actor-johndoe.json",
				},
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/actors?type=%s", apiURL, pub.GroupType),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/actors?type=%s", apiURL, pub.GroupType),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 0,
					},
				},
			},
		},
	},
	{
		name:  "ActorsCollectionTypeApplication",
		mocks: nil,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/service.json",
				},
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/actors?type=%s", apiURL, pub.ApplicationType),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/actors?type=%s", apiURL, pub.ApplicationType),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 2,
					},
				},
			},
		},
	},
	{
		name:  "ActivitiesCollection",
		mocks: nil,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/service.json",
					"mocks/actor-johndoe.json",
				},
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/activities", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/activities", apiURL),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 0,
					},
				},
			},
		},
	},
	{
		name:  "ObjectsCollection",
		mocks: nil,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/service.json",
					"mocks/actor-johndoe.json",
				},
				req: testReq{
					met: http.MethodGet,
					url: fmt.Sprintf("%s/objects", apiURL),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/objects", apiURL),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 0,
					},
				},
			},
		},
	},
	{
		name:  "CreateActor",
		mocks: nil,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/service.json",
					"mocks/actor-johndoe.json",
					"mocks/application.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: &defaultTestAccount,
					urlFn:   func() string { return fmt.Sprintf("%s/outbox", *(&defaultTestAccount.Id)) },
					bodyFn:  loadMockJson("mocks/create-actor.json", &defaultTestAccount),
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
		name:  "AnonymousCreateActor",
		mocks: nil,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/service.json",
					"mocks/actor-johndoe.json",
					"mocks/application.json",
				},
				req: testReq{
					met:    http.MethodPost,
					urlFn:  func() string { return fmt.Sprintf("%s/outbox", *(&defaultTestAccount.Id)) },
					bodyFn: loadMockJson("mocks/create-actor.json", &defaultTestAccount),
				},
				res: testRes{
					code: http.StatusUnauthorized,
				},
			},
		},
	},
	{
		name:  "UpdateActor",
		mocks: nil,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/service.json",
					"mocks/actor-johndoe.json",
					"mocks/application.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: &defaultTestAccount,
					urlFn:   func() string { return fmt.Sprintf("%s/outbox", *(&defaultTestAccount.Id)) },
					bodyFn:  loadMockJson("mocks/update-actor.json", &defaultTestAccount),
				},
				res: testRes{
					code: http.StatusCreated,
					val: &objectVal{
						typ: string(pub.UpdateType),
						act: &objectVal{
							id:  *(&defaultTestAccount.Id),
							typ: string(pub.PersonType),
						},
						obj: &objectVal{
							id:                *(&defaultTestAccount.Id),
							name:              "Jane Doe",
							preferredUsername: "jennyjane",
							typ:               string(pub.PersonType),
							inbox: &objectVal{
								id: fmt.Sprintf("%s/inbox", *(&defaultTestAccount.Id)),
							},
							outbox: &objectVal{
								id: fmt.Sprintf("%s/outbox", *(&defaultTestAccount.Id)),
							},
						},
					},
				},
			},
		},
	},
	{
		name:  "DeleteActor",
		mocks: nil,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/service.json",
					"mocks/actor-johndoe.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: &defaultTestAccount,
					urlFn:   func() string { return fmt.Sprintf("%s/outbox", *(&defaultTestAccount.Id)) },
					bodyFn:  loadMockJson("mocks/activity.json", actMock{"Delete", *(&defaultTestAccount.Id), *(&defaultTestAccount.Id)}),
				},
				res: testRes{
					code: http.StatusGone,
					val: &objectVal{
						typ: string(pub.DeleteType),
						act: &objectVal{
							id:  *(&defaultTestAccount.Id),
							typ: string(pub.TombstoneType),
						},
						obj: &objectVal{
							id:  *(&defaultTestAccount.Id),
							typ: string(pub.TombstoneType),
						},
					},
				},
			},
		},
	},
	{
		name:  "CreateArticle",
		mocks: nil,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/service.json",
					"mocks/actor-johndoe.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: &defaultTestAccount,
					urlFn:   func() string { return fmt.Sprintf("%s/outbox", *(&defaultTestAccount.Id)) },
					bodyFn:  loadMockJson("mocks/create-article.json", &defaultTestAccount),
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
					urlFn: func() string { return fmt.Sprintf("%s/outbox", *(&defaultTestAccount.Id)) },
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/outbox", *(&defaultTestAccount.Id)),
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
					urlFn: func() string { return fmt.Sprintf("%s/following", *(&defaultTestAccount.Id)) },
				},
				res: testRes{
					code: http.StatusNotFound,
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: func() string { return fmt.Sprintf("%s/followers", *(&defaultTestAccount.Id)) },
				},
				res: testRes{
					code: http.StatusNotFound,
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: func() string { return fmt.Sprintf("%s/liked", *(&defaultTestAccount.Id)) },
				},
				res: testRes{
					code: http.StatusNotFound,
				},
			},
		},
	},
	{
		name: "LikeNote",
		mocks: []string{
			"mocks/service.json",
			"mocks/actor-johndoe.json",
			"mocks/note.json",
		},
		tests: []testPair{
			{
				req: testReq{
					met:     http.MethodPost,
					account: &defaultTestAccount,
					urlFn:   func() string { return fmt.Sprintf("%s/outbox", *(&defaultTestAccount.Id)) },
					bodyFn:  loadMockJson("mocks/activity.json", &actMock{Type: "Like", ActorId: *(&defaultTestAccount.Id), ObjectId: "http://127.0.0.1:9998/objects/41e7ec45-ff92-473a-b79d-974bf30a0aba"}),
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
					urlFn: func() string { return fmt.Sprintf("%s/outbox", *(&defaultTestAccount.Id)) },
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/outbox", *(&defaultTestAccount.Id)),
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
					url: fmt.Sprintf("%s/liked", *(&defaultTestAccount.Id)),
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/liked", *(&defaultTestAccount.Id)),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: func() string { return fmt.Sprintf("%s/following", *(&defaultTestAccount.Id)) },
				},
				res: testRes{
					code: http.StatusNotFound,
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: func() string { return fmt.Sprintf("%s/followers", *(&defaultTestAccount.Id)) },
				},
				res: testRes{
					code: http.StatusNotFound,
				},
			},
		},
	},
	{
		name: "FollowActor",
		mocks: []string{
			"mocks/service.json",
			"mocks/actor-johndoe.json",
			"mocks/actor-extra.json",
		},
		tests: []testPair{
			{
				req: testReq{
					met:     http.MethodPost,
					account: &defaultTestAccount,
					urlFn:   func() string { return fmt.Sprintf("%s/outbox", *(&defaultTestAccount.Id)) },
					bodyFn:  loadMockJson("mocks/activity.json", &actMock{Type: "Follow", ActorId: *(&defaultTestAccount.Id), ObjectId: "http://127.0.0.1:9998/actors/58e877c7-067f-4842-960b-3896d76aa4ed"}),
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
					urlFn: func() string { return fmt.Sprintf("%s/outbox", *(&defaultTestAccount.Id)) },
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/outbox", *(&defaultTestAccount.Id)),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: func() string { return fmt.Sprintf("%s/following", *(&defaultTestAccount.Id)) },
				},
				res: testRes{
					code: http.StatusNotFound,
				},
			},
			{
				req: testReq{
					met:   http.MethodGet,
					urlFn: func() string { return fmt.Sprintf("%s/inbox", *(&extraAccount.Id)) },
				},
				res: testRes{
					code: http.StatusOK,
					val: &objectVal{
						id:        fmt.Sprintf("%s/inbox", *(&extraAccount.Id)),
						typ:       string(pub.OrderedCollectionType),
						itemCount: 1,
					},
				},
			},
			//{
			//	req: testReq{
			//		met:     http.MethodPost,
			//		account: &defaultTestAccount,
			//		urlFn:   func() string { return fmt.Sprintf("%s/outbox", *(&extraAccount.Id)) },
			//		bodyFn:  loadMockJson("mocks/accept-follow.json", &actMock{ActorId: *(&extraAccount.Id), ObjectId: *(&lastActivity.id)}),
			//	},
			//	res: testRes{
			//		code: http.StatusCreated,
			//		val: &objectVal{
			//			typ: string(pub.AcceptType),
			//			act: &objectVal{
			//				typ:               string(pub.PersonType),
			//				preferredUsername: "extra",
			//			},
			//			obj: &objectVal{
			//				typ: string(pub.FollowType),
			//			},
			//		},
			//	},
			//},
		},
	},
}

func Test_C2SRequests(t *testing.T) {
	runTestSuite(t, C2STests)
}
