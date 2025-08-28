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

var (
	Prefix         = "fedbox"
	BaseRuntimeDir = "/var/run"
)

type BackendConfig struct {
	Enabled bool
	Host    string
	Port    int64
	User    string
	Pw      string
	Name    string
}

type Options struct {
	AppName            string
	Version            string
	Env                env.Type
	LogLevel           lw.Level
	LogOutput          string
	TimeOut            time.Duration
	Secure             bool
	CertPath           string
	KeyPath            string
	Hostname           string
	Listen             string
	BaseURL            string
	Storage            StorageType
	StoragePath        string
	StorageCache       bool
	RequestCache       bool
	UseIndex           bool
	Profile            bool
	MastodonCompatible bool
	MaintenanceMode    bool
	ShuttingDown       bool
}

type StorageType string

const (
	XdgRuntimeDir = "XDG_RUNTIME_DIR"

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
	p = strings.ReplaceAll(p, varHost, url.PathEscape(o.Hostname))
	return filepath.Clean(p)
}

func (o Options) BaseStoragePath() (string, error) {
	basePath := normalizeConfigPath(o.StoragePath, o)
	fi, err := os.Stat(basePath)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(basePath, defaultDirPerm)
	}
	if err != nil {
		return "", err
	}
	fi, err = os.Stat(basePath)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		panic(errors.NotValidf("path %s is invalid for storage", basePath))
	}
	return basePath, nil
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

func Load(path string, e env.Type, timeOut time.Duration) (Options, error) {
	if !env.ValidType(e) {
		e = env.Type(Getval(KeyENV, ""))
	}
	configs := []string{
		".env",
	}
	appendIfFile := func(path string, typ env.Type) {
		envFile := fmt.Sprintf(".env.%s", typ)
		if _, err := os.Stat(filepath.Join(path, envFile)); err == nil {
			configs = append(configs, envFile)
		}
	}
	if !env.ValidType(e) {
		for _, typ := range env.Types {
			appendIfFile(path, typ)
		}
	} else {
		appendIfFile(path, e)
	}
	err := godotenv.Load(configs...)
	if err != nil {
		return Options{}, err
	}

	opts := LoadFromEnv()
	opts.AppName = strings.Trim(Prefix, "_")
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
	if conf.Hostname == "" {
		conf.Hostname = Getval(KeyHostname, conf.Hostname)
	}
	conf.TimeOut = 0
	if to, _ := time.ParseDuration(Getval(KeyTimeOut, "")); to > 0 {
		conf.TimeOut = to
	}
	conf.Secure, _ = strconv.ParseBool(Getval(KeyHTTPS, "false"))
	if conf.Secure {
		conf.BaseURL = fmt.Sprintf("https://%s", conf.Hostname)
	} else {
		conf.BaseURL = fmt.Sprintf("http://%s", conf.Hostname)
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

func (o Options) RuntimePath() string {
	path := BaseRuntimeDir
	if runtimeDir := os.Getenv(XdgRuntimeDir); runtimeDir != "" {
		path = runtimeDir
	}
	return path
}

func (o Options) DefaultSocketPath() string {
	name := strings.ToLower(o.pathInstanceName())
	return filepath.Join(o.RuntimePath(), name+".sock")
}

func (o Options) pathInstanceName() string {
	name := o.AppName
	if o.Hostname != "" {
		host := strings.Replace(o.Hostname, "https://", "", 1)
		host = strings.Replace(host, "http://", "", 1)
		host = strings.Replace(host, ".", "-", -1)
		name += "-" + host
	}
	return name
}

func (o Options) PidPath() string {
	name := strings.ToLower(o.pathInstanceName())
	return filepath.Join(o.RuntimePath(), name+".pid")
}

func (o Options) WritePid() error {
	pid := os.Getpid()
	raw := make([]byte, 0)
	raw = strconv.AppendUint(raw, uint64(pid), 10)

	pidPath := o.PidPath()
	if err := os.MkdirAll(filepath.Dir(pidPath), 0o700); err != nil {
		return err
	}

	return os.WriteFile(pidPath, raw, 0o600)
}

func (o Options) ReadPid() (int, error) {
	raw, err := os.ReadFile(o.PidPath())
	if err != nil {
		return -1, err
	}

	pid, err := strconv.ParseUint(string(raw), 10, 32)
	if err != nil {
		return -1, err
	}
	return int(pid), nil
}
