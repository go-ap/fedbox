package config

import (
	"fmt"
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
	Env         env.Type
	LogLevel    log.Level
	Secure      bool
	CertPath    string
	KeyPath     string
	Host        string
	Listen      string
	BaseURL     string
	Storage     StorageType
	StoragePath string
}

type StorageType string

const (
	KeyENV          = "ENV"
	KeyLogLevel     = "LOG_LEVEL"
	KeyHostname     = "HOSTNAME"
	KeyHTTPS        = "HTTPS"
	KeyCertPath     = "CERT_PATH"
	KeyKeyPath      = "KEY_PATH"
	KeyListen       = "LISTEN"
	KeyDBHost       = "DB_HOST"
	KeyDBPort       = "DB_PORT"
	KeyDBName       = "DB_NAME"
	KeyDBUser       = "DB_USER"
	KeyDBPw         = "DB_PASSWORD"
	KeyStorage      = "STORAGE"
	KeyStoragePath  = "STORAGE_PATH"
	StorageBoltDB   = StorageType("boltdb")
	StorageFS       = StorageType("fs")
	StorageBadger   = StorageType("badger")
	StoragePostgres = StorageType("postgres")
)

func clean(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(path.Clean(name), ".", "-"), ":", "-")
}

func GetDBPath(dir, file string, env env.Type) string {
	return fmt.Sprintf("%s/%s-%s.bdb", dir, clean(file), env)
}

func (o Options) BoltDB() string {
	return GetDBPath(o.StoragePath, o.Host, o.Env)
}

func (o Options) BoltDBOAuth2() string {
	return GetDBPath(o.StoragePath, fmt.Sprintf("%s-oauth", o.Host), o.Env)
}

func (o Options) Badger() string {
	return fmt.Sprintf("%s/%s/%s", o.StoragePath, o.Env, o.Host)
}

func (o Options) BadgerOAuth2() string {
	return fmt.Sprintf("%s/%s/%s", o.StoragePath, o.Env, "oauth")
}

func k(k string) string {
	if Prefix != "" {
		return fmt.Sprintf("%s_%s", Prefix, k)
	}
	return k
}

func LoadFromEnv(e env.Type) (Options, error) {
	conf := Options{}
	if !env.ValidType(e) {
		e = env.Type(os.Getenv(k(KeyENV)))
	}
	configs := []string{
		".env",
	}
	appendIfFile := func(typ env.Type) {
		envFile := fmt.Sprintf(".env.%s", typ)
		if _, err := os.Stat(envFile); err == nil {
			configs = append(configs, envFile)
		}
	}
	if !env.ValidType(e) {
		for _, typ := range env.Types {
			appendIfFile(typ)
		}
	} else {
		appendIfFile(e)
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

	if !env.ValidType(e) {
		e = env.Type(os.Getenv(k(KeyENV)))
	}
	conf.Env = e
	if conf.Host == "" {
		conf.Host = os.Getenv(k(KeyHostname))
	}
	conf.Secure, _ = strconv.ParseBool(os.Getenv(k(KeyHTTPS)))
	if conf.Secure {
		conf.BaseURL = fmt.Sprintf("https://%s", conf.Host)
	} else {
		conf.BaseURL = fmt.Sprintf("http://%s", conf.Host)
	}
	conf.KeyPath = os.Getenv(k(KeyKeyPath))
	conf.CertPath = os.Getenv(k(KeyCertPath))

	conf.Listen = os.Getenv(k(KeyListen))
	envStorage := os.Getenv(k(KeyStorage))
	conf.Storage = StorageType(strings.ToLower(envStorage))
	conf.StoragePath = os.Getenv(k(KeyStoragePath))
	if conf.StoragePath == "" {
		conf.StoragePath = os.TempDir()
	}
	conf.StoragePath = path.Clean(conf.StoragePath)

	return conf, nil
}
