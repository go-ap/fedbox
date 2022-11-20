//go:build integration

package tests

import (
	"bytes"
	"context"
	"crypto"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"runtime/debug"
	"sort"
	"strings"
	"testing"
	"time"

	"git.sr.ht/~mariusor/lw"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/fedbox"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/httpsig"
	"github.com/go-ap/jsonld"
)

// UserAgent value that the client uses when performing requests
var UserAgent = "test-go-http-client"
var HeaderAccept = `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`

var activityCount = 0
var fedboxApp *fedbox.FedBOX

var C2SConfig = config.Options{
	Env:         env.TEST,
	Host:        "127.0.0.1:9998",
	Listen:      "127.0.0.1:9998",
	BaseURL:     "http://127.0.0.1:9998/",
	LogLevel:    lw.NoLevel,
	StoragePath: storagePath(),
	Storage:     storageType(),
}

var S2SConfig = config.Options{
	Env:         env.TEST,
	Host:        "127.0.2.1:9999",
	Listen:      "127.0.2.1:9999",
	BaseURL:     "http://127.0.2.1:9999/",
	LogLevel:    lw.NoLevel,
	StoragePath: storagePath(),
	Storage:     storageType(),
}

var c2sConfigs = []config.Options{
	C2SConfig,
}

var s2sConfigs = []config.Options{
	C2SConfig,
	S2SConfig,
}

type actS2SMock struct {
	Id       string
	Type     string
	ActorId  string
	ObjectId string
}

type actC2SMock struct {
	Type    string
	ActorId string
	Object  vocab.Item
}

type testSuite struct {
	name    string
	configs []config.Options
	mocks   []string
	tests   []testPair
}

type testPairs []testSuite

type testAccount struct {
	Id         string `json:"id"`
	Handle     string `json:"handle"`
	Hash       string `json:"hash"`
	PublicKey  crypto.PublicKey
	PrivateKey crypto.PrivateKey
	AuthToken  string
}

type testReq struct {
	met      string
	url      string
	urlFn    func() string
	headers  http.Header
	account  *testAccount
	clientID string
	bodyFn   func() (string, error)
	body     string
}

func (t testPair) label() string {
	if t.name != "" {
		return t.name
	}
	b := t.req.url
	if b == "" {
		b = t.req.urlFn()
	}
	return fmt.Sprintf("[%s]%s", t.req.met, b)
}

type testRes struct {
	code int
	val  *objectVal
	body string
}

type testPair struct {
	name  string
	mocks []string
	req   testReq
	act   *objectVal
	res   testRes
}

type objectVal struct {
	id                string
	typ               string
	name              string
	preferredUsername string
	summary           string
	url               string
	content           string
	mediaType         string
	author            string
	partOf            *objectVal
	inbox             *objectVal
	outbox            *objectVal
	following         *objectVal
	followers         *objectVal
	liked             *objectVal
	act               *objectVal
	obj               *objectVal
	oneOf             []*objectVal
	anyOf             []*objectVal
	tag               []*objectVal
	itemCount         int64
	first             *objectVal
	next              *objectVal
	last              *objectVal
	current           *objectVal
	items             map[string]*objectVal
	audience          []string
}

var storagePath = func() string {
	path, err := ioutil.TempDir(".cache", "*")
	if err != nil {
		panic(err)
	}
	return path
}

var storageType = func() config.StorageType {
	envStorage := config.Getval("STORAGE", "all")
	if len(envStorage) > 0 {
		return config.StorageType(envStorage)
	}
	return config.DefaultStorage
}

func defaultC2SAccount() *testAccount {
	return &defaultTestAccountC2S
}

func defaultS2SAccount() *testAccount {
	return &defaultTestAccountS2S
}

func InboxURL(account *testAccount) func() string {
	return func() string {
		return fmt.Sprintf("%s/inbox", account.Id)
	}
}

