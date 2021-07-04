// +build integration,s2s

package tests

import (
	"fmt"
	"net/http"
	"path"
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/internal/log"
)

var S2SConfig = config.Options{
	Env:         env.TEST,
	Host:        "127.0.2.1:9999",
	Listen:      "127.0.2.1:9999",
	BaseURL:     "http://127.0.2.1:9999/",
	LogLevel:    log.DebugLevel,
	StoragePath: ".cache",
	Storage:     storageType(),
}

var s2sConfigs = []config.Options{
	C2SConfig,
	S2SConfig,
}

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
		configs: s2sConfigs,
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
