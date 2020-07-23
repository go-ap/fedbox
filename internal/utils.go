package internal

import pub "github.com/go-ap/activitypub"

func replaceHostInIRI(iri *pub.IRI, repl pub.IRI) *pub.IRI {
	if iri.Contains(repl, false) { return iri }

	u, eu := iri.URL()
	if eu != nil { return iri }

	ru, er := repl.URL()
	if er != nil { return iri }

	u.Scheme = ru.Scheme
	u.Host = ru.Host
	*iri = pub.IRI(u.String())
	return iri
}

func replaceHostInActor(a *pub.Actor, repl pub.IRI) {
	pub.OnObject(a, func(o *pub.Object) error {
		replaceHostInObject(o, repl)
		return nil
	})
}

func replaceHostInObject(o *pub.Object, repl pub.IRI) {
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

func replaceHostInActivity(o *pub.Activity, repl pub.IRI) {
	pub.OnIntransitiveActivity(o, func(o *pub.IntransitiveActivity) error {
		replaceHostInIntransitiveActivity(o, repl)
		return nil
	})
	replaceHostInItem(o.Object, repl)
}

func replaceHostInIntransitiveActivity(o *pub.IntransitiveActivity, repl pub.IRI) {
	pub.OnObject(o, func(o *pub.Object) error {
		replaceHostInObject(o, repl)
		return nil
	})
	replaceHostInItem(o.Actor, repl)
	replaceHostInItem(o.Target, repl)
	replaceHostInItem(o.Result, repl)
	replaceHostInItem(o.Origin, repl)
	replaceHostInItem(o.Instrument, repl)
}

func replaceHostInOrderedCollection (c *pub.OrderedCollection, repl pub.IRI) { }
func replaceHostInOrderedCollectionPage (c *pub.OrderedCollectionPage, repl pub.IRI) { }
func replaceHostInCollection (c *pub.Collection, repl pub.IRI) { }
func replaceHostInCollectionPage (c *pub.CollectionPage, repl pub.IRI) { }
func replaceHostInCollectionOfItems (c pub.ItemCollection, repl pub.IRI) {
	for _, it := range c {
		replaceHostInItem(it, repl)
	}
}

func replaceHostInItem(it pub.Item, repl pub.IRI) {
	if it == nil {
		return
	}
	if it.IsCollection() {
		if it.GetType() == pub.OrderedCollectionType {
			pub.OnOrderedCollection(it, func(c *pub.OrderedCollection) error {
				replaceHostInOrderedCollection(c, repl)
				return nil
			})
		}
		if it.GetType() == pub.OrderedCollectionPageType {
			pub.OnOrderedCollectionPage(it, func(c *pub.OrderedCollectionPage) error {
				replaceHostInOrderedCollectionPage(c, repl)
				return nil
			})
		}
		if it.GetType() == pub.CollectionType {
			pub.OnCollection(it, func(c *pub.Collection) error {
				replaceHostInCollection(c, repl)
				return nil
			})
		}
		if it.GetType() == pub.CollectionPageType {
			pub.OnCollectionPage(it, func(c *pub.CollectionPage) error {
				replaceHostInCollectionPage(c, repl)
				return nil
			})
		}
		if it.GetType() == pub.CollectionOfItems {
			pub.OnItemCollection(it, func(c *pub.ItemCollection) error {
				replaceHostInCollectionOfItems(*c, repl)
				return nil
			})
		}
	}
	if it.IsObject() {
		if pub.IntransitiveActivityTypes.Contains(it.GetType()) {
			pub.OnIntransitiveActivity(it, func(a *pub.IntransitiveActivity) error {
				replaceHostInIntransitiveActivity(a, repl)
				return nil
			})
		}
		if pub.ActivityTypes.Contains(it.GetType()) {
			pub.OnActivity(it, func(a *pub.Activity) error {
				replaceHostInActivity(a, repl)
				return nil
			})
		}
		if pub.ActorTypes.Contains(it.GetType()) {
			pub.OnActor(it, func(a *pub.Actor) error {
				replaceHostInActor(a, repl)
				return nil
			})
		}
		if pub.ObjectTypes.Contains(it.GetType()) {
			pub.OnObject(it, func(o *pub.Object) error {
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

