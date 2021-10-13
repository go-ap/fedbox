package cmd

import (
	"fmt"

	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/urfave/cli/v2"
)

func types(c *cli.Context) ap.CompStrs {
	if c == nil {
		return nil
	}
	typ := c.StringSlice("type")
	types := make(ap.CompStrs, 0)
	for _, t := range typ {
		tt := pub.ActivityVocabularyType(t)
		if pub.Types.Contains(tt) {
			types = append(types, ap.StringEquals(string(tt)))
		}
	}
	return types
}

func FilterFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "path",
			Usage: "Pass the path at which to start.",
			Value: "/",
		},
		&cli.StringSliceFlag{
			Name:        "type",
			Usage:       fmt.Sprintf("The type of activitypub object to list"),
			DefaultText: fmt.Sprintf("Valid values: %v", ValidGenericTypes),
		},
		&cli.StringSliceFlag{
			Name:  "name",
			Usage: fmt.Sprintf("The name/preferredName of the activitypub object to list"),
		},
		&cli.StringSliceFlag{
			Name:  "cont",
			Usage: fmt.Sprintf("The content of the activitypub object to list"),
		},
		&cli.StringSliceFlag{
			Name:  "to",
			Usage: fmt.Sprintf("The to recipients of the activitypub object to list"),
		},
		&cli.StringSliceFlag{
			Name:  "cc",
			Usage: fmt.Sprintf("The cc recipients of the activitypub object to list"),
		},
		&cli.StringSliceFlag{
			Name:  "author",
			Usage: fmt.Sprintf("The author of the activitypub object to list"),
		},
	}
	/*
		baseURL       pub.IRI                     `qstring:"-"`
		Name          CompStrs                    `qstring:"name,omitempty"`
		Cont          CompStrs                    `qstring:"content,omitempty"`
		Authenticated *pub.Actor                  `qstring:"-"`
		To            *pub.Actor                  `qstring:"-"`
		Author        *pub.Actor                  `qstring:"-"`
		Parent        *pub.Actor                  `qstring:"-"`
		IRI           pub.IRI                     `qstring:"-"`
		Collection    h.CollectionType            `qstring:"-"`
		URL           CompStrs                    `qstring:"url,omitempty"`
		MedTypes      []pub.MimeType              `qstring:"mediaType,omitempty"`
		Aud           CompStrs                    `qstring:"recipients,omitempty"`
		Gen           CompStrs                    `qstring:"generator,omitempty"`
		Key           []Hash                      `qstring:"-"`
		ItemKey       CompStrs                    `qstring:"iri,omitempty"`
		Type          pub.ActivityVocabularyTypes `qstring:"type,omitempty"`
		AttrTo        CompStrs                    `qstring:"attributedTo,omitempty"`
		InReplTo      CompStrs                    `qstring:"inReplyTo,omitempty"`
		OP            CompStrs                    `qstring:"context,omitempty"`
		FollowedBy    []Hash                      `qstring:"followedBy,omitempty"` // todo(marius): not really used
		OlderThan     time.Time                   `qstring:"olderThan,omitempty"`
		NewerThan     time.Time                   `qstring:"newerThan,omitempty"`
		Prev          Hash                        `qstring:"before,omitempty"`
		Next          Hash                        `qstring:"after,omitempty"`
		Object        *Filters                    `qstring:"object,omitempty"`
		Actor         *Filters                    `qstring:"actor,omitempty"`
		Target        *Filters                    `qstring:"target,omitempty"`
		CurPage       uint                        `qstring:"page,omitempty"`
		MaxItems      uint                        `qstring:"maxItems,omitempty"`
	*/
}

func LoadFilters(c *cli.Context) (*ap.Filters, error) {
	if c == nil {
		return nil, errors.Newf("invalid nil context")
	}
	f := new(ap.Filters)
	f.Type = types(c)
	return f, nil
}
