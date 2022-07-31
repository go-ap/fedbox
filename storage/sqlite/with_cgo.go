//go:build cgo

package sqlite

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

var sqlOpen = sql.Open
