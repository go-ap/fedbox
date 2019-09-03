package tests

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
	"net/http"
	"testing"
)

var C2STests = testPairs{
	"SelfService": {
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
					id:      "http://127.0.0.1:9998",
					typ:     string(as.ServiceType),
					summary: "Generic ActivityPub service",
					inbox: &objectVal{
						id: "http://127.0.0.1:9998/inbox",
					},
					outbox: &objectVal{
						id: "http://127.0.0.1:9998/outbox",
					},
					name: "self",
				},
			},
		},
	},
	"ActorsCollection": {
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
					typ:       string(as.OrderedCollectionType),
					itemCount: 2,
					items: map[string]*objectVal{
						"e869bdca-dd5e-4de7-9c5d-37845eccc6a1": {
							id:      "http://127.0.0.1:9998/actors/e869bdca-dd5e-4de7-9c5d-37845eccc6a1",
							typ:     string(as.PersonType),
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
	"ActorsCollectionTypePerson": {
		{
			mocks: []string{
				"mocks/service.json",
				"mocks/actor-johndoe.json",
			},
			req: testReq{
				met: http.MethodGet,
				url: fmt.Sprintf("%s/actors?type=%s", apiURL, as.PersonType),
			},
			res: testRes{
				code: http.StatusOK,
				val: &objectVal{
					id:        fmt.Sprintf("%s/actors?type=%s", apiURL, as.PersonType),
					typ:       string(as.OrderedCollectionType),
					itemCount: 1,
					items: map[string]*objectVal{
						"e869bdca-dd5e-4de7-9c5d-37845eccc6a1": {
							id:      "http://127.0.0.1:9998/actors/e869bdca-dd5e-4de7-9c5d-37845eccc6a1",
							typ:     string(as.PersonType),
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
	"ActorsCollectionTypeGroup": {
		{
			mocks: []string{
				"mocks/service.json",
				"mocks/actor-johndoe.json",
			},
			req: testReq{
				met: http.MethodGet,
				url: fmt.Sprintf("%s/actors?type=%s", apiURL, as.GroupType),
			},
			res: testRes{
				code: http.StatusOK,
				val: &objectVal{
					id:        fmt.Sprintf("%s/actors?type=%s", apiURL, as.GroupType),
					typ:       string(as.OrderedCollectionType),
					itemCount: 0,
				},
			},
		},
	},
	"ActorsCollectionTypeApplication": {
		{
			mocks: []string{
				"mocks/service.json",
			},
			req: testReq{
				met: http.MethodGet,
				url: fmt.Sprintf("%s/actors?type=%s", apiURL, as.ApplicationType),
			},
			res: testRes{
				code: http.StatusOK,
				val: &objectVal{
					id:        fmt.Sprintf("%s/actors?type=%s", apiURL, as.ApplicationType),
					typ:       string(as.OrderedCollectionType),
					itemCount: 1,
				},
			},
		},
	},
	"ActivitiesCollection": {
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
					typ:       string(as.OrderedCollectionType),
					itemCount: 0,
				},
			},
		},
	},
	"ObjectsCollection": {
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
					typ:       string(as.OrderedCollectionType),
					itemCount: 0,
				},
			},
		},
	},
	"CreateActor": {
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
					typ: string(as.CreateType),
					act: &objectVal{
						typ:               string(as.PersonType),
						preferredUsername: "johndoe",
					},
					obj: &objectVal{
						typ:               string(as.PersonType),
						preferredUsername: "jennyjane",
						name:              "Jane Doe",
					},
				},
			},
		},
	},
	"UpdateActor": {
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
				code: http.StatusOK,
				val: &objectVal{
					typ: string(as.UpdateType),
					act: &objectVal{
						id:                *(&defaultTestAccount.Id),
						typ:               string(as.PersonType),
						preferredUsername: "johndoe",
					},
					obj: &objectVal{
						id:                *(&defaultTestAccount.Id),
						name:              "Jane Doe",
						preferredUsername: "jennyjane",
						typ:               string(as.PersonType),
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
	"DeleteActor": {
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
				bodyFn:  loadMockJson("mocks/delete-actor.json", &defaultTestAccount),
			},
			res: testRes{
				code: http.StatusGone,
				val: &objectVal{
					typ: string(as.DeleteType),
					act: &objectVal{
						id:  *(&defaultTestAccount.Id),
						typ: string(as.TombstoneType),
					},
					obj: &objectVal{
						id:  *(&defaultTestAccount.Id),
						typ: string(as.TombstoneType),
					},
				},
			},
		},
	},
	"CreateArticle": {
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
				bodyFn:  loadMockJson("mocks/create-article.json", &defaultTestAccount),
			},
			res: testRes{
				code: http.StatusCreated,
				val: &objectVal{
					typ: string(as.CreateType),
					act: &objectVal{
						typ:               string(as.PersonType),
						preferredUsername: "johndoe",
					},
					obj: &objectVal{
						typ:     string(as.ArticleType),
						content: "\\u003cp\\u003eHello world\\u003c/p\\u003e", // FIXME(marius): This sucks balls,
					},
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
					typ:       string(as.OrderedCollectionType),
					itemCount: 1,
				},
			},
		},
	},
}

func Test_C2SRequests(t *testing.T) {
	testSuite(t, C2STests)
}
