package config

import (
	"fmt"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/joho/godotenv"
	"os"
	"path"
	"strconv"
	"strings"
)

var Prefix = ""

type BackendConfig struct {
	Enabled bool
	Host    string
	Port    int64
	User    string
	Pw      string
	Name    string
}

type Options struct {
	Env        env.Type
	LogLevel   log.Level
	Secure     bool
	Host       string
	Listen     string
	BaseURL    string
	Storage    StorageType
	DB         BackendConfig
	BoltDBPath string
}

type StorageType string

const (
	KeyENV      = "ENV"
	KeyLogLevel = "LOG_LEVEL"
	KeyHostname = "HOSTNAME"
	KeyHTTPS    = "HTTPS"
	KeyListen   = "LISTEN"
	KeyStorage  = "STORAGE"
	KeyDBHost   = "DB_HOST"
	KeyDBPort   = "DB_PORT"
	KeyDBName   = "DB_NAME"
	KeyDBUser   = "DB_USER"
	KeyDBPw     = "DB_PASSWORD"
	BoltDB      = StorageType("boltdb")
	Postgres    = StorageType("postgres")
)

func k(k string) string {
	if Prefix != "" {
		return fmt.Sprintf("%s_%s", Prefix, k)
	}
	return k
}

func LoadFromEnv(e string) (Options, error) {
	conf := Options{}
	if !env.ValidType(e) {
		e = os.Getenv(k(KeyENV))
	}

	conf.Env = env.ValidTypeOrDev(e)
	configs := []string{
		".env",
		fmt.Sprintf(".env.%s", conf.Env),
	}

	lvl := os.Getenv(k(KeyLogLevel))
	switch strings.ToLower(lvl) {
	case "trace":
		conf.LogLevel = log.TraceLevel
	case "debug":
		conf.LogLevel = log.DebugLevel
	case "warn":
		conf.LogLevel = log.WarnLevel
	case "error":
		conf.LogLevel = log.ErrorLevel
	case "info":
		fallthrough
	default:
		conf.LogLevel = log.InfoLevel
	}

	for _, f := range configs {
		godotenv.Overload(f)
	}

	if conf.Host == "" {
		conf.Host = os.Getenv(k(KeyHostname))
	}
	conf.Secure, _ = strconv.ParseBool(os.Getenv(k(KeyHTTPS)))
	if conf.Secure {
		conf.BaseURL = fmt.Sprintf("https://%s", conf.Host)
	} else {
		conf.BaseURL = fmt.Sprintf("http://%s", conf.Host)
	}

	conf.Listen = os.Getenv(k(KeyListen))

	envStorage := os.Getenv(k(KeyStorage))
	conf.Storage = StorageType(strings.ToLower(envStorage))
	switch conf.Storage {
	case BoltDB:
		conf.BoltDBPath = fmt.Sprintf("%s/%s-%s.bdb", os.TempDir(), path.Clean(conf.Host), conf.Env)
	case "":
		conf.Storage = Postgres
		fallthrough
	case Postgres:
		conf.DB.Host = os.Getenv(k(KeyDBHost))
		conf.DB.Pw = os.Getenv(k(KeyDBPw))
		conf.DB.Name = os.Getenv(k(KeyDBName))
		var err error
		if conf.DB.Port, err = strconv.ParseInt(os.Getenv(k(KeyDBPort)), 10, 32); err != nil {
			conf.DB.Port = 5432
		}

		conf.DB.User = os.Getenv(k(KeyDBUser))

	default:
		return conf, errors.Errorf("Invalid STORAGE value %s", envStorage)
	}

	return conf, nil
}
