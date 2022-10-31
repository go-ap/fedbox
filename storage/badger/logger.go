//go:build storage_badger || storage_all || (!storage_pgx && !storage_boltdb && !storage_fs && !storage_sqlite)

package badger

import "git.sr.ht/~mariusor/lw"

type logger struct {
	lw.Logger
}

func (l logger) Warningf(s string, p ...interface{}) {
	l.Warnf(s, p...)
}
