package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/joho/godotenv"
)

var Prefix = "fedbox"

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
	LogOutput   string
	TimeOut     time.Duration
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
	KeyTimeOut      = "TIME_OUT"
	KeyLogLevel     = "LOG_LEVEL"
	KeyLogOutput    = "LOG_OUTPUT"
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
	StorageSqlite   = StorageType("sqlite")
)

const defaultPerm = os.ModeDir | os.ModePerm | 0700

func (o Options) BaseStoragePath() string {
	if !filepath.IsAbs(o.StoragePath) {
		o.StoragePath, _ = filepath.Abs(o.StoragePath)
	}
	basePath := path.Clean(path.Join(o.StoragePath, string(o.Storage), string(o.Env), o.Host))
	fi, err := os.Stat(basePath)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(basePath, defaultPerm)
	}
	if err != nil {
		panic(err)
	}
	fi, err = os.Stat(basePath)
	if !fi.IsDir() {
		panic(errors.NotValidf("path %s is invalid for storage", basePath))
	}
	return basePath
}

func (o Options) BoltDBOAuth2() string {
	return fmt.Sprintf("%s/oauth.bdb", o.BaseStoragePath())
}

func (o Options) BadgerOAuth2(base string) string {
	return path.Join(path.Dir(base), "oauth", path.Base(base))
}

func prefKey(k string) string {
	if Prefix != "" {
		return fmt.Sprintf("%s_%s", strings.ToUpper(Prefix), k)
	}
	return k
}

func loadKeyFromEnv(name, def string) string {
	if val := os.Getenv(prefKey(name)); len(val) > 0 {
		return val
	}
	if val := os.Getenv(name); len(val) > 0 {
		return val
	}
	return def
}

func LoadFromEnv(e env.Type, timeOut time.Duration) (Options, error) {
	conf := Options{}
	if !env.ValidType(e) {
		e = env.Type(loadKeyFromEnv(KeyENV, ""))
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
	for _, f := range configs {
		godotenv.Overload(f)
	}

	lvl := loadKeyFromEnv(KeyLogLevel, "")
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
	conf.LogOutput = loadKeyFromEnv(KeyLogOutput, "")

	if !env.ValidType(e) {
		e = env.Type(loadKeyFromEnv(KeyENV, "dev"))
	}
	conf.Env = e
	if conf.Host == "" {
		conf.Host = loadKeyFromEnv(KeyHostname, conf.Host)
	}
	conf.TimeOut = timeOut
	if to, _ := time.ParseDuration(loadKeyFromEnv(KeyTimeOut, "")); to > 0 {
		conf.TimeOut = to
	}
	conf.Secure, _ = strconv.ParseBool(loadKeyFromEnv(KeyHTTPS, "false"))
	if conf.Secure {
		conf.BaseURL = fmt.Sprintf("https://%s", conf.Host)
	} else {
		conf.BaseURL = fmt.Sprintf("http://%s", conf.Host)
	}
	conf.KeyPath = loadKeyFromEnv(KeyKeyPath, "")
	conf.CertPath = loadKeyFromEnv(KeyCertPath, "")

	conf.Listen = loadKeyFromEnv(KeyListen, "")
	envStorage := loadKeyFromEnv(KeyStorage, string(DefaultStorage))
	if len(DefaultStorage) > 0 {
		envStorage = string(DefaultStorage)
	}
	conf.Storage = StorageType(strings.ToLower(envStorage))
	conf.StoragePath = loadKeyFromEnv(KeyStoragePath, "")
	if conf.StoragePath == "" {
		conf.StoragePath = os.TempDir()
	}
	conf.StoragePath = path.Clean(conf.StoragePath)

	return conf, nil
}
