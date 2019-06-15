package tests

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/activitypub"
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
					id:  fmt.Sprintf("%s/actors", apiURL),
					typ: string(as.OrderedCollectionType),
					itemCount: 1,
					items: map[string]*objectVal{
						selfAccount.Hash: {
							id: selfAccount.id,
						},
					},
					first: &objectVal{
						id: fmt.Sprintf("%s/actors?maxItems=%d&page=1", apiURL, activitypub.MaxItems),
						typ: string(as.OrderedCollectionPageType),
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
					id:  fmt.Sprintf("%s/activities", apiURL),
					typ: string(as.OrderedCollectionType),
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
					id:  fmt.Sprintf("%s/objects", apiURL),
					typ: string(as.OrderedCollectionType),
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
					id:  selfAccount.id,
					typ: string(as.ServiceType),
					name: selfAccount.Handle,
					audience: []string{
						"https://www.w3.org/ns/activitystreams#Public",
					},
				},
			},
		},
	},
}

func Test_C2SRequests(t *testing.T) {
	testSuite(t, C2STests)
}
