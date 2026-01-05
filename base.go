package fedbox

import (
	"crypto"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	cache2 "git.sr.ht/~mariusor/cache"
	"git.sr.ht/~mariusor/lw"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/client"
	"github.com/go-ap/client/debug"
	"github.com/go-ap/client/s2s"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/filters"
	"github.com/go-ap/processing"
	"github.com/openshift/osin"
)

func (ctl *Base) SendSignalToServer(sig syscall.Signal) func() error {
	return func() error {
		return ctl.SendSignal(sig)
	}
}

func (ctl *Base) SendSignal(sig syscall.Signal) error {
	pid, err := ctl.Conf.ReadPid()
	if err != nil {
		return errors.Annotatef(err, "unable to read pid file")
	}
	return syscall.Kill(pid, sig)
}

func (ctl *Base) infFn(s string, p ...any) {
	if ctl.Logger != nil {
		ctl.Logger.Infof(s, p...)
	}
}

func (ctl *Base) errFn(s string, p ...any) {
	if ctl.Logger != nil {
		ctl.Logger.Errorf(s, p...)
	}
}

type keyStorage interface {
	LoadKey(vocab.IRI) (crypto.PrivateKey, error)
}

func (ctl *Base) LoadLocalActorWithKey(actorIRI vocab.IRI) (*vocab.Actor, crypto.PrivateKey, error) {
	signActorID := ctl.Service.ID

	var signActor *vocab.Actor = &ctl.Service
	if maybeActorID, col := vocab.Split(actorIRI); filters.ValidCollection(col) {
		signActorID = maybeActorID
	}

	it, err := ctl.Storage.Load(signActorID)
	if err != nil {
		return signActor, nil, err
	}
	act, err := vocab.ToActor(it)
	if err != nil {
		return signActor, nil, err
	}
	signActor = act

	keyStore, ok := ctl.Storage.(keyStorage)
	if !ok {
		return signActor, nil, nil
	}

	prv, err := keyStore.LoadKey(signActorID)
	if err != nil {
		return signActor, prv, err
	}

	return signActor, prv, nil
}

func (ctl *Base) List(iris vocab.IRIs, types ...vocab.ActivityVocabularyType) (vocab.ItemCollection, error) {
	var typeFilter []vocab.ActivityVocabularyType
	if len(types) > 0 {
		typeFilter = loadPubTypes(types...)
	}
	var items vocab.ItemCollection
	var err error

	for _, iri := range iris {
		ff, _ := filters.FromIRI(iri)
		ff = append(ff, filters.HasType(typeFilter...))

		col, err := ctl.Storage.Load(ap.IRIWithFilters(iri, ap.ByType(typeFilter...)), ff...)
		if err != nil {
			return items, err
		}
		_ = vocab.OnItem(col, func(it vocab.Item) error {
			if !vocab.IsNil(it) {
				items = append(items, it)
			}
			return nil
		})
	}
	return items, err
}

func loadPubTypes(types ...vocab.ActivityVocabularyType) []vocab.ActivityVocabularyType {
	objectTyp := make(vocab.ActivityVocabularyTypes, 0)
	actorTyp := make(vocab.ActivityVocabularyTypes, 0)
	activityTyp := make(vocab.ActivityVocabularyTypes, 0)
	if len(types) == 0 {
		objectTyp = vocab.ObjectTypes
		actorTyp = vocab.ActorTypes
		activityTyp = vocab.ActivityTypes
	} else {
		for _, t := range types {
			if vocab.ObjectTypes.Contains(t) {
				objectTyp = append(objectTyp, t)
			}
			if vocab.ActorTypes.Contains(t) {
				actorTyp = append(actorTyp, t)
			}
			if vocab.ActivityTypes.Contains(t) {
				activityTyp = append(activityTyp, t)
			}
			if strings.ToLower(string(t)) == strings.ToLower(string(vocab.ObjectType)) {
				objectTyp = vocab.ObjectTypes
			}
			if strings.ToLower(string(t)) == strings.ToLower(string(vocab.ActorType)) {
				actorTyp = vocab.ActorTypes
			}
			if strings.ToLower(string(t)) == strings.ToLower(string(vocab.ActivityType)) {
				activityTyp = vocab.ActivityTypes
			}
		}
	}
	return append(append(objectTyp, actorTyp...), activityTyp...)
}

