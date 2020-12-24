// +build storage_pgx

package app

import (
	auth "github.com/go-ap/auth/pgx"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/pgx"
)

func Storage(c config.Options, l logrus.FieldLogger) (st.Repository, osin.Storage, error) {
	// @todo(marius): we're no longer loading SQL db config env variables
	conf := config.BackendConfig{}
	db, err := pgx.New(conf, c.BaseURL, l)

	oauth := auth.New(auth.Config{
		Enabled: true,
		Host:    conf.Host,
		Port:    conf.Port,
		User:    conf.User,
		Pw:      conf.Pw,
		Name:    conf.Name,
		LogFn:   InfoLogFn(l),
		ErrFn:   ErrLogFn(l),
	})
	return db, oauth, err
}
