package tests

import (
	"fmt"
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
				code: http.StatusInternalServerError,
				//val: objectVal{
				//	id:      fmt.Sprintf("%s/actors", apiURL),
				//	typ:     string(as.OrderedCollectionType),
				//	first:  &objectVal{
				//		id: fmt.Sprintf("%s/actors?maxItems=%d&page=1", apiURL, activitypub.MaxItems),
				//	},
				//},
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
				code: http.StatusInternalServerError,
				//val: objectVal{
				//	id:      fmt.Sprintf("%s/activities", apiURL),
				//	typ:     string(as.OrderedCollectionType),
				//	first:  &objectVal{
				//		id: fmt.Sprintf("%s/activities?maxItems=%d&page=1", apiURL, activitypub.MaxItems),
				//	},
				//},
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
				code: http.StatusInternalServerError,
				//val: objectVal{
				//	id:      fmt.Sprintf("%s/objects", apiURL),
				//	typ:     string(as.OrderedCollectionType),
				//	first:  &objectVal{
				//		id: fmt.Sprintf("%s/objects?maxItems=%d&page=1", apiURL, activitypub.MaxItems),
				//	},
				//},
			},
		},
	},
}

func Test_C2SRequests(t *testing.T) {
	testSuite(t, C2STests)
}
