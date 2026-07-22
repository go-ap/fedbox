package fedbox

import (
	"context"
	"crypto"
	"fmt"
	"strconv"
	"testing"
	"time"

	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/storage-all"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/internal/config"
	pg "github.com/go-ap/storage-pg"
	"github.com/jackc/pgx/v5"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type tbLogger struct {
	testing.TB
}

func (t *tbLogger) Printf(s string, args ...any) {
	t.Logf(s, args...)
}

func (t *tbLogger) Accept(l tc.Log) {
	t.Helper()
	t.Logf("%s", l.Content)
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
	logger  *tbLogger
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

func (f *fboxImage) Name() string {
	return f.name
}

func (f *fboxImage) InitFns() []tc.ContainerCustomizer {
	initFns := []tc.ContainerCustomizer{c.WithImage(f.name), c.WithEnvFromConfig(*f.conf)}
	if len(f.env) > 0 {
		initFns = append(initFns, c.WithEnvFile(f.env))
	}

	if f.args != nil {
		initFns = append(initFns, tc.WithCmdArgs(f.args...))
	}
	if f.key != nil {
		initFns = append(initFns, c.WithPrivateKey(f.key))
	}
	if f.pw != nil {
		initFns = append(initFns, c.WithPassword(f.pw))
	}
	if len(f.mocks) > 0 {
		importCmd := c.SSHCmd{
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
		initFns = append(initFns, c.WithMocks(f.mocks...), tc.WithAfterReadyCommand(importCmd))
	}
	if f.logger != nil {
		initFns = append(initFns, tc.WithLogConsumers(f.logger))
	}
	return initFns
}

func initPGSidecar(ctx context.Context, f *fboxImage) (tc.Container, error) {
	pgInitFns := []tc.ContainerCustomizer{
		c.WithInitScript(),
		postgres.WithDatabase("storage"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		tc.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5 * time.Second)),
	}
	if f.logger != nil {
		pgInitFns = append(pgInitFns, tc.WithLogger(f.logger))
	}

	pgContainer, err := postgres.Run(ctx, "postgres:18-alpine", pgInitFns...)
	if err != nil {
		return nil, err
	}
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, err
	}
	pconf, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	pgConf := pg.Config{
		Host:     pconf.Host,
		Port:     pconf.Port,
		Database: pconf.Database,
		User:     pconf.User,
		Password: pconf.Password,
	}
	f.conf.StoragePath = pgConf.DSN()
	return pgContainer, nil
}

func (f *fboxImage) Start(ctx context.Context, t testing.TB) ([]tc.Container, error) {
	cont := make([]tc.Container, 0, 2)
	if f.conf.Storage == config.StoragePostgres {
		// NOTE(marius): currently the FedBOX service doesn't really work with postgres
		pg, err := initPGSidecar(ctx, f)
		if err != nil {
			return nil, fmt.Errorf("unable to initialize PostgreSQL container: %w", err)
		}
		cont = append(cont, pg)
	}

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

	c := fboxContainer{Container: fc, img: *f}
	if err = c.Start(ctx); err != nil {
		return nil, fmt.Errorf("unable to start FedBOX container: %w", err)
	}
	cont = append(cont, &c)
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

	return cont, err
}

type imageInitFn func(*fboxImage)

func WithTestLogger(t testing.TB, enabled bool) imageInitFn {
	return func(f *fboxImage) {
		if !enabled {
			return
		}
		f.conf.LogLevel = lw.TraceLevel
		logger := tbLogger{TB: t}
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

func WithCmd(cmds ...tc.Executable) imageInitFn {
	return func(f *fboxImage) {
		f.cmds = cmds
	}
}

func New(fns ...imageInitFn) *fboxImage {
	img := new(fboxImage)
	for _, fn := range fns {
		fn(img)
	}
	return img
}

func ConfigFromBuildInfo(base config.Options) config.Options {
	if base.Env == "" {
		base.Env = c.ExtractEnvTagFromBuild()
	}
	if base.Storage == "" {
		storageTyp := c.ExtractStorageTagFromBuild()
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

func defaultFedBOXRequest(fb *fboxImage, _ testing.TB) tc.GenericContainerRequest {
	return tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image: fb.name,
			WaitingFor: wait.ForAny(
				wait.ForListeningPort(strconv.Itoa(fb.conf.HTTPPort)),
				wait.ForListeningPort(strconv.Itoa(fb.conf.SSHPort)),
			).WithDeadline(5 * time.Second),
		},
		ProviderType: tc.ProviderPodman,
		Started:      true,
	}
}
