package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/env"
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
	Env          env.Type
	LogLevel     lw.Level
	LogOutput    string
	TimeOut      time.Duration
	Secure       bool
	CertPath     string
	KeyPath      string
	Host         string
	Listen       string
	BaseURL      string
	Storage      StorageType
	StoragePath  string
	StorageCache bool
	RequestCache bool
	Profile      bool
}

type StorageType string

const (
	KeyENV                 = "ENV"
	KeyTimeOut             = "TIME_OUT"
	KeyLogLevel            = "LOG_LEVEL"
	KeyLogOutput           = "LOG_OUTPUT"
	KeyHostname            = "HOSTNAME"
	KeyHTTPS               = "HTTPS"
	KeyCertPath            = "CERT_PATH"
	KeyKeyPath             = "KEY_PATH"
	KeyListen              = "LISTEN"
	KeyDBHost              = "DB_HOST"
	KeyDBPort              = "DB_PORT"
	KeyDBName              = "DB_NAME"
	KeyDBUser              = "DB_USER"
	KeyDBPw                = "DB_PASSWORD"
	KeyStorage             = "STORAGE"
	KeyStoragePath         = "STORAGE_PATH"
	KeyCacheDisable        = "DISABLE_CACHE"
	KeyStorageCacheDisable = "DISABLE_STORAGE_CACHE"
	KeyRequestCacheDisable = "DISABLE_REQUEST_CACHE"
	StorageBoltDB          = StorageType("boltdb")
	StorageFS              = StorageType("fs")
	StorageBadger          = StorageType("badger")
	StoragePostgres        = StorageType("postgres")
	StorageSqlite          = StorageType("sqlite")
)

const defaultDirPerm = os.ModeDir | os.ModePerm | 0700

func (o Options) BaseStoragePath() string {
	if !filepath.IsAbs(o.StoragePath) {
		o.StoragePath, _ = filepath.Abs(o.StoragePath)
	}
	basePath := path.Clean(path.Join(o.StoragePath, string(o.Storage), string(o.Env), o.Host))
	fi, err := os.Stat(basePath)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(basePath, defaultDirPerm)
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

func Getval(name, def string) string {
	val := def
	if pf := os.Getenv(prefKey(name)); len(pf) > 0 {
		val = pf
	}
	if p := os.Getenv(name); len(p) > 0 {
		val = p
	}
	return val
}

func LoadFromEnv(e env.Type, timeOut time.Duration) (Options, error) {
	conf := Options{}
	if !env.ValidType(e) {
		e = env.Type(Getval(KeyENV, ""))
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
		godotenv.Load(f)
	}

	lvl := Getval(KeyLogLevel, "")
	switch strings.ToLower(lvl) {
	case "none":
		conf.LogLevel = lw.NoLevel
	case "trace":
		conf.LogLevel = lw.TraceLevel
	case "debug":
		conf.LogLevel = lw.DebugLevel
	case "warn":
		conf.LogLevel = lw.WarnLevel
	case "error":
		conf.LogLevel = lw.ErrorLevel
	case "info":
		fallthrough
	default:
		conf.LogLevel = lw.InfoLevel
	}
	conf.LogOutput = Getval(KeyLogOutput, "")

	if !env.ValidType(e) {
		e = env.Type(Getval(KeyENV, "dev"))
	}
	conf.Env = e
	if conf.Host == "" {
		conf.Host = Getval(KeyHostname, conf.Host)
	}
	conf.TimeOut = timeOut
	if to, _ := time.ParseDuration(Getval(KeyTimeOut, "")); to > 0 {
		conf.TimeOut = to
	}
	conf.Secure, _ = strconv.ParseBool(Getval(KeyHTTPS, "false"))
	if conf.Secure {
		conf.BaseURL = fmt.Sprintf("https://%s", conf.Host)
	} else {
		conf.BaseURL = fmt.Sprintf("http://%s", conf.Host)
	}
	conf.KeyPath = Getval(KeyKeyPath, "")
	conf.CertPath = Getval(KeyCertPath, "")

	conf.Listen = Getval(KeyListen, "")
	envStorage := Getval(KeyStorage, string(DefaultStorage))
	if len(DefaultStorage) > 0 {
		envStorage = string(DefaultStorage)
	}
	conf.Storage = StorageType(strings.ToLower(envStorage))
	conf.StoragePath = Getval(KeyStoragePath, "")
	if conf.StoragePath == "" {
		conf.StoragePath = os.TempDir()
	}
	conf.StoragePath = path.Clean(conf.StoragePath)

	disableCache, _ := strconv.ParseBool(Getval(KeyCacheDisable, "false"))
	conf.StorageCache = !disableCache
	conf.RequestCache = !disableCache
	if disableStorageCache, err := strconv.ParseBool(Getval(KeyStorageCacheDisable, "false")); err == nil {
		conf.StorageCache = !disableStorageCache
	}
	if disableRequestCache, err := strconv.ParseBool(Getval(KeyRequestCacheDisable, "false")); err == nil {
		conf.RequestCache = !disableRequestCache
	}

	return conf, nil
}
