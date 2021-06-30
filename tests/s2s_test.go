// +build integration,s2s

package tests

import (
	"fmt"
	"net/http"
	"path"
	"testing"

	pub "github.com/go-ap/activitypub"
)

var activityCount = 0

func NewActivity(actor *testAccount) actMock {
	activityCount++
	id := baseURL + "/" + path.Join("activities", fmt.Sprintf("%d", activityCount))
	return actMock{
		Id:       id,
		ActorId:  actor.Id,
	}
}

var S2STests = testPairs{
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
					account: defaultC2SAccount(),
					urlFn:   InboxURL(defaultC2SAccount()),
					bodyFn:  loadMockJson("mocks/s2s/create-actor.json", NewActivity(defaultC2SAccount())),
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
							preferredUsername: "johndoe",
							name:              "Johnathan Doe",
						},
					},
				},
			},
		},
	},
}

func Test_S2SRequests(t *testing.T) {
	runTestSuite(t, S2STests)
}