func ActorClient(ctl *Base, actor vocab.Item) *client.C {
	var tr http.RoundTripper = &http.Transport{}
	if ctl.debugMode.Load() {
		tr = debug.New(debug.WithTransport(tr), debug.WithPath(ctl.Conf.StoragePath))
	}

	if vocab.IsNil(actor) || !vocab.PublicNS.Equal(actor.GetLink()) {
		signActor, prv, err := ctl.LoadLocalActorWithKey(actor.GetLink())
		if err != nil {
			ctl.errFn("unable to sign request: %+s", err)
		} else if prv != nil && signActor != nil {
			tr = s2s.New(s2s.WithActor(signActor, prv), s2s.WithLogger(ctl.Logger.WithContext(lw.Ctx{"log": "HTTP-Sig"})))
		}
	}

	return initClient(tr, ctl.Conf, ctl.Logger)
}

func initClient(tr http.RoundTripper, conf config.Options, l lw.Logger) *client.C {
	cachePath, err := os.UserCacheDir()
	if err != nil {
		cachePath = os.TempDir()
	}

	if tr == nil {
		tr = &http.Transport{}
	}

	ua := fmt.Sprintf("%s@%s (+%s)", conf.BaseURL, conf.Version, ap.ProjectURL)
	baseClient := &http.Client{
		Transport: client.UserAgentTransport(ua, cache2.Private(tr, cache2.FS(filepath.Join(cachePath, conf.AppName)))),
	}

	return client.New(
		client.WithLogger(l.WithContext(lw.Ctx{"log": "client"})),
		client.WithHTTPClient(baseClient),
		client.SkipTLSValidation(!conf.Env.IsProd()),
	)
}

var InternalIRI = vocab.IRI("https://fedbox/")

// GenerateID creates an IRI that can be used to uniquely identify the "it" item, based on the collection "col" and
// its creator "by"
func GenerateID(base vocab.IRI) func(it vocab.Item, col vocab.Item, by vocab.Item) (vocab.ID, error) {
	return func(it vocab.Item, col vocab.Item, by vocab.Item) (vocab.ID, error) {
		typ := it.GetType()

		var partOf vocab.IRI
		if vocab.ActivityTypes.Contains(typ) || vocab.IntransitiveActivityTypes.Contains(typ) {
			partOf = filters.ActivitiesType.IRI(base)
		} else if vocab.ActorTypes.Contains(typ) || typ == vocab.ActorType {
			partOf = filters.ActorsType.IRI(base)
		} else {
			partOf = filters.ObjectsType.IRI(base)
		}
		return ap.GenerateID(it, partOf, by)
	}
}

func (ctl *Base) Saver(actor *vocab.Actor) processing.P {
	baseIRI := vocab.IRI(ctl.Conf.BaseURL)
	db := ctl.Storage
	l := ctl.Logger.WithContext(lw.Ctx{"log": "processing"})
	if vocab.IsNil(actor) {
		actor = &ctl.Service
	}
	cl := ActorClient(ctl, actor)
	p := processing.New(
		processing.WithIRI(baseIRI, InternalIRI), processing.WithLogger(l),
		processing.WithStorage(db), processing.WithClient(cl),
		processing.WithIDGenerator(GenerateID(baseIRI)),
		//processing.WithLocalIRIChecker(),
	)
	return p
}

