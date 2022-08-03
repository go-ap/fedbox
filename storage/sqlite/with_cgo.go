//go:build cgo

package sqlite

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

var sqlOpen = func(dataSourceName string) (*sql.DB, error) {
	return sql.Open("sqlite3", dataSourceName)
}