func FollowersURL(account *testAccount) func() string {
	return func() string {
		return fmt.Sprintf("%s/followers", account.Id)
	}
}

func FollowingURL(account *testAccount) func() string {
	return func() string {
		return fmt.Sprintf("%s/following", account.Id)
	}
}

func OutboxURL(account *testAccount) func() string {
	return func() string {
		return fmt.Sprintf("%s/outbox", account.Id)
	}
}

func ServiceObjectsURL(service *testAccount) string {
	return fmt.Sprintf("%s/objects", service.Id)
}

func ServiceActorsURL(service *testAccount) string {
	return fmt.Sprintf("%s/actors", service.Id)
}

func RepliesURL(ob vocab.Item) func() string {
	return func() string {
		return fmt.Sprintf("%s/replies", ob.GetID())
	}
}

const (
	serviceHash = "d3ab037c-0f15-4c09-b635-3d6e201c11aa"

	testAppHash = "23767f95-8ea0-40ba-a6ef-b67284e1cdb1"

	testActorHash   = "e869bdca-dd5e-4de7-9c5d-37845eccc6a1"
	testActorHandle = "johndoe"
	testActorPem    = ""

	extraActorHash   = "58e877c7-067f-4842-960b-3896d76aa4ed"
	extraActorHandle = "extra"

	host    = "127.0.0.1:9998"
	s2shost = "127.0.2.1:9999"
)

var (
	apiURL = "http://127.0.0.1:9998"

	authCallbackURL = fmt.Sprintf("%s/auth/local/callback", apiURL)
	inboxURL        = InboxURL(&service)
	outboxURL       = OutboxURL(&service)
	baseURL         = service.Id

	key = loadPrivateKeyFromDisk("mocks/keys/johndoe.prv")

	meta interface{} = nil

	service = testAccount{Id: apiURL}

	defaultTestAccountC2S = testAccount{
		Id:         fmt.Sprintf("http://%s/actors/%s", host, testActorHash),
		Handle:     testActorHandle,
		Hash:       testActorHash,
		PublicKey:  publicKeyFrom(key),
		PrivateKey: key,
	}

	defaultTestAccountS2S = testAccount{
		Id:         fmt.Sprintf("http://%s/actors/%d", s2shost, 666),
		Handle:     "lou",
		Hash:       "666",
		PublicKey:  publicKeyFrom(key),
		PrivateKey: key,
	}

	extraAccount = testAccount{
		Id:     fmt.Sprintf("http://%s/actors/%s", host, extraActorHash),
		Handle: extraActorHandle,
		Hash:   extraActorHash,
	}

	defaultTestApp = testAccount{
		Id:   fmt.Sprintf("http://%s/actors/%s", host, testAppHash),
		Hash: testAppHash,
	}

	selfAccount = testAccount{
		Id:     fmt.Sprintf("http://%s/", host),
		Hash:   serviceHash,
		Handle: "self",
	}

	lastActivity = &objectVal{}
)

type assertFn func(v bool, msg string, args ...interface{})
type errFn func(format string, args ...interface{})
type requestGetAssertFn func(iri string, acc *testAccount) map[string]interface{}
type objectPropertiesAssertFn func(ob map[string]interface{}, testVal *objectVal)
type mapFieldAssertFn func(ob map[string]interface{}, key string, testVal interface{})
type stringArrFieldAssertFn func(ob []interface{}, testVal []string)

func errorf(t *testing.T) errFn {
	return func(msg string, args ...interface{}) {
		msg = fmt.Sprintf("%s\n------- Stack -------\n%s\n", msg, debug.Stack())
		if args == nil || len(args) == 0 {
			return
		}
		t.Fatalf(msg, args...)
	}
}

func errIfNotTrue(t *testing.T) assertFn {
	return func(v bool, msg string, args ...interface{}) {
		if !v {
			errorf(t)(msg, args...)
		}
	}
}

