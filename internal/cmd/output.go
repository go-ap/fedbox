package cmd

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	vocab "github.com/go-ap/activitypub"
)

func bytef(s string, p ...interface{}) []byte {
	return []byte(fmt.Sprintf(s, p...))
}

func outObject(o *vocab.Object, b io.Writer) error {
	b.Write(bytef("[%s] %s // %s", o.Type, o.ID, o.Published.Format("02 Jan 2006 15:04:05")))
	if len(o.Name) > 0 {
		for _, s := range o.Name {
			ss := strings.Trim(s.Value.String(), "\n\r\t ")
			if s.Ref != vocab.NilLangRef {
				b.Write(bytef("\n\tName[%s]: %s", s.Ref, ss))
			}
			b.Write(bytef("\n\tName: %s", ss))
		}
	}
	if o.Summary != nil {
		for _, s := range o.Summary {
			ss := strings.Trim(s.Value.String(), "\n\r\t ")
			if s.Ref != vocab.NilLangRef {
				cont := s.Ref
				if len(cont) > 72 {
					cont = cont[:72]
				}
				b.Write(bytef("\n\tSummary[%s]: %s", cont, ss))
			}
			b.Write(bytef("\n\tSummary: %s", ss))
		}
	}
	if o.Content != nil {
		for _, c := range o.Content {
			cc := strings.Trim(c.Value.String(), "\n\r\t ")
			if c.Ref != vocab.NilLangRef {
				cont := c.Ref
				if len(cont) > 72 {
					cont = cont[:72]
				}
				b.Write(bytef("\n\tContent[%s]: %s", cont, cc))
			}
			b.Write(bytef("\n\tContent: %s", cc))
		}
	}
	return nil
}

func outActivity(a *vocab.Activity, b io.Writer) error {
	err := vocab.OnObject(a, func(o *vocab.Object) error {
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

func outActor(a *vocab.Actor, b io.Writer) error {
	err := vocab.OnObject(a, func(o *vocab.Object) error {
		return outObject(o, b)
	})
	if err != nil {
		return err
	}
	if len(a.PreferredUsername) > 0 {
		for _, s := range a.PreferredUsername {
			ss := strings.Trim(s.Value.String(), "\n\r\t ")
			if s.Ref != vocab.NilLangRef {
				b.Write(bytef("\n\tPreferredUsername[%s]: %s", s.Ref, ss))
			}
			b.Write(bytef("\n\tPreferredUsername: %s", ss))
		}
	}
	return nil
}
func outItem(it vocab.Item, b io.Writer) error {
	if it.IsCollection() {
		return vocab.OnCollectionIntf(it, func(c vocab.CollectionInterface) error {
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
	if vocab.ActivityTypes.Contains(typ) || vocab.IntransitiveActivityTypes.Contains(typ) {
		return vocab.OnActivity(it, func(a *vocab.Activity) error {
			return outActivity(a, b)
		})
	}
	if vocab.ActorTypes.Contains(typ) {
		return vocab.OnActor(it, func(a *vocab.Actor) error {
			return outActor(a, b)
		})
	}
	return vocab.OnObject(it, func(o *vocab.Object) error {
		return outObject(o, b)
	})
}

func outText(it vocab.Item) error {
	b := new(bytes.Buffer)
	err := outItem(it, b)
	if err != nil {
		return err
	}
	fmt.Printf("%s", b.Bytes())
	return nil
}

func outJSON(it vocab.Item) error {
	out, err := vocab.MarshalJSON(it)
	if err != nil {
		return err
	}
	fmt.Printf("%s", out)
	return nil
}
