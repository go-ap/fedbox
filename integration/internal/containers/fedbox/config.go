package fedbox

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/storage-all"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	tc "github.com/testcontainers/testcontainers-go"
)

type PrivateKey interface {
	Public() crypto.PublicKey
}

var (
	DefaultC2SOptions = config.Options{
		Hostname: "primary.localdomain",
		HTTPPort: 80,
		SSHPort:  422,
		Env:      env.TEST,
		LogLevel: lw.TraceLevel,
	}

	DefaultS2SOptions = config.Options{
		Hostname: "secondary.localdomain",
		HTTPPort: 80,
		SSHPort:  422,
		Storage:  storage.FS,
		Env:      env.TEST,
		LogLevel: lw.TraceLevel,
	}
)

func C2SConfig(init ...configInitFn) *Config {
	init = append(init, WithConfig(DefaultC2SOptions))
	return InitConfig(init...)
}

func S2SConfig(init ...configInitFn) *Config {
	init = append(init, WithConfig(DefaultS2SOptions))
	return InitConfig(init...)
}

type configInitFn func(*Config)

func WithItems(items ...vocab.Item) configInitFn {
	return func(c *Config) {
		c.Items = items
	}
}

func WithCommands(cmds ...tc.Executable) configInitFn {
	return func(c *Config) {
		c.Fns = cmds
	}
}

func WithPrivateKey(prv PrivateKey) configInitFn {
	return func(c *Config) {
		c.Key = prv
	}
}

func Verbose(b bool) configInitFn {
	return func(c *Config) {
		c.Verbose = b
	}
}

func WithCodeCoverage(b bool) configInitFn {
	return func(c *Config) {
		c.Coverage = b
	}
}

func WithPw(pw string) configInitFn {
	return func(c *Config) {
		c.Pw = pw
	}
}

func WithImageName(image string) configInitFn {
	return func(c *Config) {
		c.Image = image
	}
}

func WithConfig(conf config.Options) configInitFn {
	return func(c *Config) {
		c.Conf = conf
	}
}

func InitConfig(init ...configInitFn) *Config {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	conf := &Config{
		Items: make(vocab.ItemCollection, 0),
		Key:   privateKey,
		Pw:    rand.Text()[:8],
		Fns:   make([]tc.Executable, 0, len(init)),
	}

	for _, in := range init {
		in(conf)
	}
	return conf
}

type Config struct {
	Image    string
	Conf     config.Options
	Items    vocab.ItemCollection
	Key      PrivateKey
	Pw       string
	Fns      []tc.Executable
	Verbose  bool
	Coverage bool
}

func StartContainers(ctx context.Context, t *testing.T, configs ...*Config) (c.Running, error) {
	startFns := make([]c.ContainerInitializer, 0, len(configs))
	errs := make([]error, 0, len(configs))
	for _, conf := range configs {
		if conf.Key == nil {
			conf.Key, _ = rsa.GenerateKey(rand.Reader, 2048)
		}
		if conf.Pw == "" {
			conf.Pw = rand.Text()[:8]
		}
		initFns := []imageInitFn{
			withImageName(conf.Image),
			withConfig(ConfigFromBuildInfo(conf.Conf)),
			withKey(conf.Key),
			withPw(conf.Pw),
			withTestLogger(t, conf.Verbose),
			withItems(conf.Items...),
			withCmd(conf.Fns...),
		}
		if conf.Coverage {
			coveragePath := filepath.Join(t.ArtifactDir(), url.PathEscape(conf.Conf.Hostname), "coverage")
			if coverDir := os.Getenv("GOCOVERDIR"); coverDir != "" {
				coveragePath = coverDir
			}
			initFns = append(initFns, withCodeCoveragePath(coveragePath))
		}
		startFns = append(startFns, New(initFns...))
	}
	if len(errs) > 0 {
		return c.Running{}, errors.Join(errs...)
	}

	images := c.Suite(startFns...)
	running, err := c.Start(ctx, t, images...)
	t.Cleanup(func() {
		running.Cleanup(t)
	})
	return running, err
}
