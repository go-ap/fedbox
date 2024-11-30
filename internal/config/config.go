package config

import (
	"fmt"
	"net/url"
	"os"
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
	Env                env.Type
	LogLevel           lw.Level
	LogOutput          string
	TimeOut            time.Duration
	Secure             bool
	CertPath           string
	KeyPath            string
	Host               string
	Listen             string
	BaseURL            string
	Storage            StorageType
	StoragePath        string
	StorageCache       bool
	RequestCache       bool
	UseIndex           bool
	Profile            bool
	MastodonCompatible bool
}

type StorageType string

const (
	KeyENV                          = "ENV"
	KeyTimeOut                      = "TIME_OUT"
	KeyLogLevel                     = "LOG_LEVEL"
	KeyLogOutput                    = "LOG_OUTPUT"
	KeyHostname                     = "HOSTNAME"
	KeyHTTPS                        = "HTTPS"
	KeyCertPath                     = "CERT_PATH"
	KeyKeyPath                      = "KEY_PATH"
	KeyListen                       = "LISTEN"
	KeyDBHost                       = "DB_HOST"
	KeyDBPort                       = "DB_PORT"
	KeyDBName                       = "DB_NAME"
	KeyDBUser                       = "DB_USER"
	KeyDBPw                         = "DB_PASSWORD"
	KeyStorage                      = "STORAGE"
	KeyStoragePath                  = "STORAGE_PATH"
	KeyCacheDisable                 = "DISABLE_CACHE"
	KeyStorageCacheDisable          = "DISABLE_STORAGE_CACHE"
	KeyRequestCacheDisable          = "DISABLE_REQUEST_CACHE"
	KeyStorageIndexDisable          = "DISABLE_STORAGE_INDEX"
	KeyMastodonCompatibilityDisable = "DISABLE_MASTODON_COMPATIBILITY"

	varEnv     = "%env%"
	varStorage = "%storage%"
	varHost    = "%host%"

	StorageBoltDB   = StorageType("boltdb")
	StorageFS       = StorageType("fs")
	StorageBadger   = StorageType("badger")
	StoragePostgres = StorageType("postgres")
	StorageSqlite   = StorageType("sqlite")
)

const defaultDirPerm = os.ModeDir | os.ModePerm | 0700

func normalizeConfigPath(p string, o Options) string {
	if len(p) == 0 {
		return p
	}
	if p[0] == '~' {
		p = os.Getenv("HOME") + p[1:]
	}
	if !filepath.IsAbs(p) {
		p, _ = filepath.Abs(p)
	}
	p = strings.ReplaceAll(p, varEnv, string(o.Env))
	p = strings.ReplaceAll(p, varStorage, string(o.Storage))
	p = strings.ReplaceAll(p, varHost, url.PathEscape(o.Host))
	return filepath.Clean(p)
}

func (o Options) BaseStoragePath() string {
	basePath := normalizeConfigPath(o.StoragePath, o)
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

func prefKey(k string) string {
	if Prefix == "" {
		return k
	}
	return strings.Join([]string{strings.ToUpper(Prefix), k}, "_")
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

func Load(e env.Type, timeOut time.Duration) (Options, error) {
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
		_ = godotenv.Load(f)
	}

	opts := LoadFromEnv()
	opts.Env = e
	opts.TimeOut = timeOut

	return opts, nil
}

func LoadFromEnv() Options {
	conf := Options{}
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

	conf.Env = env.Type(Getval(KeyENV, "dev"))
	if conf.Host == "" {
		conf.Host = Getval(KeyHostname, conf.Host)
	}
	conf.TimeOut = 0
	if to, _ := time.ParseDuration(Getval(KeyTimeOut, "")); to > 0 {
		conf.TimeOut = to
	}
	conf.Secure, _ = strconv.ParseBool(Getval(KeyHTTPS, "false"))
	if conf.Secure {
		conf.BaseURL = fmt.Sprintf("https://%s", conf.Host)
	} else {
		conf.BaseURL = fmt.Sprintf("http://%s", conf.Host)
	}

	conf.Listen = Getval(KeyListen, "")
	conf.Storage = StorageType(strings.ToLower(Getval(KeyStorage, string(DefaultStorage))))
	conf.StoragePath = Getval(KeyStoragePath, "")
	if conf.StoragePath == "" {
		conf.StoragePath = os.TempDir()
	}
	conf.StoragePath = filepath.Clean(conf.StoragePath)

	disableCache, _ := strconv.ParseBool(Getval(KeyCacheDisable, "false"))
	conf.StorageCache = !disableCache
	conf.RequestCache = !disableCache

	if v := Getval(KeyStorageCacheDisable, ""); v != "" {
		disableStorageCache, _ := strconv.ParseBool(v)
		conf.StorageCache = !disableStorageCache
	}

	if v := Getval(KeyRequestCacheDisable, ""); v != "" {
		disableRequestCache, _ := strconv.ParseBool(v)
		conf.RequestCache = !disableRequestCache
	}

	if v := Getval(KeyStorageIndexDisable, "false"); v != "" {
		disableStorageIndex, _ := strconv.ParseBool(v)
		conf.UseIndex = !disableStorageIndex
	}

	disableMastodonCompatibility, _ := strconv.ParseBool(Getval(KeyMastodonCompatibilityDisable, "false"))
	conf.MastodonCompatible = !disableMastodonCompatibility

	conf.KeyPath = normalizeConfigPath(Getval(KeyKeyPath, ""), conf)
	conf.CertPath = normalizeConfigPath(Getval(KeyCertPath, ""), conf)

	return conf
}
