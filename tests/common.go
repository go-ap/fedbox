//go:build integration

package tests

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"testing"
	"time"

	"git.sr.ht/~mariusor/lw"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/fedbox"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/cmd"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/filters"
	"github.com/go-ap/jsonld"
	"github.com/go-fed/httpsig"
)

// UserAgent value that the client uses when performing requests
var UserAgent = "test-go-http-client"
var HeaderAccept = `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`

var activityCount = 0

var C2SConfig = config.Options{
	Env:         env.TEST,
	Host:        "127.0.0.1:9998",
	Listen:      "127.0.0.1:9998",
	BaseURL:     "http://127.0.0.1:9998/",
	LogLevel:    lw.NoLevel,
	StoragePath: filepath.Join(storagePath(), "127.0.0.1:9998"),
	Storage:     storageType(),
}

var S2SConfig = config.Options{
	Env:         env.TEST,
	Host:        "127.0.2.1:9999",
	Listen:      "127.0.2.1:9999",
	BaseURL:     "http://127.0.2.1:9999/",
	LogLevel:    lw.NoLevel,
	StoragePath: filepath.Join(storagePath(), "127.0.2.1:9999"),
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
	apps    map[vocab.IRI]*fedbox.FedBOX
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
	path, err := os.MkdirTemp(".cache", "*")
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

func Objects(iri vocab.IRI) vocab.IRI {
	return filters.ObjectsType.IRI(iri)
}

func Activities(iri vocab.IRI) vocab.IRI {
	return filters.ActivitiesType.IRI(iri)
}

func Actors(iri vocab.IRI) vocab.IRI {
	return filters.ActorsType.IRI(iri)
}

func ObjectsURL() string {
	return Objects(vocab.IRI(service.Id)).String()
}

func ActorsURL() string {
	return Actors(vocab.IRI(service.Id)).String()
}

func ActivitiesURL() string {
	return Activities(vocab.IRI(service.Id)).String()
}

func firstPage() url.Values {
	return url.Values(filters.FirstPage())
}
func InboxURL(account *testAccount) func() string {
	return func() string {
		return string(vocab.Inbox.IRI(vocab.IRI(account.Id)))
	}
}

func LikedURL(account *testAccount) func() string {
	return func() string {
		return string(vocab.Liked.IRI(vocab.IRI(account.Id)))
	}
}

func FollowersURL(account *testAccount) func() string {
	return func() string {
		return string(vocab.Followers.IRI(vocab.IRI(account.Id)))
	}
}

func FollowingURL(account *testAccount) func() string {
	return func() string {
		return string(vocab.Following.IRI(vocab.IRI(account.Id)))
	}
}

func OutboxURL(account *testAccount) func() string {
	return func() string {
		return string(vocab.Outbox.IRI(vocab.IRI(account.Id)))
	}
}

func RepliesURL(ob vocab.Item) func() string {
	return func() string {
		return fmt.Sprintf("%s/replies", ob.GetID())
	}
}

func qv(k string, ss ...string) url.Values {
	q := url.Values{k: make([]string, 0)}
	needsEscape := !strings.Contains(strings.ToLower(k), "iri")
	for _, s := range ss {
		if needsEscape {
			s = url.QueryEscape(s)
		}
		q[k] = append(q[k], s)
	}
	return q
}

func objectIRI(ss ...string) url.Values {
	return qv("object.iri", ss...)
}

func actorIRI(ss ...string) url.Values {
	return qv("actor.iri", ss...)
}

func nameOf(ss ...string) url.Values {
	return qv("name", ss...)
}

func attrTo(ss ...string) url.Values {
	return qv("attributedTo", ss...)
}

func iri(ss ...string) url.Values {
	return qv("iri", ss...)
}

func inReplyTo(ss ...string) url.Values {
	return qv("inReplyTo", ss...)
}

func urlOf(ss ...string) url.Values {
	return qv("url", ss...)
}

func typeOf(tt ...vocab.ActivityVocabularyType) url.Values {
	ss := make([]string, 0)
	for _, t := range tt {
		ss = append(ss, string(t))
	}
	return qv("type", ss...)
}

func CollectionURL(orig string, qq ...url.Values) string {
	u, _ := url.Parse(orig)
	q := u.Query()
	for _, qi := range qq {
		for k, vv := range qi {
			if _, ok := q[k]; !ok {
				q[k] = make([]string, 0)
			}
			q[k] = append(q[k], vv...)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
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
	apiIRI = vocab.IRI(apiURL)

	authCallbackURL = fmt.Sprintf("%s/auth/local/callback", apiURL)
	inboxURL        = InboxURL(&service)
	outboxURL       = OutboxURL(&service)
	baseURL         = service.Id

	edKey  = loadPrivateKeyFromDisk("mocks/keys/ed25519.prv")
	rsaKey = loadPrivateKeyFromDisk("mocks/keys/rsa256.prv")

	meta any = nil

	service = testAccount{Id: apiURL}

	defaultTestAccountC2S = testAccount{
		Id:         fmt.Sprintf("http://%s/actors/%s", host, testActorHash),
		Handle:     testActorHandle,
		Hash:       testActorHash,
		PublicKey:  publicKeyFrom(rsaKey),
		PrivateKey: rsaKey,
	}

	defaultTestAccountS2S = testAccount{
		Id:         fmt.Sprintf("http://%s/actors/%d", s2shost, 666),
		Handle:     "lou",
		Hash:       "666",
		PublicKey:  publicKeyFrom(edKey),
		PrivateKey: edKey,
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

type assertFn func(v bool, msg string, args ...any)
type errFn func(format string, args ...any)
type requestGetAssertFn func(iri string, acc *testAccount) map[string]any
type objectPropertiesAssertFn func(ob map[string]any, testVal *objectVal)
type mapFieldAssertFn func(ob map[string]any, key string, testVal any)
type stringArrFieldAssertFn func(ob []any, testVal []string)

func errorf(t *testing.T, errFn func(format string, args ...any)) errFn {
	t.Helper()
	return func(msg string, args ...any) {
		t.Helper()
		msg = fmt.Sprintf("%s\n------- Stack -------\n%s\n", msg, debug.Stack())
		if args == nil || len(args) == 0 {
			return
		}
		errFn(msg, args...)
	}
}

func errIfNotTrue(t *testing.T, errFn func(format string, args ...any)) assertFn {
	return func(v bool, msg string, args ...any) {
		t.Helper()
		if !v {
			errorf(t, errFn)(msg, args...)
		}
	}
}

func errOnArray(t *testing.T) stringArrFieldAssertFn {
	return func(arrI []any, tVal []string) {
		arr := make([]string, len(arrI))
		for k, v := range arrI {
			arr[k] = fmt.Sprintf("%s", v)
		}
		errIfNotTrue(t, t.Errorf)(len(tVal) == len(arr), "invalid array count %d, expected %d", len(arr), len(tVal))
		if len(tVal) > 0 {
			sort.Strings(tVal)
			sort.Strings(arr)
			for k, iri := range tVal {
				t.Run(fmt.Sprintf("[%s]", iri), func(t *testing.T) {
					vk := arr[k]
					errIfNotTrue(t, t.Errorf)(iri == vk, "array element at pos %d, %s does not match expected %s", k, vk, iri)
				})
			}
		}
	}
}

func errOnMapProp(t *testing.T, errFn func(format string, args ...any)) mapFieldAssertFn {
	return func(ob map[string]any, key string, tVal any) {
		t.Helper()
		t.Run(key, func(t *testing.T) {
			assertTrue := errIfNotTrue(t, errFn)
			assertMapKey := errOnMapProp(t, errFn)
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

				i := int64(v)
				ti := tt.(int64)
				assertTrue(i == ti, "Invalid %q expected %d, received %d, in %+v", key, ti, i, ob)
			case string, []byte:
				// the case when the mock test value is a string, but corresponds to an object in the json
				// so, we need to verify the JSON object's ID against our mock value
				v1, okA := val.(string)
				v2, okB := val.(map[string]any)
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
					v2, okB := val.(map[string]any)
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
				v1, okA := val.([]any)
				v2, okB := tVal.([]string)
				assertTrue(okA || okB, "Unable to convert %#v to %T or %T types, Received %#v:(%T)", val, v1, v2, val, val)
				assertArrayValues(v1, v2)
			case []*objectVal:
				v1, okA := val.([]any)
				v2, okB := tVal.([]*objectVal)
				assertTrue(okA && okB, "Unable to convert %#v to %T or %T types, Received %#v:(%T)", val, v1, v2, val, val)
				for i, intf := range v1 {
					ob, ok := intf.(map[string]any)
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
	t.Helper()
	return func(ob map[string]any, tVal *objectVal) {
		t.Helper()
		t.Run(fmt.Sprintf("[%s]%s", tVal.typ, tVal.id), func(t *testing.T) {
			fail := errorf(t, t.Errorf)
			assertTrue := errIfNotTrue(t, t.Errorf)
			assertMapKey := errOnMapProp(t, t.Errorf)
			warnMapKey := errOnMapProp(t, t.Logf)
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
					var dAct map[string]any
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
					var dOb map[string]any
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
				aud, ok := audOb.([]any)
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
				warnMapKey(ob, "totalItems", tVal.itemCount)
				itemsKey := func(typ string) string {
					if typ == string(vocab.CollectionType) {
						return "items"
					}
					return "orderedItems"
				}(tVal.typ)
				if len(tVal.items) > 0 {
					val, ok := ob[itemsKey]
					assertTrue(ok, "Could not load %q property of collection:\n\n %#v\n\n%#v\n\n", itemsKey, ob, tVal)
					items, ok := val.([]any)
					assertTrue(ok, "Invalid property %q %#v, expected %T", itemsKey, val, items)
					assertTrue(ok, "Invalid property %q %#v, expected %T", "totalItems", val, items)
					assertTrue(len(items) == int(tVal.itemCount),
						"Invalid item count for collection %q %d, expected %d", itemsKey, len(items), tVal.itemCount,
					)
				foundItem:
					for k, testIt := range tVal.items {
						url, _ := url.Parse(tVal.id)
						iri := fmt.Sprintf("%s%s/%s", apiURL, url.Path, k)
						for _, it := range items {
							switch act := it.(type) {
							case map[string]any:
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

func getRequest(t *testing.T, st int) func(iri string, acc *testAccount) map[string]any {
	return func(iri string, acc *testAccount) map[string]any {
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
	t.Helper()
	return getRequest(t, http.StatusOK)
}

func errNotGoneGetRequest(t *testing.T) requestGetAssertFn {
	t.Helper()
	return getRequest(t, http.StatusGone)
}

func addOAuth2Auth(r *http.Request, a *testAccount) error {
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.AuthToken))
	return nil
}

func addHTTPSigAuth(req *http.Request, acc *testAccount) error {
	signHdrs := []string{httpsig.RequestTarget, "Host", "Date"}
	bodyBuf := bytes.Buffer{}

	if req.Method == http.MethodPost {
		signHdrs = append(signHdrs, "Digest")

		if req.Body != nil {
			if _, err := io.Copy(&bodyBuf, req.Body); err == nil {
				req.Body = io.NopCloser(&bodyBuf)
			}
		}
	}

	keyId := fmt.Sprintf("%s#main-key", acc.Id)
	signatureExpiration := int64(time.Hour.Seconds())

	algos := make([]httpsig.Algorithm, 0)
	switch acc.PrivateKey.(type) {
	case *rsa.PrivateKey:
		algos = append(algos, httpsig.RSA_SHA256, httpsig.RSA_SHA512)
	case *ecdsa.PrivateKey:
		algos = append(algos, httpsig.ECDSA_SHA512, httpsig.ECDSA_SHA256)
	case ed25519.PrivateKey:
		algos = append(algos, httpsig.ED25519)
	}
	var err error
	for _, alg := range algos {
		signer, _, err1 := httpsig.NewSigner([]httpsig.Algorithm{alg}, httpsig.DigestSha256, signHdrs, httpsig.Signature, signatureExpiration)
		if err1 == nil {
			err2 := signer.SignRequest(acc.PrivateKey, keyId, req, bodyBuf.Bytes())
			if err2 == nil {
				return nil
			} else {
				err = err2
			}
		} else {
			err = err1
		}
	}
	return err
}

func signRequest(req *http.Request, acc *testAccount) error {
	date, _ := time.Parse(time.RFC3339, "2019-01-23T01:23:45Z")
	req.Header.Set("Date", date.UTC().Format(http.TimeFormat))

	if path.Base(req.URL.Path) == "inbox" {
		return addHTTPSigAuth(req, acc)
	}
	return addOAuth2Auth(req, acc)
}

func hostFromUrl(uu string) string {
	u, err := url.ParseRequestURI(uu)
	if err != nil {
		return uu
	}
	return u.Host
}

func errOnRequest(t *testing.T) func(testPair) map[string]any {
	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 400 * time.Second,
	}
	return func(test testPair) map[string]any {
		t.Helper()
		res := make(map[string]any)
		t.Run(test.label(), func(t *testing.T) {
			assertTrue := errIfNotTrue(t, t.Errorf)
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
			if test.req.headers.Get("Host") == "" {
				test.req.headers.Set("Host", hostFromUrl(test.req.url))
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

			assertTrue(err == nil, "Error: request failed: %s", err)
			assertTrue(resp != nil, "Error: request failed: response is nil")
			if resp == nil {
				return
			}

			b, err = io.ReadAll(resp.Body)
			assertTrue(err == nil, "Error: invalid HTTP body! Read %d bytes %s", len(b), b)

			assertTrue(resp.StatusCode == test.res.code,
				"Error: invalid HTTP response %d, expected %d\nReq[%s:%s]\n    %v\n%s\nRes[%s]:\n    %v\n    %s",
				resp.StatusCode, test.res.code, req.Method, req.URL, req.Header, body, resp.Status, resp.Header, b)

			if test.req.met == http.MethodPost {
				if resp.StatusCode >= http.StatusFound && resp.StatusCode < http.StatusBadRequest {
					location, ok := resp.Header["Location"]
					assertTrue(ok, "Server didn't respond with a Location header even though it responded with a %d status", resp.StatusCode)
					if ok {
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

var Verbose bool

func loadAfterPost(test testPair, req *http.Request) bool {
	return test.res.val.id != "" && test.res.val.id != req.URL.String()
}

func cleanupTestPairs(pairs testPairs, t *testing.T) {
	if t.Failed() {
		return
	}
	for _, suite := range pairs {
		for _, options := range suite.configs {
			// NOTE(marius): we removed the deferred app.Stop(),
			// to avoid race conditions when running multiple FedBOX instances for the federated tests
			cleanDB(t, options)
		}
	}
}

func runTestSuite(t *testing.T, pairs testPairs) {
	defer cleanupTestPairs(pairs, t)

	t.Helper()

	for _, suite := range pairs {
		ctx, stopFn := context.WithCancel(context.TODO())
		runInstances := func() {
			suite.apps = make(map[vocab.IRI]*fedbox.FedBOX)
			for _, options := range suite.configs {
				if Verbose {
					options.LogLevel = lw.TraceLevel
				}

				self := ap.Self(ap.DefaultServiceIRI(options.BaseURL))
				if err := cmd.Bootstrap(options, self); err != nil {
					t.Fatalf("%+v", err)
					return
				}
				fb, err := getTestFedBOX(options)
				if err != nil {
					t.Fatalf("%s", err)
					return
				}
				suite.apps[self.ID] = fb
				go func() {
					if err = fb.Run(ctx); err != nil {
						t.Logf("Err: %+v", err)
					}
				}()
			}
		}

		name := suite.name
		t.Run(name, func(t *testing.T) {
			for _, test := range suite.tests {
				t.Run(test.label(), func(t *testing.T) {

					runInstances()

					for _, options := range suite.configs {
						app := suite.apps[vocab.IRI(options.BaseURL)]
						fields := lw.Ctx{"action": "seeding", "storage": options.Storage, "path": options.StoragePath}
						l := lw.Dev(lw.SetLevel(lw.DebugLevel)).WithContext(fields)

						mocks := append(suite.mocks, test.mocks...)
						if err := saveMocks(mocks, app, l); err != nil {
							t.Fatalf("%s", err)
						}
					}

					errOnRequest(t)(test)
					stopFn()
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
