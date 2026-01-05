package ap

import (
	"time"

	vocab "github.com/go-ap/activitypub"
)

func wrapInActivity(p vocab.Item, author vocab.Item, typ vocab.ActivityVocabularyType) vocab.Activity {
	act := vocab.Activity{
		Type:    typ,
		To:      vocab.ItemCollection{vocab.PublicNS},
		Updated: time.Now().Truncate(time.Second).UTC(),
		Object:  p,
	}
	if act.AttributedTo == nil {
		act.AttributedTo = author.GetLink()
	}
	if act.Actor == nil {
		act.Actor = author
	}
	_ = act.CC.Append(author.GetLink(), vocab.Followers.Of(author))
	return act
}

func WrapObjectInUpdate(p vocab.Item, author vocab.Item) vocab.Activity {
	return wrapInActivity(p, author, vocab.UpdateType)
}

func WrapObjectInCreate(p vocab.Item, author vocab.Item) vocab.Activity {
	return wrapInActivity(p, author, vocab.CreateType)
}
