package cmd

import (
	"bytes"
	"fmt"
	pub "github.com/go-ap/activitypub"
	"io"
	"strings"
	"time"
)

func bytef(s string, p ...interface{}) []byte {
	return []byte(fmt.Sprintf(s, p...))
}

func outObject(o *pub.Object, b io.Writer) error {
	b.Write(bytef("[%s] %s // %s", o.Type, o.ID, o.Published.Format(time.Stamp)))
	if len(o.Name) > 0 {
		for _, s := range o.Name {
			ss := strings.Trim(s.Value, "\n\r\t ")
			if s.Ref != pub.NilLangRef {
				b.Write(bytef("\n\tName[%s]: %s", s.Ref, ss))
			}
			b.Write(bytef("\n\tName: %s", ss))
		}
	}
	if o.Summary != nil {
		for _, s := range o.Summary {
			ss := strings.Trim(s.Value, "\n\r\t ")
			if s.Ref != pub.NilLangRef {
				b.Write(bytef("\n\tSummary[%s]: %s", s.Ref, ss))
			}
			b.Write(bytef("\n\tSummary: %s", ss))
		}
	}
	if o.Content != nil {
		for _, c := range o.Content {
			cc := strings.Trim(c.Value, "\n\r\t ")
			if c.Ref != pub.NilLangRef {
				b.Write(bytef("\n\tContent[%s]: %s", c.Ref, cc))
			}
			b.Write(bytef("\n\tContent: %s", cc))
		}
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
		b.Write(bytef("\n\tActor: "))
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
	if len(a.PreferredUsername) > 0 {
		for _, s := range a.PreferredUsername {
			ss := strings.Trim(s.Value, "\n\r\t ")
			if s.Ref != pub.NilLangRef {
				b.Write(bytef("\n\tPreferredUsername[%s]: %s", s.Ref, ss))
			}
			b.Write(bytef("\n\tPreferredUsername: %s", ss))
		}
	}
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
