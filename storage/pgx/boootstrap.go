package pgx

import (
	"github.com/go-ap/fedbox/internal/config"
	"github.com/jackc/pgx"
	"github.com/sirupsen/logrus"
)

var RootDb = "postgres"

func Bootstrap(conf config.BackendConfig, rootUser string, rootPw []byte) error {
	log := logrus.New()
	conn, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			Host:     conf.Host,
			Port:     uint16(conf.Port),
			Database: RootDb,
			User:     rootUser,
			Password: string(rootPw),
			Logger:   DBLogger(log),
		},
		MaxConnections: 3,
	})
	if err != nil {
		return err
	}
	var cur string
	rows, err := conn.Query("select current_user;")
	defer rows.Close()
	if err != nil {
		return err
	}
	for rows.Next() {
		err = rows.Scan(&cur)
		if err != nil {
			return err
		}
	}
	log.Infof("Current pg user %s\n", cur)
	return nil
}

func Clean(c config.BackendConfig) error {
	return nil
}