func errOnArray(t *testing.T) stringArrFieldAssertFn {
	return func(arrI []interface{}, tVal []string) {
		arr := make([]string, len(arrI))
		for k, v := range arrI {
			arr[k] = fmt.Sprintf("%s", v)
		}
		errIfNotTrue(t)(len(tVal) == len(arr), "invalid array count %d, expected %d", len(arr), len(tVal))
		if len(tVal) > 0 {
			sort.Strings(tVal)
			sort.Strings(arr)
			for k, iri := range tVal {
				t.Run(fmt.Sprintf("[%s]", iri), func(t *testing.T) {
					vk := arr[k]
					errIfNotTrue(t)(iri == vk, "array element at pos %d, %s does not match expected %s", k, vk, iri)
				})
			}
		}
	}
}

func errOnMapProp(t *testing.T) mapFieldAssertFn {
	return func(ob map[string]interface{}, key string, tVal interface{}) {
		t.Run(key, func(t *testing.T) {
			assertTrue := errIfNotTrue(t)
			assertMapKey := errOnMapProp(t)
			assertObjectProperties := errOnObjectProperties(t)
			assertArrayValues := errOnArray(t)
			val, ok := ob[key]
			assertTrue(ok, "Could not load %q property of item: %#v", key, ob)

			switch tt := tVal.(type) {
			case float64, float32:
				v, okA := val.(float64)

				assertTrue(okA, "Unable to convert %#v to %T type, Received %#v:(%T)", val, v, val, val)
				assertTrue(v == tt, "Invalid %q, %d expected, received %d", key, tt, v)
			case int64, int32, int16, int8:
				// for some reason the json.Decode extracts the int values as floats
				v, okA := val.(float64)

				assertTrue(okA, "Unable to convert %#v to %T type, Received %#v:(%T)", val, v, val, val)
				assertTrue(int64(v) == tt, "Invalid %q, %d expected, received %d", key, tt, int64(v))
			case string, []byte:
				// the case when the mock test value is a string, but corresponds to an object in the json
				// so we need to verify the json's object id against our mock value
				v1, okA := val.(string)
				v2, okB := val.(map[string]interface{})
				assertTrue(okA || okB, "Unable to convert %#v to %T or %T types, Received %#v:(%T)", val, v1, v2, val, val)
				if okA {
					assertTrue(v1 == tt, "Invalid %q, %q expected, received %q", key, tt, v1)
				}
				if okB {
					assertMapKey(v2, "id", tt)
				}
			case *objectVal:
				// this is the case where the mock value is a pointer to objectVal (so we can dereference against it's id)
				// and check the subsequent properties
				if tt != nil {
					v1, okA := val.(string)
					v2, okB := val.(map[string]interface{})
					assertTrue(okA || okB, "Unable to convert %#v to %T or %T types, Received %#v:(%T)", val, v1, v2, val, val)
					if okA {
						if tt.id == "" {
							// the id was empty - probably an object loaded dynamically
							tt.id = v1
						}
						assertTrue(v1 == tt.id, "Invalid %q, %q expected in %#v", "id", v1, tt)
					}
					if okB {
						assertObjectProperties(v2, tt)
					}
				}
			case []string:
				v1, okA := val.([]interface{})
				v2, okB := tVal.([]string)
				assertTrue(okA || okB, "Unable to convert %#v to %T or %T types, Received %#v:(%T)", val, v1, v2, val, val)
				assertArrayValues(v1, v2)
			case []*objectVal:
				v1, okA := val.([]interface{})
				v2, okB := tVal.([]*objectVal)
				assertTrue(okA && okB, "Unable to convert %#v to %T or %T types, Received %#v:(%T)", val, v1, v2, val, val)
				for i, intf := range v1 {
					ob, ok := intf.(map[string]interface{})
					assertTrue(ok, "Unable to convert %s prop with value %#v to %T", key, intf, ob)
					assertObjectProperties(ob, v2[i])
				}
			default:
				assertTrue(false, "UNKNOWN check for %q, %#v expected %#v", key, val, t)
			}
		})
	}
}

