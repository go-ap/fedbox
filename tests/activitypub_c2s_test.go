package tests

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/activitypub"
	"net/http"
	"testing"
)

var C2STests = testPairs{
	"Load": {
		{
			req: testReq{
				met: http.MethodGet,
				url: fmt.Sprintf("%s/actors", apiURL),
			},
			res: testRes{
				code: http.StatusOK,
				val: objectVal{
					id:      fmt.Sprintf("%s/actors", apiURL),
					typ:     string(as.OrderedCollectionType),
					first:  &objectVal{
						id: fmt.Sprintf("%s/actors?maxItems=%d&page=1", apiURL, activitypub.MaxItems),
					},
				},
			},
		},
	},
}

func Test_C2SRequests(t *testing.T) {
	testSuite(t, C2STests)
}
