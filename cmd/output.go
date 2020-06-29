package cmd

import (
	"bytes"
	"fmt"
	pub "github.com/go-ap/activitypub"
	"io"
	"time"
)

func bytef(s string, p ...interface{}) []byte {
	return []byte(fmt.Sprintf(s, p...))
}

func outObject(o *pub.Object, b io.Writer) error {
	b.Write(bytef("[%s] %s // %s", o.Type, o.ID, o.Published.Format(time.Stamp)))
	if o.Content != nil {
		b.Write(bytef("\n\tContent: %s", o.Content))
	}
	return nil
}

func outActivity(a *pub.Activity, b io.Writer) error {
	err := pub.OnObject(a, func(o *pub.Object) error {
		return outObject(o, b)
	})
	if err != nil {
		return err
	}
	if a.Actor != nil {
		b.Write(bytef("\tActor: "))
		outItem(a.Actor, b)
	}
	if a.Object != nil {
		b.Write(bytef("\n\tObject: "))
		outItem(a.Object, b)
	}
	
	return nil
}

func outActor(a *pub.Actor, b io.Writer) error {
	err := pub.OnObject(a, func(o *pub.Object) error {
		return outObject(o, b)
	})
	if err != nil {
		return err
	}
	b.Write(bytef("\n\tName: %s", a.Name))
	return nil
}
func outItem(it pub.Item, b io.Writer) error {
	if it.IsCollection() {
		return pub.OnCollectionIntf(it, func(c pub.CollectionInterface) error {
			for _, it := range c.Collection() {
				outItem(it, b)
				b.Write([]byte("\n"))
			}
			return nil
		})
	}
	if it.IsLink() {
		_, err := b.Write([]byte(it.GetLink()))
		return err
	}
	typ := it.GetType()
	if pub.ActivityTypes.Contains(typ) || pub.IntransitiveActivityTypes.Contains(typ) {
		return pub.OnActivity(it, func(a *pub.Activity) error {
			return outActivity(a, b)
		})
	}
	if pub.ActorTypes.Contains(typ) {
		return pub.OnActor(it, func(a *pub.Actor) error {
			return outActor(a, b)
		})
	}
	return pub.OnObject(it, func(o *pub.Object) error {
		return outObject(o, b)
	})
}

func outText(it pub.Item) error {
	b := new(bytes.Buffer)
	err := outItem(it, b)
	if err != nil {
		return err
	}
	fmt.Printf("%s", b.Bytes())
	return nil
}

func outJSON(it pub.Item) error {
	out, err := pub.MarshalJSON(it)
	if err != nil {
		return err
	}
	fmt.Printf("%s", out)
	return nil
}
