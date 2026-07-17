package containers

import (
	"context"
	"crypto"
	"fmt"
	"testing"

	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/storage-all"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	tc "github.com/testcontainers/testcontainers-go"
)

type ContainerInitializer interface {
	Start(ctx context.Context, t testing.TB) (tc.Container, error)
}

type fboxImage struct {
	name    string
	args    []string
	conf    *config.Options
	env     map[string]string
	rootIRI string
	key     crypto.PrivateKey
	pw      []byte
	mocks   vocab.ItemCollection
	cmds    []tc.Executable
	logger  *testLogger
}

func (f *fboxImage) EnvType() string {
	return string(f.conf.Env)
}

func (f *fboxImage) StorageType() string {
	return string(f.conf.Storage)
}

func (f *fboxImage) Hostname() string {
	return f.conf.Hostname
}

func (f *fboxImage) InitFns() []tc.ContainerCustomizer {
	initFns := []tc.ContainerCustomizer{WithImage(f.name), WithEnvFromConfig(*f.conf)}
	if len(f.env) > 0 {
		initFns = append(initFns, WithEnvFile(f.env))
	}

	if f.args != nil {
		initFns = append(initFns, tc.WithCmdArgs(f.args...))
	}
	if f.key != nil {
		initFns = append(initFns, WithPrivateKey(f.key))
	}
	if f.pw != nil {
		initFns = append(initFns, WithPassword(f.pw))
	}
	if len(f.mocks) > 0 {
		importCmd := SSHCmd{
			Cmd:  []string{ /*ctlBin, "--env", envType, */ "pub", "import", "/storage/import.json"},
			User: f.rootIRI,
		}
		if f.key != nil {
			importCmd.Key = f.key
		}
		if f.pw != nil {
			importCmd.Pw = f.pw
		}
		// NOTE(marius): we add the mocks to the import file, and the SSH command to actually import it.
		initFns = append(initFns, WithMocks(f.mocks...), tc.WithAfterReadyCommand(importCmd))
	}
	if f.logger != nil {
		initFns = append(initFns, tc.WithLogConsumers(f.logger))
	}
	return initFns
}

func (f *fboxImage) Start(ctx context.Context, t testing.TB) (tc.Container, error) {
	opts := f.InitFns()

	req := defaultFedBOXRequest(f, t)
	hostCfg := func(hostConfig *container.HostConfig) {
		hostConfig.NetworkMode = network.NetworkBridge
		hostConfig.AutoRemove = true
		hostConfig.ExtraHosts = []string{"localhost:" + f.Hostname()}
	}

	opts = append(opts, tc.WithHostConfigModifier(hostCfg))
	for _, opt := range opts {
		if err := opt.Customize(&req); err != nil {
			return nil, err
		}
	}

	// NOTE(marius): for fedbox containers we use our custom Exec, that is capable of
	// running the Post Ready Hook commands as SSH
	var cmds []tc.ContainerHook
	for i, hook := range req.LifecycleHooks {
		if len(hook.PostReadies) > 0 {
			cmds = hook.PostReadies
			req.LifecycleHooks[i].PostReadies = nil
		}
	}

	fc, err := tc.GenericContainer(ctx, req)
	if err != nil {
		return nil, err
	}

	c := fboxContainer{Container: fc}
	if err = c.Start(ctx); err != nil {
		return &c, err
	}
	if len(cmds) > 0 {
		errs := make([]error, 0, len(cmds))
		name, _ := c.Name(ctx)
		for _, ex := range cmds {
			if err = ex(ctx, c); err != nil {
				errs = append(errs, fmt.Errorf("unable to run startup command on container %s[%T]: %w", name, f, err))
			}
		}
		err = errors.Join(errs...)
	}

	return &c, err
}

type imageInitFn func(*fboxImage)

func WithTestLogger(t testing.TB, enabled bool) imageInitFn {
	return func(f *fboxImage) {
		if !enabled {
			return
		}
		f.conf.LogLevel = lw.TraceLevel
		logger := testLogger{T: t.(*testing.T)}
		f.logger = &logger
	}
}

func WithItems(it ...vocab.Item) imageInitFn {
	return func(f *fboxImage) {
		f.mocks = it
	}
}

func WithImageName(name string) imageInitFn {
	return func(f *fboxImage) {
		f.name = name
	}
}

func WithRootIRI(user vocab.IRI) imageInitFn {
	return func(f *fboxImage) {
		f.rootIRI = string(user)
	}
}

func WithKey(key crypto.PrivateKey) imageInitFn {
	return func(f *fboxImage) {
		f.key = key
	}
}

func WithPw(pw string) imageInitFn {
	return func(f *fboxImage) {
		f.pw = []byte(pw)
	}
}

func WithEnv(m map[string]string) imageInitFn {
	return func(f *fboxImage) {
		f.env = m
	}
}

func WithConfig(opts config.Options) imageInitFn {
	return func(f *fboxImage) {
		f.conf = &opts
	}
}

func WithArgs(args []string) imageInitFn {
	return func(f *fboxImage) {
		f.args = args
	}
}

func WithCmds(cmds ...tc.Executable) imageInitFn {
	return func(f *fboxImage) {
		f.cmds = cmds
	}
}

func FedBOXNew(fns ...imageInitFn) *fboxImage {
	img := new(fboxImage)
	for _, fn := range fns {
		fn(img)
	}
	return img
}

func ConfigFromBuildInfo(base config.Options) config.Options {
	if base.Env == "" {
		base.Env = ExtractEnvTagFromBuild()
	}
	if base.Storage == "" {
		storageTyp := ExtractStorageTagFromBuild()
		if storageTyp == "all" {
			storageTyp = storage.Default
		}
		base.Storage = storageTyp
	}
	if base.StoragePath == "" {
		base.StoragePath = "/storage"
	}
	return base
}
