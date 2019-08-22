package tests

import (
	"bytes"
	"fmt"
	as "github.com/go-ap/activitystreams"
	"io"
	"net/http"
	"os"
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
				url: selfAccount.id,
			},
			res: testRes{
				code: http.StatusOK,
				val: &objectVal{
					id:   selfAccount.id,
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
				url:     fmt.Sprintf("%s/outbox", apiURL),
				bodyFn:  loadMockJson("mocks/create-actor.json", &defaultTestAccount.id),
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
				url:     fmt.Sprintf("%s/outbox", apiURL),
				bodyFn:  loadMockJson("mocks/update-actor.json", &defaultTestAccount.id, &defaultTestAccount.id, &defaultTestAccount.id),
			},
			res: testRes{
				code: http.StatusOK,
				val: &objectVal{
					typ: string(as.UpdateType),
					act: &objectVal{
						id:                *(&defaultTestAccount.id),
						typ:               string(as.PersonType),
						preferredUsername: "johndoe",
					},
					obj: &objectVal{
						id:                *(&defaultTestAccount.id),
						name:              "Jane Doe",
						preferredUsername: "jennyjane",
						typ:               string(as.PersonType),
						inbox: &objectVal{
							id: fmt.Sprintf("%s/inbox", *(&defaultTestAccount.id)),
						},
						outbox: &objectVal{
							id: fmt.Sprintf("%s/outbox", *(&defaultTestAccount.id)),
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
				url:     fmt.Sprintf("%s/outbox", apiURL),
				bodyFn:  loadMockJson("mocks/delete-actor.json", &defaultTestAccount.id, &selfAccount.id),
			},
			res: testRes{
				code: http.StatusGone,
				val: &objectVal{
					typ: string(as.DeleteType),
					act: &objectVal{
						id:  *(&defaultTestAccount.id),
						typ: string(as.TombstoneType),
					},
					obj: &objectVal{
						id:  *(&defaultTestAccount.id),
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
				url:     fmt.Sprintf("%s/outbox", apiURL),
				bodyFn:  loadMockJson("mocks/create-article.json", &defaultTestAccount.id, &selfAccount.id),
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

func loadMockJson(path string, params ...*string) func() string {
	f, err := os.Open(path)
	if err != nil {
		return func() string { return "" }
	}

	st, err := f.Stat()
	if err != nil {
		return func() string { return "" }
	}

	data := make([]byte, st.Size())
	io.ReadFull(f, data)
	data = bytes.Trim(data, "\x00")

	return func() string {
		par := make([]interface{}, len(params))
		for k, v := range params {
			par[k] = *v
		}
		return fmt.Sprintf(string(data), par...)
	}
}

func Test_C2SRequests(t *testing.T) {
	testSuite(t, C2STests)
}
