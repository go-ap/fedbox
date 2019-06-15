package tests

import (
	"bytes"
	"fmt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/activitypub"
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
					itemCount: 1,
					items: map[string]*objectVal{
						selfAccount.Hash: {
							id: selfAccount.id,
						},
					},
					first: &objectVal{
						id:        fmt.Sprintf("%s/actors?maxItems=%d&page=1", apiURL, activitypub.MaxItems),
						typ:       string(as.OrderedCollectionPageType),
						itemCount: 1,
						items: map[string]*objectVal{
							selfAccount.Hash: {
								id: selfAccount.id,
							},
						},
					},
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
				},
			},
		},
	},
	"CreateActor": {
		{
			req: testReq{
				met: http.MethodPost,
				url: fmt.Sprintf("%s/inbox", apiURL),
				body: loadMockJson("mocks/create-actor.json", selfAccount.id),
			},
			res: testRes{
				code: http.StatusInternalServerError,
			},
		},
	},
	"UpdateActor": {
		{
			req: testReq{
				met: http.MethodPost,
				url: fmt.Sprintf("%s/inbox", apiURL),
				body: loadMockJson("mocks/update-actor.json", selfAccount.id),
			},
			res: testRes{
				code: http.StatusInternalServerError,
			},
		},
	},
	"DeleteActor": {
		{
			req: testReq{
				met: http.MethodPost,
				url: fmt.Sprintf("%s/inbox", apiURL),
				body: loadMockJson("mocks/delete-actor.json", selfAccount.id),
			},
			res: testRes{
				code: http.StatusInternalServerError,
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
