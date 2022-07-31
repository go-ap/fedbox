//go:build !cgo

package sqlite

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

var sqlOpen = sql.Open
