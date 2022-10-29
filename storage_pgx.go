//go:build storage_pgx

package fedbox

import (
	"git.sr.ht/~mariusor/lw"
	auth "github.com/go-ap/auth/pgx"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/pgx"
	"github.com/openshift/osin"
)

func Storage(c config.Options, l lw.Logger) (processing.Store, osin.Storage, error) {
	// @todo(marius): we're no longer loading SQL db config env variables
	l.Debugf("Initializing pgx storage at %s", c.StoragePath)
	conf := pgx.Config{}
	db, err := pgx.New(conf, c.BaseURL, l)
	if err != nil {
		return nil, nil, errors.Annotatef(err, "unable to connect to pgx storage")
	}
	oauth := auth.New(authpgx.Config{
		Enabled: true,
		Host:    conf.Host,
		Port:    int64(conf.Port),
		User:    conf.User,
		Pw:      conf.Password,
		Name:    conf.Database,
		LogFn:   InfoLogFn(l),
		ErrFn:   ErrLogFn(l),
	})
	return db, oauth, errors.NotImplementedf("pgx storage is not implemented yet")
}
