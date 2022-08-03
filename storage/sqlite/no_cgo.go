//go:build !cgo

package sqlite

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

var sqlOpen = func(dataSourceName string) (*sql.DB, error) {
	return sql.Open("sqlite", dataSourceName)
}