func (ctl *Base) AddActor(p *vocab.Person, pw []byte, author vocab.Actor) (*vocab.Person, error) {
	if ctl.Storage == nil {
		return nil, errors.Errorf("invalid storage backend")
	}
	if author.GetLink().Equals(auth.AnonymousActor.GetLink(), false) {
		self, err := ap.LoadActor(ctl.Storage, ap.DefaultServiceIRI(ctl.Conf.BaseURL))
		if err != nil {
			return nil, errors.NewNotFound(err, "unable to load current's instance Application actor")
		}
		if self.ID == "" {
			return nil, errors.NotFoundf("unable to load current's instance Application actor")
		}
		author = self
	}
	if author.GetID() == "" {
		return nil, errors.NotFoundf("unable to load current's instance Application actor: %s", ctl.Conf.BaseURL)
	}

	create := ap.WrapObjectInCreate(p, author)
	outbox := vocab.Outbox.Of(author)
	if vocab.IsNil(outbox) {
		return nil, errors.Newf("unable to find Actor's outbox: %s", author)
	}

	if _, err := ctl.Saver(&author).ProcessClientActivity(create, author, outbox.GetLink()); err != nil {
		return nil, err
	}

	var err error
	if pwManager, ok := ctl.Storage.(passwordChanger); ok && pw != nil {
		err = pwManager.PasswordSet(p.GetLink(), pw)
	}
	return p, err
}

func (ctl *Base) AddObject(p *vocab.Object, author vocab.Actor) (*vocab.Object, error) {
	if ctl.Storage == nil {
		return nil, errors.Errorf("invalid storage backend")
	}
	if author.GetLink().Equals(auth.AnonymousActor.GetLink(), false) {
		self, err := ap.LoadActor(ctl.Storage, ap.DefaultServiceIRI(ctl.Conf.BaseURL))
		if err != nil {
			return nil, errors.NewNotFound(err, "unable to load current's instance Application actor")
		}
		if self.ID == "" {
			return nil, errors.NotFoundf("unable to load current's instance Application actor")
		}
		author = self
	}
	if author.GetID() == "" {
		return nil, errors.NotFoundf("unable to load current's instance Application actor: %s", ctl.Conf.BaseURL)
	}

	processor := ctl.Saver(&author)
	outbox := vocab.Outbox.Of(author).GetLink()
	if vocab.IsNil(outbox) {
		return nil, errors.Newf("unable to find Actor's outbox: %s", author)
	}

	create := ap.WrapObjectInCreate(p, author)
	if _, err := processor.ProcessClientActivity(create, author, outbox); err != nil {
		return nil, err
	}
	return p, nil
}

func (ctl *Base) DeleteObjects(reason string, inReplyTo []string, ids ...vocab.IRI) error {
	invalidRemoveTypes := append(append(vocab.ActivityTypes, vocab.IntransitiveActivityTypes...), vocab.TombstoneType)
	self := ap.Self(vocab.IRI(ctl.Conf.BaseURL))

	d := new(vocab.Delete)
	d.Type = vocab.DeleteType
	d.To = vocab.ItemCollection{vocab.PublicNS}
	d.CC = make(vocab.ItemCollection, 0)
	if reason != "" {
		d.Content = vocab.NaturalLanguageValuesNew()
		_ = d.Content.Append(vocab.NilLangRef, vocab.Content(reason))
	}
	if len(inReplyTo) > 0 {
		replIRI := make(vocab.ItemCollection, 0)
		for _, repl := range inReplyTo {
			if _, err := url.Parse(repl); err != nil {
				continue
			}
			replIRI = append(replIRI, vocab.IRI(repl))
		}
		d.InReplyTo = replIRI
	}
	d.Actor = self

	delItems := make(vocab.ItemCollection, 0)
	for _, iri := range ids {
		it, err := ctl.Storage.Load(iri)
		if err != nil {
			continue
		}
		// NOTE(marius): this should work if "it" is a collection or a single object
		_ = vocab.OnObject(it, func(o *vocab.Object) error {
			if invalidRemoveTypes.Contains(o.GetType()) {
				return nil
			}
			d.To = o.To
			d.Bto = o.Bto
			d.CC = o.CC
			d.BCC = o.BCC
			if o.AttributedTo != nil {
				d.CC = append(d.CC, o.AttributedTo.GetLink())
			}
			delItems = append(delItems, o.GetLink())
			return nil
		})
	}
	d.CC = append(d.CC, self.GetLink())
	if len(delItems) == 0 {
		return errors.NotFoundf("No items found to delete")
	}
	d.Object = delItems

	if _, err := ctl.Saver(&ctl.Service).ProcessClientActivity(d, self, vocab.Outbox.Of(d.Actor).GetLink()); err != nil {
		return err
	}

	//_ = printItem(d, "text")
	return nil
}