func errOnObjectProperties(t *testing.T) objectPropertiesAssertFn {
	return func(ob map[string]interface{}, tVal *objectVal) {
		t.Run(fmt.Sprintf("[%s]%s", tVal.typ, tVal.id), func(t *testing.T) {
			fail := errorf(t)
			assertTrue := errIfNotTrue(t)
			assertMapKey := errOnMapProp(t)
			assertGetRequest := errNotOKGetRequest(t)
			assertObjectProperties := errOnObjectProperties(t)

			if tVal == nil {
				return
			}
			if tVal.id != "" {
				assertMapKey(ob, "id", tVal.id)
			}
			if tVal.typ != "" {
				assertMapKey(ob, "type", tVal.typ)
			}
			if tVal.name != "" {
				assertMapKey(ob, "name", tVal.name)
			}
			if tVal.preferredUsername != "" {
				assertMapKey(ob, "preferredUsername", tVal.preferredUsername)
			}
			if tVal.content != "" {
				assertMapKey(ob, "content", tVal.content)
			}
			if tVal.summary != "" {
				assertMapKey(ob, "summary", tVal.summary)
			}
			if tVal.url != "" {
				assertMapKey(ob, "url", tVal.url)
			}
			if tVal.author != "" {
				assertMapKey(ob, "attributedTo", tVal.author)
			}
			if tVal.inbox != nil {
				assertMapKey(ob, "inbox", tVal.inbox)
				if tVal.inbox.typ != "" && len(tVal.inbox.id) > 0 {
					dCol := assertGetRequest(tVal.inbox.id, nil)
					assertObjectProperties(dCol, tVal.inbox)
				}
			}
			if tVal.outbox != nil {
				assertMapKey(ob, "outbox", tVal.outbox)
				if tVal.outbox.typ != "" && len(tVal.outbox.id) > 0 {
					dCol := assertGetRequest(tVal.outbox.id, nil)
					assertObjectProperties(dCol, tVal.outbox)
				}
			}
			if tVal.liked != nil {
				assertMapKey(ob, "liked", tVal.liked)
				if tVal.liked.typ != "" && len(tVal.liked.id) > 0 {
					dCol := assertGetRequest(tVal.liked.id, nil)
					assertObjectProperties(dCol, tVal.liked)
				}
			}
			if tVal.following != nil {
				assertMapKey(ob, "following", tVal.following)
				if tVal.following.typ != "" && len(tVal.following.id) > 0 {
					dCol := assertGetRequest(tVal.following.id, nil)
					assertObjectProperties(dCol, tVal.following)
				}
			}
			if tVal.followers != nil {
				assertMapKey(ob, "followers", tVal.followers)
				if tVal.followers.typ != "" && len(tVal.followers.id) > 0 {
					dCol := assertGetRequest(tVal.followers.id, nil)
					assertObjectProperties(dCol, tVal.followers)
				}
			}
			if tVal.act != nil {
				assertMapKey(ob, "actor", tVal.act)
				if tVal.act.typ != "" && len(tVal.act.id) > 0 {
					var dAct map[string]interface{}
					if tVal.act.typ == "Tombstone" {
						dAct = errNotGoneGetRequest(t)(tVal.act.id, nil)
					} else {
						dAct = assertGetRequest(tVal.act.id, nil)
					}
					assertObjectProperties(dAct, tVal.act)
				}
			}
			if tVal.obj != nil {
				assertMapKey(ob, "object", tVal.obj)
				if tVal.obj.typ != "" && len(tVal.obj.id) > 0 {
					var dOb map[string]interface{}
					if tVal.obj.typ == "Tombstone" {
						dOb = errNotGoneGetRequest(t)(tVal.obj.id, nil)
					} else {
						dOb = assertGetRequest(tVal.obj.id, nil)
					}
					assertObjectProperties(dOb, tVal.obj)
				}
			}
			if tVal.audience != nil {
				assertMapKey(ob, "audience", tVal.audience)
				audOb, _ := ob["audience"]
				aud, ok := audOb.([]interface{})
				assertTrue(ok, "received audience is not a []string, received %T", aud)
				errOnArray(t)(aud, tVal.audience)
			}
			if tVal.first != nil {
				assertMapKey(ob, "first", tVal.first)
				if tVal.first.typ != "" && len(tVal.first.id) > 0 {
					derefCol := assertGetRequest(tVal.first.id, nil)
					assertObjectProperties(derefCol, tVal.first)
				}
			}
			if tVal.next != nil {
				assertMapKey(ob, "next", tVal.next)
				if tVal.next.typ != "" && len(tVal.next.id) > 0 {
					derefCol := assertGetRequest(tVal.next.id, nil)
					assertObjectProperties(derefCol, tVal.next)
				}
			}
			if tVal.current != nil {
				assertMapKey(ob, "current", tVal.current)
				if tVal.current.typ != "" && len(tVal.current.id) > 0 {
					dCol := assertGetRequest(tVal.current.id, nil)
					assertObjectProperties(dCol, tVal.current)
				}
			}
			if tVal.last != nil {
				assertMapKey(ob, "last", tVal.last)
				if tVal.last.typ != "" && len(tVal.last.id) > 0 {
					derefCol := assertGetRequest(tVal.last.id, nil)
					assertObjectProperties(derefCol, tVal.last)
				}
			}
			if tVal.partOf != nil {
				assertMapKey(ob, "partOf", tVal.partOf)
				if tVal.partOf.typ != "" && len(tVal.partOf.id) > 0 {
					derefCol := assertGetRequest(tVal.partOf.id, nil)
					assertObjectProperties(derefCol, tVal.partOf)
				}
			}
			if tVal.tag != nil {
				assertMapKey(ob, "tag", tVal.tag)
				if len(tVal.tag) > 0 {
					for _, tVal := range tVal.tag {
						if len(tVal.id) == 0 {
							continue
						}
						derefCol := assertGetRequest(tVal.id, nil)
						assertObjectProperties(derefCol, tVal)
					}
				}
			}
			if tVal.anyOf != nil {
				assertMapKey(ob, "anyOf", tVal.anyOf)
				if len(tVal.anyOf) > 0 {
					for _, tVal := range tVal.anyOf {
						if len(tVal.id) == 0 {
							continue
						}
						derefCol := assertGetRequest(tVal.id, nil)
						assertObjectProperties(derefCol, tVal)
					}
				}
			}
			if tVal.oneOf != nil {
				assertMapKey(ob, "oneOf", tVal.oneOf)
				if len(tVal.oneOf) > 0 {
					for _, tVal := range tVal.oneOf {
						if len(tVal.id) == 0 {
							continue
						}
						derefCol := assertGetRequest(tVal.id, nil)
						assertObjectProperties(derefCol, tVal)
					}
				}
			}
			if tVal.itemCount > 0 {
				assertMapKey(ob, "totalItems", tVal.itemCount)
				itemsKey := func(typ string) string {
					if typ == string(vocab.CollectionType) {
						return "items"
					}
					return "orderedItems"
				}(tVal.typ)
				if len(tVal.items) > 0 {
					val, ok := ob[itemsKey]
					assertTrue(ok, "Could not load %q property of collection: %#v\n\n%#v\n\n", itemsKey, ob, tVal.items)
					items, ok := val.([]interface{})
					assertTrue(ok, "Invalid property %q %#v, expected %T", itemsKey, val, items)
					ti, ok := ob["totalItems"].(float64)
					assertTrue(ok, "Invalid property %q %#v, expected %T", "totalItems", val, items)
					assertTrue(len(items) == int(ti),
						"Invalid item count for collection %q %d, expected %d", itemsKey, len(items), tVal.itemCount,
					)
				foundItem:
					for k, testIt := range tVal.items {
						url, _ := url.Parse(tVal.id)
						iri := fmt.Sprintf("%s%s/%s", apiURL, url.Path, k)
						for _, it := range items {
							switch act := it.(type) {
							case map[string]interface{}:
								assertTrue(ok, "Unable to convert %#v to %T type, Received %#v:(%T)", it, act, it, it)
								itId, ok := act["id"]
								assertTrue(ok, "Could not load %q property of item: %#v", "id", act)
								itIRI, ok := itId.(string)
								assertTrue(ok, "Unable to convert %#v to %T type, Received %#v:(%T)", itId, itIRI, val, val)
								if strings.EqualFold(itIRI, iri) {
									assertObjectProperties(act, testIt)
									dAct := assertGetRequest(itIRI, nil)
									assertObjectProperties(dAct, testIt)
									continue foundItem
								} else {
									continue
								}
							case string:
								if testIt.id != "" {
									if strings.EqualFold(act, iri) {
										assertTrue(act == testIt.id, "invalid item ID %s, expected %s", act, testIt.id)
										continue foundItem
									}
								}
							}
						}
						fail("Unable to find %s in the %s collection %#v", iri, itemsKey, items)
					}
				}
			}
		})
	}
}

