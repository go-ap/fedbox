package cmd

import (
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/filters"
)

func types(incoming ...string) filters.Check {
	validTypes := make(vocab.ActivityVocabularyTypes, 0, len(incoming))
	for _, t := range incoming {
		if tt := vocab.ActivityVocabularyType(t); vocab.Types.Contains(tt) {
			validTypes = append(validTypes)
		}
	}
	return filters.HasType(validTypes...)
}

func names(incoming ...string) filters.Check {
	checks := make(filters.Checks, 0, len(incoming))
	for _, t := range incoming {
		checks = append(checks, filters.NameIs(t))
	}
	return filters.Any(checks...)
}
