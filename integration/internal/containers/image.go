package containers

import (
	"context"
	"crypto"
	"fmt"
	"testing"

	vocab "github.com/go-ap/activitypub"
	tc "github.com/testcontainers/testcontainers-go"
)

type ContainerInitializer interface {
	//initFns() []tc.ContainerCustomizer
	Start(ctx context.Context, t testing.TB) (tc.Container, error)
}

type fboxImage struct {
	name string
	env  map[string]string
	user string
	key  crypto.PrivateKey
	pw   []byte
	//startCmds []tc.Executable
	mocks vocab.ItemCollection
}

func (f *fboxImage) InitFns() []tc.ContainerCustomizer {
	initFns := []tc.ContainerCustomizer{WithImage(f.name), WithEnvFile(f.env)}
	if len(f.mocks) > 0 {
		// NOTE(marius): the default import for the mocks
		initFns = append(initFns, WithMocks(f.mocks...))
		if f.pw != nil || f.key != nil {
			// NOTE(marius): we also need to add an SSH command to import the mocks file
			importCmd := SSHCmd{
				Cmd:  []string{ /*ctlBin, "--env", envType, */ "pub", "import", "/storage/import.json"},
				User: f.user,
			}
			if len(f.user) > 0 {
			}
			if f.key != nil {
				importCmd.Key = f.key
				initFns = append(initFns, WithPrivateKey(f.key))
			}
			if f.pw != nil {
				importCmd.Pw = f.pw
				initFns = append(initFns, WithPassword(f.pw))
			}
			initFns = append(initFns, tc.WithAfterReadyCommand(importCmd))
		}
	}
	return initFns
}

func (f *fboxImage) Start(ctx context.Context, t testing.TB) (tc.Container, error) {
	opts := f.InitFns()

	logger := testLogger{T: t.(*testing.T)}

	req := defaultFedBOXRequest(f.name)
	opts = append(opts, tc.WithLogConsumers(logger))
	//opts = append(opts, tc.WithLogger(log.TestLogger(t)))
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
		name, _ := c.Name(ctx)
		for _, ex := range cmds {
			if err = ex(ctx, c); err != nil {
				return nil, fmt.Errorf("unable to run startup commands on container %s[%T]: %w", name, f, err)
			}
		}
	}

	return &c, nil
}

type imageInitFn func(*fboxImage)

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

func WithUser(user vocab.IRI) imageInitFn {
	return func(f *fboxImage) {
		f.user = string(user)
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

func C2SfedBOX(fns ...imageInitFn) *fboxImage {
	img := &fboxImage{}
	for _, fn := range fns {
		fn(img)
	}
	return img
}