func (ctl *Base) operateOnObjects(fn func(col vocab.IRI, it vocab.Item) error, to vocab.IRI, from ...vocab.IRI) error {
	if !vocab.ValidCollectionIRI(to) {
		return errors.Newf("destination is not a valid collection %s", to)
	}
	_, err := ctl.Storage.Load(to)
	if err != nil {
		return err
	}

	for _, iri := range from {
		it, err := ctl.Storage.Load(iri.GetLink())
		if err != nil {
			return err
		}
		if vocab.IsItemCollection(it) {
			return vocab.OnCollectionIntf(it, func(col vocab.CollectionInterface) error {
				return ctl.operateOnObjects(fn, to, col.Collection().IRIs()...)
			})
		}
		if !vocab.IsObject(it) {
			return errors.Newf("Invalid object at IRI %s, %v", from, it)
		}

		if err = fn(to, it); err != nil {
			return err
		}
	}

	return nil
}

func (ctl *Base) MoveObjects(to vocab.IRI, from ...vocab.IRI) error {
	st, ok := ctl.Storage.(processing.CollectionStore)
	if !ok {
		return errors.Newf("invalid storage %T", ctl.Storage)
	}

	copyFn := func(col vocab.IRI, it vocab.Item) error {
		if err := st.AddTo(col.GetLink(), it); err != nil {
			return err
		}

		if err := ctl.Storage.Delete(it.GetLink()); err != nil {
			return err
		}
		return nil
	}
	return ctl.operateOnObjects(copyFn, to, from...)
}

func (ctl *Base) CopyObjects(to vocab.IRI, from ...vocab.IRI) error {
	st, ok := ctl.Storage.(processing.CollectionStore)
	if !ok {
		return errors.Newf("invalid storage %T", ctl.Storage)
	}

	copyFn := func(col vocab.IRI, it vocab.Item) error {
		err := st.AddTo(col.GetLink(), it)
		if err != nil {
			ctl.Logger.Errorf("Error: %s", err)
		}
		return nil
	}
	return ctl.operateOnObjects(copyFn, to, from...)
}

func (ctl *Base) DeleteClient(id string) error {
	iri := vocab.IRI(id)
	if _, err := iri.URL(); err != nil {
		iri = vocab.IRI(fmt.Sprintf("%s/%s/%s", ctl.Conf.BaseURL, filters.ActorsType, id))
	}
	err := ctl.DeleteObjects("Remove OAuth2 Client", nil, iri)
	if err != nil {
		return err
	}

	return ctl.Storage.RemoveClient(iri.String())
}

func (ctl *Base) ListClients() ([]osin.Client, error) {
	return ctl.Storage.ListClients()
}

