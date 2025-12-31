package fedbox

import (
	"net/url"
	"strings"

	"git.sr.ht/~mariusor/lw"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/filters"
	"github.com/go-ap/processing"
)

func ByName(names ...string) url.Values {
	q := make(url.Values)
	q["name"] = names
	return q
}

func ByType(types ...vocab.ActivityVocabularyType) url.Values {
	q := make(url.Values)
	tt := make([]string, len(types))
	for i, t := range types {
		tt[i] = string(t)
	}
	q["type"] = tt
	return q
}

func ByURL(urls ...vocab.IRI) url.Values {
	q := make(url.Values)
	uu := make([]string, len(urls))
	for i, u := range urls {
		uu[i] = u.String()
	}
	q["url"] = uu
	return q
}

func IRIWithFilters(iri vocab.IRI, searchParams ...url.Values) vocab.IRI {
	q := make(url.Values)
	for _, params := range searchParams {
		for k, vals := range params {
			if _, ok := q[k]; !ok {
				q[k] = make([]string, 0)
			}
			q[k] = append(q[k], vals...)
		}
	}
	if s, err := iri.URL(); err == nil {
		s.RawQuery = q.Encode()
		iri = vocab.IRI(s.String())
	}
	return iri
}

func SearchActorsIRI(baseIRI vocab.IRI, searchParams ...url.Values) vocab.IRI {
	return IRIWithFilters(filters.ActorsType.IRI(baseIRI), searchParams...)
}

func (b *Base) List(iris vocab.IRIs, types ...vocab.ActivityVocabularyType) (vocab.ItemCollection, error) {
	var typeFilter []vocab.ActivityVocabularyType
	if len(types) > 0 {
		typeFilter = loadPubTypes(types...)
	}
	var items vocab.ItemCollection
	var err error

	for _, iri := range iris {
		ff, _ := filters.FromIRI(iri)
		ff = append(ff, filters.HasType(typeFilter...))

		col, err := b.Storage.Load(IRIWithFilters(iri, ByType(typeFilter...)), ff...)
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

func Saver(ctl *Base, actor *vocab.Actor) processing.P {
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

func (b *Base) AddActor(p *vocab.Person, pw []byte, author vocab.Actor) (*vocab.Person, error) {
	if b.Storage == nil {
		return nil, errors.Errorf("invalid storage backend")
	}
	if author.GetLink().Equals(auth.AnonymousActor.GetLink(), false) {
		self, err := ap.LoadActor(b.Storage, ap.DefaultServiceIRI(b.Conf.BaseURL))
		if err != nil {
			return nil, errors.NewNotFound(err, "unable to load current's instance Application actor")
		}
		if self.ID == "" {
			return nil, errors.NotFoundf("unable to load current's instance Application actor")
		}
		author = self
	}
	if author.GetID() == "" {
		return nil, errors.NotFoundf("unable to load current's instance Application actor: %s", b.Conf.BaseURL)
	}

	create := WrapObjectInCreate(p, author)
	outbox := vocab.Outbox.Of(author)
	if vocab.IsNil(outbox) {
		return nil, errors.Newf("unable to find Actor's outbox: %s", author)
	}

	if _, err := Saver(b, &author).ProcessClientActivity(create, author, outbox.GetLink()); err != nil {
		return nil, err
	}

	var err error
	if pwManager, ok := b.Storage.(PasswordChanger); ok && pw != nil {
		err = pwManager.PasswordSet(p.GetLink(), pw)
	}
	return p, err
}

func (b *Base) AddObject(p *vocab.Object, author vocab.Actor) (*vocab.Object, error) {
	if b.Storage == nil {
		return nil, errors.Errorf("invalid storage backend")
	}
	if author.GetLink().Equals(auth.AnonymousActor.GetLink(), false) {
		self, err := ap.LoadActor(b.Storage, ap.DefaultServiceIRI(b.Conf.BaseURL))
		if err != nil {
			return nil, errors.NewNotFound(err, "unable to load current's instance Application actor")
		}
		if self.ID == "" {
			return nil, errors.NotFoundf("unable to load current's instance Application actor")
		}
		author = self
	}
	if author.GetID() == "" {
		return nil, errors.NotFoundf("unable to load current's instance Application actor: %s", b.Conf.BaseURL)
	}

	processor := Saver(b, &author)
	outbox := vocab.Outbox.Of(author).GetLink()
	if vocab.IsNil(outbox) {
		return nil, errors.Newf("unable to find Actor's outbox: %s", author)
	}

	create := WrapObjectInCreate(p, author)
	if _, err := processor.ProcessClientActivity(create, author, outbox); err != nil {
		return nil, err
	}
	return p, nil
}

func (b *Base) DeleteObjects(reason string, inReplyTo []string, ids ...vocab.IRI) error {
	invalidRemoveTypes := append(append(vocab.ActivityTypes, vocab.IntransitiveActivityTypes...), vocab.TombstoneType)
	self := ap.Self(vocab.IRI(b.Conf.BaseURL))

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
		it, err := b.Storage.Load(iri)
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

	if _, err := Saver(b, &b.Service).ProcessClientActivity(d, self, vocab.Outbox.Of(d.Actor).GetLink()); err != nil {
		return err
	}

	_ = printItem(d, "text")
	return nil
}

func (b *Base) operateOnObjects(fn func(col vocab.IRI, it vocab.Item) error, to vocab.IRI, from ...vocab.IRI) error {
	if !vocab.ValidCollectionIRI(to) {
		return errors.Newf("destination is not a valid collection %s", to)
	}
	_, err := b.Storage.Load(to)
	if err != nil {
		return err
	}

	for _, iri := range from {
		it, err := b.Storage.Load(iri.GetLink())
		if err != nil {
			return err
		}
		if vocab.IsItemCollection(it) {
			return vocab.OnCollectionIntf(it, func(col vocab.CollectionInterface) error {
				return b.operateOnObjects(fn, to, col.Collection().IRIs()...)
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

func (b *Base) MoveObjects(to vocab.IRI, from ...vocab.IRI) error {
	st, ok := b.Storage.(processing.CollectionStore)
	if !ok {
		return errors.Newf("invalid storage %T", b.Storage)
	}

	copyFn := func(col vocab.IRI, it vocab.Item) error {
		if err := st.AddTo(col.GetLink(), it); err != nil {
			return err
		}

		if err := b.Storage.Delete(it.GetLink()); err != nil {
			return err
		}
		return nil
	}
	return b.operateOnObjects(copyFn, to, from...)
}

func (b *Base) CopyObjects(to vocab.IRI, from ...vocab.IRI) error {
	st, ok := b.Storage.(processing.CollectionStore)
	if !ok {
		return errors.Newf("invalid storage %T", b.Storage)
	}

	copyFn := func(col vocab.IRI, it vocab.Item) error {
		err := st.AddTo(col.GetLink(), it)
		if err != nil {
			b.Logger.Errorf("Error: %s", err)
		}
		return nil
	}
	return b.operateOnObjects(copyFn, to, from...)
}
