package internal

import vocab "github.com/go-ap/activitypub"

func replaceHostInIRI(iri *vocab.IRI, repl vocab.IRI) *vocab.IRI {
	if iri.Contains(repl, false) {
		return iri
	}

	u, eu := iri.URL()
	if eu != nil {
		return iri
	}

	ru, er := repl.URL()
	if er != nil {
		return iri
	}

	u.Scheme = ru.Scheme
	u.Host = ru.Host
	*iri = vocab.IRI(u.String())
	return iri
}

func replaceHostInActor(a *vocab.Actor, repl vocab.IRI) {
	vocab.OnObject(a, func(o *vocab.Object) error {
		replaceHostInObject(o, repl)
		return nil
	})
}

func replaceHostInObject(o *vocab.Object, repl vocab.IRI) {
	replaceHostInIRI(&o.ID, repl)
	replaceHostInItem(o.AttributedTo, repl)
	replaceHostInItem(o.Attachment, repl)
	replaceHostInItem(o.Audience, repl)
	replaceHostInItem(o.Context, repl)
	replaceHostInItem(o.Generator, repl)
	replaceHostInItem(o.Icon, repl)
	replaceHostInItem(o.Image, repl)
	replaceHostInItem(o.InReplyTo, repl)
	replaceHostInItem(o.Location, repl)
	replaceHostInItem(o.Preview, repl)
	replaceHostInItem(o.Replies, repl)
	replaceHostInItem(o.Tag, repl)
	replaceHostInItem(o.To, repl)
	replaceHostInItem(o.Bto, repl)
	replaceHostInItem(o.CC, repl)
	replaceHostInItem(o.BCC, repl)
	replaceHostInItem(o.Likes, repl)
	replaceHostInItem(o.Shares, repl)
}

func replaceHostInActivity(o *vocab.Activity, repl vocab.IRI) {
	vocab.OnIntransitiveActivity(o, func(o *vocab.IntransitiveActivity) error {
		replaceHostInIntransitiveActivity(o, repl)
		return nil
	})
	replaceHostInItem(o.Object, repl)
}

func replaceHostInIntransitiveActivity(o *vocab.IntransitiveActivity, repl vocab.IRI) {
	vocab.OnObject(o, func(o *vocab.Object) error {
		replaceHostInObject(o, repl)
		return nil
	})
	replaceHostInItem(o.Actor, repl)
	replaceHostInItem(o.Target, repl)
	replaceHostInItem(o.Result, repl)
	replaceHostInItem(o.Origin, repl)
	replaceHostInItem(o.Instrument, repl)
}

func replaceHostInOrderedCollection(c *vocab.OrderedCollection, repl vocab.IRI)         {}
func replaceHostInOrderedCollectionPage(c *vocab.OrderedCollectionPage, repl vocab.IRI) {}
func replaceHostInCollection(c *vocab.Collection, repl vocab.IRI)                       {}
func replaceHostInCollectionPage(c *vocab.CollectionPage, repl vocab.IRI)               {}
func replaceHostInCollectionOfItems(c vocab.ItemCollection, repl vocab.IRI) {
	for _, it := range c {
		replaceHostInItem(it, repl)
	}
}

func replaceHostInItem(it vocab.Item, repl vocab.IRI) {
	if vocab.IsNil(it) {
		return
	}
	if it.IsCollection() {
		if it.GetType() == vocab.OrderedCollectionType {
			vocab.OnOrderedCollection(it, func(c *vocab.OrderedCollection) error {
				replaceHostInOrderedCollection(c, repl)
				return nil
			})
		}
		if it.GetType() == vocab.OrderedCollectionPageType {
			vocab.OnOrderedCollectionPage(it, func(c *vocab.OrderedCollectionPage) error {
				replaceHostInOrderedCollectionPage(c, repl)
				return nil
			})
		}
		if it.GetType() == vocab.CollectionType {
			vocab.OnCollection(it, func(c *vocab.Collection) error {
				replaceHostInCollection(c, repl)
				return nil
			})
		}
		if it.GetType() == vocab.CollectionPageType {
			vocab.OnCollectionPage(it, func(c *vocab.CollectionPage) error {
				replaceHostInCollectionPage(c, repl)
				return nil
			})
		}
		if it.GetType() == vocab.CollectionOfItems {
			vocab.OnItemCollection(it, func(c *vocab.ItemCollection) error {
				replaceHostInCollectionOfItems(*c, repl)
				return nil
			})
		}
	}
	if it.IsObject() {
		if vocab.IntransitiveActivityTypes.Contains(it.GetType()) {
			vocab.OnIntransitiveActivity(it, func(a *vocab.IntransitiveActivity) error {
				replaceHostInIntransitiveActivity(a, repl)
				return nil
			})
		}
		if vocab.ActivityTypes.Contains(it.GetType()) {
			vocab.OnActivity(it, func(a *vocab.Activity) error {
				replaceHostInActivity(a, repl)
				return nil
			})
		}
		if vocab.ActorTypes.Contains(it.GetType()) {
			vocab.OnActor(it, func(a *vocab.Actor) error {
				replaceHostInActor(a, repl)
				return nil
			})
		}
		if vocab.ObjectTypes.Contains(it.GetType()) {
			vocab.OnObject(it, func(o *vocab.Object) error {
				replaceHostInObject(o, repl)
				return nil
			})
		}
	}
	if it.IsLink() {
		l := it.GetLink()
		// FIXME(marius): because
		it = *replaceHostInIRI(&l, repl)
	}
}