func (ctl *Base) GenAuthToken(clientID, actorIdentifier string, _ any) (string, error) {
	if u, err := vocab.IRI(clientID).URL(); err == nil {
		clientID = filepath.Base(u.Path)
	}
	cl, err := ctl.Storage.GetClient(clientID)
	if err != nil {
		return "", err
	}

	now := time.Now().Truncate(time.Second).UTC()
	var f vocab.IRI
	if u, err := url.Parse(actorIdentifier); err == nil {
		f = vocab.IRI(u.String())
	} else {
		f = ap.SearchActorsIRI(ctl.Service.ID, ap.ByName(actorIdentifier), ap.ByType(vocab.ActorTypes...))
	}
	maybeActors, err := ctl.Storage.Load(f.GetLink())
	if err != nil {
		return "", err
	}
	if vocab.IsNil(maybeActors) {
		return "", errors.NotFoundf("not found")
	}
	var actor vocab.Item
	err = vocab.OnActor(maybeActors, func(act *vocab.Actor) error {
		actor = act
		return nil
	})
	if err != nil {
		return "", err
	}

	aud := &osin.AuthorizeData{
		Client:      cl,
		CreatedAt:   now,
		ExpiresIn:   86400,
		RedirectUri: cl.GetRedirectUri(),
		State:       "state",
	}

	// generate token code
	aud.Code, err = (&osin.AuthorizeTokenGenDefault{}).GenerateAuthorizeToken(aud)
	if err != nil {
		return "", err
	}

	// generate token directly
	ar := &osin.AccessRequest{
		Type:          osin.AUTHORIZATION_CODE,
		AuthorizeData: aud,
		Client:        cl,
		RedirectUri:   cl.GetRedirectUri(),
		Scope:         "scope",
		Authorized:    true,
		Expiration:    86400,
	}

	ad := &osin.AccessData{
		Client:        ar.Client,
		AuthorizeData: ar.AuthorizeData,
		AccessData:    ar.AccessData,
		ExpiresIn:     ar.Expiration,
		Scope:         ar.Scope,
		RedirectUri:   cl.GetRedirectUri(),
		CreatedAt:     now,
		UserData:      actor.GetLink(),
	}

	// generate access token
	ad.AccessToken, ad.RefreshToken, err = (&osin.AccessTokenGenDefault{}).GenerateAccessToken(ad, ar.GenerateRefresh)
	if err != nil {
		return "", err
	}
	// save authorize data
	if err = ctl.Storage.SaveAuthorize(aud); err != nil {
		return "", err
	}
	// save access token
	if err = ctl.Storage.SaveAccess(ad); err != nil {
		return "", err
	}

	return ad.AccessToken, nil
}

const URISeparator = "\n"

func (ctl *Base) AddClient(pw []byte, redirectUris []string, u any) (string, error) {
	var id string

	self := ap.Self(vocab.IRI(ctl.Conf.BaseURL))
	now := time.Now().UTC()
	name := "oauth-client-app"
	urls := make(vocab.ItemCollection, 0)

	for i, redirectUri := range redirectUris {
		if u, err := url.ParseRequestURI(redirectUri); err == nil {
			u.Path = filepath.Clean(u.Path)
			name = u.Host
			curURL := u.String()
			redirectUris[i] = curURL

			u.Path = ""
			_ = urls.Append(vocab.IRI(u.String()), vocab.IRI(curURL))
		}
	}
	p := &vocab.Application{
		Type:              vocab.ApplicationType,
		AttributedTo:      self.GetLink(),
		Audience:          vocab.ItemCollection{vocab.PublicNS},
		Generator:         self.GetLink(),
		Published:         now,
		PreferredUsername: vocab.DefaultNaturalLanguage(name),
		URL:               urls,
	}
	app, err := ctl.AddActor(p, pw, self)
	if err != nil {
		return "", err
	}
	if metaSaver, ok := ctl.Storage.(ap.MetadataStorage); ok {
		if err = ap.AddKeyToItem(metaSaver, p, ap.KeyTypeRSA); err != nil {
			ctl.Logger.Errorf("Error saving metadata for application %s: %s", name, err)
		}
	}

	// TODO(marius): allow for updates of the application actor with incoming parameters for Icon, Summary, samd.
	id = app.GetID().String()
	if id == "" {
		return "", errors.Newf("invalid actor saved, id is null")
	}

	// TODO(marius): add a local Client struct that implements Client and ClientSecretMatcher interfaces with bcrypt support
	//   It could even be a struct composite from an vocab.Application + secret and callback properties
	userData, _ := json.Marshal(u)
	d := osin.DefaultClient{
		Id:          id,
		Secret:      string(pw),
		RedirectUri: strings.Join(redirectUris, URISeparator),
		UserData:    userData,
	}

	return id, ctl.Storage.CreateClient(&d)
}
