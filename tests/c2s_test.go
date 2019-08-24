package tests

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
	"net/http"
	"testing"
)

var C2STests = testPairs{
	"ActorsCollection": {
		{
			req: testReq{
				met: http.MethodGet,
				url: fmt.Sprintf("%s/actors", apiURL),
			},
			res: testRes{
				code: http.StatusOK,
				val: &objectVal{
					id:        fmt.Sprintf("%s/actors", apiURL),
					typ:       string(as.OrderedCollectionType),
					itemCount: 0,
				},
			},
		},
	},
	"ActivitiesCollection": {
		{
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
	"ServiceActor": {
		{
			req: testReq{
				met: http.MethodGet,
				url: selfAccount.Id,
			},
			res: testRes{
				code: http.StatusOK,
				val: &objectVal{
					id:   selfAccount.Id,
					typ:  string(as.ServiceType),
					name: selfAccount.Handle,
					audience: []string{
						as.PublicNS.String(),
					},
					inbox: &objectVal{
						id: fmt.Sprintf("%s/inbox", baseURL),
					},
					outbox: &objectVal{
						id: fmt.Sprintf("%s/outbox", baseURL),
					},
				},
			},
		},
	},
	"CreateActor": {
		{
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
						content: "<p>Hello world</p>",
					},
				},
			},
		},
	},
}

func Test_C2SRequests(t *testing.T) {
	testSuite(t, C2STests)
}
