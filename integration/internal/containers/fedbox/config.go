package fedbox

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/storage-all"
	vocab "github.com/go-ap/activitypub"
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
		Env:      env.DEV,
		LogLevel: lw.TraceLevel,
	}
	DefaultS2SOptions = config.Options{
		Hostname: "secondary.localdomain",
		HTTPPort: 80,
		SSHPort:  422,
		Storage:  storage.FS,
		Env:      env.PROD,
		LogLevel: lw.DebugLevel,
	}
)

func C2SConfig(init ...any) *FedBOXConfig {
	init = append(init, any(DefaultC2SOptions))
	return InitFedBOXConfig(init...)
}

func InitFedBOXConfig(init ...any) *FedBOXConfig {
	items := make(vocab.ItemCollection, 0, len(init))
	fns := make([]tc.Executable, 0, len(init))
	var privateKey PrivateKey
	var pw string
	var opts config.Options
	var image string
	for _, in := range init {
		switch opt := in.(type) {
		case vocab.Item:
			items.Append(opt)
		case tc.Executable:
			fns = append(fns, opt)
		case PrivateKey:
			privateKey = opt
		case []byte:
			pw = string(opt)
		case string:
			image = opt
		case config.Options:
			opts = opt
		}
	}
	if privateKey == nil {
		privateKey, _ = rsa.GenerateKey(rand.Reader, 2048)
	}
	if pw == "" {
		pw = rand.Text()[:8]
	}
	return &FedBOXConfig{
		Image: image,
		Conf:  opts,
		Items: items,
		Key:   privateKey,
		Pw:    pw,
		Fns:   fns,
	}
}

type FedBOXConfig struct {
	Image   string
	Conf    config.Options
	Items   vocab.ItemCollection
	Key     PrivateKey
	Pw      string
	Fns     []tc.Executable
	Verbose bool
}

func StartContainers(ctx context.Context, t *testing.T, configs ...*FedBOXConfig) (c.Running, error) {
	initFns := make([]c.ContainerInitializer, 0, len(configs))
	for _, conf := range configs {
		if conf.Key == nil {
			conf.Key, _ = rsa.GenerateKey(rand.Reader, 2048)
		}
		if conf.Pw == "" {
			conf.Pw = rand.Text()[:8]
		}

		initFns = append(initFns, New(
			WithImageName(conf.Image),
			WithArgs([]string{"--bootstrap"}),
			WithConfig(ConfigFromBuildInfo(conf.Conf)),
			WithKey(conf.Key),
			WithPw(conf.Pw),
			WithTestLogger(t, conf.Verbose),
			WithItems(conf.Items...),
			WithCmd(conf.Fns...),
		),
		)
	}
	images := c.Suite(initFns...)
	running, err := c.Start(ctx, t, images...)
	if err != nil {
		return running, err
	}
	return running, nil
}