func getRequest(t *testing.T, st int) func(iri string, acc *testAccount) map[string]interface{} {
	return func(iri string, acc *testAccount) map[string]interface{} {
		if iri == "" {
			return nil
		}
		tVal := testPair{
			req: testReq{met: http.MethodGet, url: iri, account: acc},
			res: testRes{code: st},
		}
		return errOnRequest(t)(tVal)
	}
}

func errNotOKGetRequest(t *testing.T) requestGetAssertFn {
	return getRequest(t, http.StatusOK)
}

func errNotGoneGetRequest(t *testing.T) requestGetAssertFn {
	return getRequest(t, http.StatusGone)
}

func addOAuth2Auth(r *http.Request, a *testAccount) error {
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.AuthToken))
	return nil
}

func addHTTPSigAuth(req *http.Request, acc *testAccount) error {
	signHdrs := []string{"(request-target)", "host", "date"}
	keyId := fmt.Sprintf("%s#main-key", acc.Id)
	return httpsig.NewSigner(keyId, acc.PrivateKey, keyType(acc.PrivateKey), signHdrs).Sign(req)
}

func signRequest(req *http.Request, acc *testAccount) error {
	date, _ := time.Parse(time.RFC3339, "2019-01-23T01:23:45Z")
	req.Header.Set("Date", date.UTC().Format(http.TimeFormat))

	if path.Base(req.URL.Path) == "inbox" {
		return addHTTPSigAuth(req, acc)
	}
	return addOAuth2Auth(req, acc)
}

