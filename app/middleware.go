package app

import (
	"context"
	"github.com/go-ap/activitypub/storage"
	ctxt "github.com/go-ap/fedbox/internal/context"
	"github.com/go-ap/fedbox/internal/errors"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/sirupsen/logrus"
	"net/http"
)

func Repo(loader storage.Loader) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			newCtx := context.WithValue(ctx, ctxt.RepositoryKey, loader)
			next.ServeHTTP(w, r.WithContext(newCtx))
		}
		return http.HandlerFunc(fn)
	}
}

func ActorFromAuthHeader(next http.Handler) http.Handler {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := log.New()
		if acct, err := loadActorFromAuthHeader(w, r, logger); err == nil {
			logger.WithFields(logrus.Fields{
				"id": acct.GetID(),
			}).Infof("Loaded actor")
		} else {
			logger.Warnf("%s", err)
		}
		next.ServeHTTP(w, r)
	})
	return http.HandlerFunc(fn)
}
