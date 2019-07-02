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
						"https://www.w3.org/ns/activitystreams#Public",
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
				met:  http.MethodPost,
				url:  fmt.Sprintf("%s/outbox", apiURL),
				body: loadMockJson("mocks/create-actor.json", selfAccount.id),
			},
			res: testRes{
				code: http.StatusCreated,
				val: &objectVal{
					typ: string(as.CreateType),
					act: &objectVal{
						id:  selfAccount.id,
						typ: string(as.ServiceType),
					},
				},
			},
		},
	},
	"UpdateActor": {
		{
			req: testReq{
				met:  http.MethodPost,
				url:  fmt.Sprintf("%s/outbox", apiURL),
				body: loadMockJson("mocks/update-actor.json", selfAccount.id, selfAccount.id, baseURL),
			},
			res: testRes{
				code: http.StatusCreated,
				val: &objectVal{
					typ: string(as.UpdateType),
					act: &objectVal{
						id: selfAccount.id,
					},
					obj: &objectVal{
						id:   selfAccount.id,
						name: "Jane Doe",
						preferredUsername: "jennyjane",
						typ:  string(as.ServiceType),
						inbox: &objectVal{
							id: fmt.Sprintf("%s/inbox", apiURL),
						},
						outbox: &objectVal{
							id: fmt.Sprintf("%s/outbox", apiURL),
						},
					},
				},
			},
		},
	},
	"DeleteActor": {
		{
			req: testReq{
				met:  http.MethodPost,
				url:  fmt.Sprintf("%s/outbox", apiURL),
				body: loadMockJson("mocks/delete-actor.json", selfAccount.id, selfAccount.id),
			},
			res: testRes{
				code: http.StatusGone,
				val: &objectVal{
					typ: string(as.DeleteType),
					act: &objectVal{
						id:  selfAccount.id,
						typ: string(as.TombstoneType),
					},
					obj: &objectVal{
						id:  selfAccount.id,
						typ: string(as.TombstoneType),
					},
				},
			},
		},
	},
	"CreateArticle": {
		{
			req: testReq{
				met:  http.MethodPost,
				url:  fmt.Sprintf("%s/outbox", apiURL),
				body: loadMockJson("mocks/create-article.json", selfAccount.id, selfAccount.id),
			},
			res: testRes{
				code: http.StatusCreated,
				val: &objectVal{
					typ: string(as.CreateType),
					act: &objectVal{
						id:  selfAccount.id,
						typ: string(as.ServiceType),
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

func loadMockJson(path string, params ...interface{}) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}

	st, err := f.Stat()
	if err != nil {
		return ""
	}

	data := make([]byte, st.Size())
	io.ReadFull(f, data)
	data = bytes.Trim(data, "\x00")

	return fmt.Sprintf(string(data), params...)
}

func Test_C2SRequests(t *testing.T) {
	testSuite(t, C2STests)
}