func errOnRequest(t *testing.T) func(testPair) map[string]interface{} {
	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 400 * time.Second,
	}
	return func(test testPair) map[string]interface{} {
		res := make(map[string]interface{})
		t.Run(test.label(), func(t *testing.T) {
			assertTrue := errIfNotTrue(t)
			assertGetRequest := errNotOKGetRequest(t)
			assertObjectProperties := errOnObjectProperties(t)
			if len(test.req.headers) == 0 {
				test.req.headers = make(http.Header, 0)
				test.req.headers.Set("User-Agent", UserAgent)

				test.req.headers.Set("Cache-Control", "no-cache")
			}
			if test.req.met == "" {
				test.req.met = http.MethodPost
			}
			if test.req.met == http.MethodPost {
				test.req.headers.Set("Content-Type", client.ContentTypeActivityJson)
			}
			if test.req.met == http.MethodGet {
				test.req.headers.Set("Accept", HeaderAccept)
			}
			if test.res.code == 0 {
				test.res.code = http.StatusCreated
			}
			if test.req.bodyFn != nil {
				test.req.body, _ = test.req.bodyFn()
			}
			body := []byte(test.req.body)
			b := make([]byte, 0)

			var err error
			if test.req.urlFn != nil {
				test.req.url = test.req.urlFn()
			}
			isClientRequest := path.Base(test.req.url) == string(vocab.Outbox)
			ctx := context.Background()
			req, err := http.NewRequestWithContext(ctx, test.req.met, test.req.url, bytes.NewReader(body))
			assertTrue(err == nil, "Error: unable to create request: %s", err)

			req.Header = test.req.headers
			if test.req.account != nil {
				err := signRequest(req, test.req.account)
				assertTrue(err == nil, "Error: unable to sign request: %s", err)
			}
			resp, err := c.Do(req)

			assertTrue(resp != nil, "Error: request failed: response is nil")
			assertTrue(err == nil, "Error: request failed: %s", err)

			b, err = ioutil.ReadAll(resp.Body)
			assertTrue(err == nil, "Error: invalid HTTP body! Read %d bytes %s", len(b), b)

			assertTrue(resp.StatusCode == test.res.code,
				"Error: invalid HTTP response %d, expected %d\nReq[%s:%s]\n    %v\n%s\nRes[%s]:\n    %v\n    %s",
				resp.StatusCode, test.res.code, req.Method, req.URL, req.Header, body, resp.Status, resp.Header, b)

			if test.req.met == http.MethodPost {
				location, ok := resp.Header["Location"]
				if ok {
					assertTrue(ok, "Server didn't respond with a Location header even though it responded with a %d status", resp.StatusCode)
					assertTrue(len(location) == 1, "Server responded with %d Location headers which is not expected", len(location))
					newObj, err := url.Parse(location[0])
					newObjURL := newObj.String()
					assertTrue(err == nil, "Location header holds invalid URL %s", newObjURL)
					if isClientRequest {
						assertTrue(strings.Contains(newObjURL, apiURL), "Location header holds invalid URL %s, expected to contain %s", newObjURL, apiURL)
					}
					test.act = &objectVal{
						id: newObjURL,
					}
					lastActivity = test.act
					if test.res.val == nil {
						test.res.val = &objectVal{}
					}
					if test.res.val.id == "" {
						// this is the location of the Activity not of the created object
						test.res.val.id = newObjURL
					}
				}
			}
			contentType := resp.Header.Get("Content-Type")
			if contentType == jsonld.ContentType || contentType == client.ContentTypeActivityJson {
				err = json.Unmarshal(b, &res)
				assertTrue(err == nil, "Error: unmarshal failed: %s", err)
				assertTrue(res != nil, "Error: unmarshal failed: nil result")
			}
			if test.res.val != nil {
				if test.req.met == http.MethodGet {
					assertObjectProperties(res, test.res.val)
				} else if loadAfterPost(test, req) {
					saved := assertGetRequest(test.res.val.id, test.req.account)
					assertObjectProperties(saved, test.res.val)
				}
			}
		})
		return res
	}
}

var (
	Verbose bool
	Silent  bool
)

func loadAfterPost(test testPair, req *http.Request) bool {
	return test.res.val.id != "" && test.res.val.id != req.URL.String()
}

func runTestSuite(t *testing.T, pairs testPairs) {
	for _, suite := range pairs {
		for _, options := range suite.configs {
			go SetupAPP(options).Run(context.TODO())
		}
		name := suite.name
		t.Run(name, func(t *testing.T) {
			for _, options := range suite.configs {
				seedTestData(t, suite.mocks, options)
			}
			for _, test := range suite.tests {
				t.Run(test.label(), func(t *testing.T) {
					for _, options := range suite.configs {
						seedTestData(t, test.mocks, options)
					}
					errOnRequest(t)(test)
				})
			}
		})
		for _, options := range suite.configs {
			// NOTE(marius): we removed the deferred app.Stop(),
			// to avoid race conditions when running multiple FedBOX instances for the federated tests
			cleanDB(t, options)
		}
	}
}
