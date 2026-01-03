package config

import (
	"fmt"
	"math/rand/v2"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/storage-all"
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
	AppName              string
	Version              string
	Env                  env.Type
	LogLevel             lw.Level
	LogOutput            string
	TimeOut              time.Duration
	Secure               bool
	CertPath             string
	KeyPath              string
	Hostname             string
	SSHPort              int
	Listen               string
	BaseURL              string
	Storage              storage.Type
	StoragePath          string
	StorageCache         bool
	RequestCache         bool
	UseIndex             bool
	Profile              bool
	MastodonIncompatible bool
	ShuttingDown         bool
}

func (o Options) StorageInitFns(l lw.Logger) []storage.InitFn {
	path, err := o.BaseStoragePath()
	if err != nil {
		return nil
	}
	initFns := []storage.InitFn{
		storage.WithPath(path),
		storage.UseIndex(o.UseIndex),
		storage.WithType(o.Storage),
		storage.WithCache(o.StorageCache),
		storage.WithEnv(string(o.Env)),
		storage.WithHostname(o.Hostname),
	}
	if l != nil {
		initFns = append(initFns, storage.WithLogger(l))
	}
	return initFns
}

type StorageType string

const (
	XdgRuntimeDir = "XDG_RUNTIME_DIR"

	KeyENV                          = "ENV"
	KeyTimeOut                      = "TIME_OUT"
	KeyLogLevel                     = "LOG_LEVEL"
	KeyLogOutput                    = "LOG_OUTPUT"
	KeyHostname                     = "HOSTNAME"
	KeySSHPort                      = "SSH_PORT"
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

	StorageBoltDB   = storage.Type("boltdb")
	StorageFS       = storage.Type("fs")
	StorageBadger   = storage.Type("badger")
	StoragePostgres = storage.Type("postgres")
	StorageSqlite   = storage.Type("sqlite")
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
	o.StoragePath = normalizeConfigPath(o.StoragePath, o)
	fi, err := os.Stat(o.StoragePath)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(o.StoragePath, defaultDirPerm)
	}
	if err != nil {
		return "", err
	}
	fi, err = os.Stat(o.StoragePath)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", errors.NotValidf("path %s is invalid for storage", o.StoragePath)
	}
	return o.StoragePath, nil
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

func findConfigs(path string, e env.Type) []string {
	configs := make([]string, 0)
	appendIfFile := func(root *os.Root, typ env.Type) {
		envFiles := []string{".env", fmt.Sprintf(".env.%s", typ)}
		for _, envFile := range envFiles {
			if _, err := root.Stat(envFile); err == nil {
				configs = append(configs, filepath.Join(root.Name(), envFile))
			}
		}
	}
	loadPath := func(path string, e env.Type) error {
		root, err := os.OpenRoot(normalizeConfigPath(path, Options{Env: e}))
		if err != nil {
			return err
		}
		if !env.ValidType(e) {
			for _, typ := range env.Types {
				appendIfFile(root, typ)
			}
		} else {
			appendIfFile(root, e)
		}
		return nil
	}
	if err := loadPath(path, e); err != nil {
		_ = loadPath(".", e)
	}
	return configs
}

func Load(path string, e env.Type, timeOut time.Duration) (Options, error) {
	if !env.ValidType(e) {
		e = env.Type(Getval(KeyENV, ""))
	}

	configs := findConfigs(path, e)
	if len(configs) > 0 {
		if err := godotenv.Overload(configs...); err != nil {
			return Options{}, err
		}
	}

	opts := LoadFromEnv()
	opts.Env = e
	opts.TimeOut = timeOut

	return opts, validateOptions(opts)
}

func validateOptions(opts Options) error {
	if opts.Env == "" {
		return errors.Errorf("invalid environment")
	}
	if opts.Storage == "" {
		return errors.Errorf("invalid storage")
	}
	if opts.StoragePath == "" {
		return errors.Errorf("invalid storage path")
	}
	if opts.Listen == "" {
		return errors.Errorf("invalid listen socket")
	}
	return nil
}

const minPort = 1024

var RandPort = minPort + rand.IntN(65536-minPort)

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
	conf.Storage = storage.Type(strings.ToLower(Getval(KeyStorage, string(storage.Default))))
	conf.StoragePath = Getval(KeyStoragePath, "")
	if conf.StoragePath != "" {
		conf.StoragePath = filepath.Clean(conf.StoragePath)
	}

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
	if v := Getval(KeySSHPort, strconv.Itoa(RandPort)); v != "" {
		sshPort, _ := strconv.ParseUint(v, 10, 32)
		conf.SSHPort = int(sshPort)
	}

	disableMastodonCompatibility, _ := strconv.ParseBool(Getval(KeyMastodonCompatibilityDisable, "false"))
	conf.MastodonIncompatible = disableMastodonCompatibility

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
	name := o.pathInstanceName()
	return filepath.Join(o.RuntimePath(), name+".sock")
}

func reverseDNSName(host string) string {
	host = strings.Replace(host, "https://", "", 1)
	host = strings.Replace(host, "http://", "", 1)
	pieces := strings.Split(host, ".")
	sort.Slice(pieces, func(i, j int) bool {
		return i > j
	})
	return strings.Join(pieces, ".")
}
func (o Options) pathInstanceName() string {
	name := o.AppName
	if o.Hostname != "" {
		name = reverseDNSName(o.Hostname) + "." + name
	}
	return strings.ToLower(name)
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
