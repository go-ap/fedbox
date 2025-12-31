package fedbox

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	vocab "github.com/go-ap/activitypub"
)

func OutItem(it vocab.Item, b io.Writer) error {
	if it.IsCollection() {
		return vocab.OnCollectionIntf(it, func(c vocab.CollectionInterface) error {
			for _, it := range c.Collection() {
				_ = OutItem(it, b)
				_, _ = b.Write([]byte("\n"))
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

func OutText(where io.Writer) func(it vocab.Item) error {
	return func(it vocab.Item) error {
		b := new(bytes.Buffer)
		err := OutItem(it, b)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(where, "%s", b.Bytes())
		return nil
	}
}

func OutJSON(where io.Writer) func(it vocab.Item) error {
	return func(it vocab.Item) error {
		out, err := vocab.MarshalJSON(it)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(where, "%s", out)
		return nil
	}
}
func outObject(o *vocab.Object, b io.Writer) error {
	_, _ = b.Write(bytef("[%s] %s // %s", o.Type, o.ID, o.Published.Format("02 Jan 2006 15:04:05")))
	if len(o.Name) > 0 {
		for ref, s := range o.Name {
			ss := strings.Trim(s.String(), "\n\r\t ")
			if ref != vocab.NilLangRef {
				_, _ = b.Write(bytef("\n\tName[%s]: %s", ref, ss))
			}
			_, _ = b.Write(bytef("\n\tName: %s", ss))
		}
	}
	if o.Summary != nil {
		for ref, s := range o.Summary {
			ss := strings.Trim(s.String(), "\n\r\t ")
			if ref != vocab.NilLangRef {
				cont := ref
				_, _ = b.Write(bytef("\n\tSummary[%s]: %s", cont, ss))
			}
			_, _ = b.Write(bytef("\n\tSummary: %s", ss))
		}
	}
	if o.Content != nil {
		for ref, c := range o.Content {
			cc := strings.Trim(c.String(), "\n\r\t ")
			if ref != vocab.NilLangRef {
				cont := ref
				_, _ = b.Write(bytef("\n\tContent[%s]: %s", cont, cc))
			}
			_, _ = b.Write(bytef("\n\tContent: %s", cc))
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
		OutItem(a.Actor, b)
	}
	if a.Object != nil {
		b.Write(bytef("\n\tObject: "))
		OutItem(a.Object, b)
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
		for ref, s := range a.PreferredUsername {
			ss := strings.Trim(s.String(), "\n\r\t ")
			if ref != vocab.NilLangRef {
				b.Write(bytef("\n\tPreferredUsername[%s]: %s", ref, ss))
			}
			b.Write(bytef("\n\tPreferredUsername: %s", ss))
		}
	}
	return nil
}

func bytef(s string, p ...any) []byte {
	return []byte(fmt.Sprintf(s, p...))
}

func printItem(it vocab.Item, outType string) error {
	if outType == "json" {
		return OutJSON(os.Stdout)(it)
	}
	return OutText(os.Stdout)(it)
}
