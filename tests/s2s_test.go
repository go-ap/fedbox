// +build integration,s2s

package tests

import (
	"fmt"
	"net/http"
	"path"
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/fedbox/internal/config"
)

var s2sConfigs = []config.Options{
	C2SConfig,
	S2SConfig,
}

var activityCount = 0

func CreateS2SActor(actor *testAccount, object *testAccount) actMock {
	activityCount++
	id := baseURL + "/" + path.Join("activities", fmt.Sprintf("%d", activityCount))
	return actMock{
		Id:       id,
		ActorId:  actor.Id,
		ObjectId:  object.Id,
	}
}

var S2STests = testPairs{
	{
		name:  "CreateActor",
		configs: s2sConfigs,
		tests: []testPair{
			{
				mocks: []string{
					"mocks/service.json",
					"mocks/actor-johndoe.json",
					"mocks/application.json",
					"mocks/s2s/actors/actor-666.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: defaultC2SAccount(),
					urlFn:   InboxURL(defaultC2SAccount()),
					bodyFn:  loadMockJson("mocks/s2s/create-actor.json", CreateS2SActor(defaultS2SAccount(), defaultS2SAccount())),
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
