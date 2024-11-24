//go:build integration && s2s

package tests

import (
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
)

func mitraUser() *testAccount {
	m := testAccount{
		Id:         fmt.Sprintf("http://%s/actors/mitraUser", s2shost),
		Handle:     "mitraUser",
		PublicKey:  publicKeyFrom(rsaKey),
		PrivateKey: rsaKey,
	}
	return &m
}

var MitraTests = testPairs{
	{
		name: "Follow",
		configs: []config.Options{
			C2SConfig,
			{
				Env:         env.TEST,
				Host:        "127.0.2.1:9999",
				Listen:      "127.0.2.1:9999",
				BaseURL:     "http://127.0.2.1:9999/",
				LogLevel:    lw.NoLevel,
				StoragePath: filepath.Join(storagePath(), "127.0.2.1:9999"),
				Storage:     storageType(),
			},
		},
		tests: []testPair{
			{
				mocks: []string{
					"mocks/c2s/actors/service.json",
					"mocks/c2s/actors/actor-johndoe.json",
					"mocks/c2s/actors/application.json",
					"mocks/s2s/actors/mitra-user.json",
				},
				req: testReq{
					met:     http.MethodPost,
					account: mitraUser(),
					urlFn:   InboxURL(defaultC2SAccount()),
					bodyFn: loadMockJson(
						"mocks/s2s/activities/follow-mitra.json",
						CreateS2SObject(defaultS2SAccount(), mitraUser()),
					),
				},
				res: testRes{
					code: http.StatusCreated,
				},
			},
		},
	},
}

func Test_S2SMitra(t *testing.T) {
	runTestSuite(t, MitraTests)
}
