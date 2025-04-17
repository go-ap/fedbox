package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/joho/godotenv"
	containers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/log"
	"github.com/testcontainers/testcontainers-go/wait"
)

type fedboxContainer struct {
	containers.Container
}

type cntrs map[string]*fedboxContainer

var defaultFedBOXImageName = "localhost/fedbox/app:dev"

type suite struct {
	name    string
	storage string
}

type testLogger func(s string, args ...any)

func (t testLogger) Printf(s string, args ...any) {
	t(s, args...)
}

func (t testLogger) Accept(l containers.Log) {
	t(string(l.Content))
}

func initMocks(ctx context.Context, t *testing.T, suites ...suite) (cntrs, error) {
	m := make(cntrs)

	for _, s := range suites {
		storage := filepath.Join(".", "mocks")
		env := filepath.Join(storage, ".env")

		img := defaultFedBOXImageName
		if s.storage != "" {
			img += "-" + s.storage
		}
		c, err := Run(ctx, t, img, WithEnvFile(env), WithStorage(storage))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize container %s: %w", s.name, err)
		}
		_, err = c.Inspect(ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to inspect container %s: %w", c.Container, err)
		}
		m[s.name] = c
	}

	return m, nil
}

func (m cntrs) cleanup(t *testing.T) {
	for _, mm := range m {
		containers.CleanupContainer(t, mm)
	}
}

func (m cntrs) Req(ctx context.Context, met, u string, body io.Reader) (*http.Request, error) {
	uu, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("received invalid url: %w", err)
	}

	fc, ok := m[uu.Host]
	if !ok {
		return nil, fmt.Errorf("no matching mock instance for the url: %s", u)
	}

	return fc.Req(ctx, met, u, body)
}

func (fc *fedboxContainer) Req(ctx context.Context, met, u string, body io.Reader) (*http.Request, error) {
	uu, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("received invalid url: %w", err)
	}

	host, err := fc.Endpoint(ctx, "https")
	if err != nil {
		return nil, fmt.Errorf("unable to compose container end-point: %w", err)
	}
	uh, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("invalid container url: %w", err)
	}

	origHost := uu.Host
	uu.Host = uh.Host

	u = uu.String()

	r, err := http.NewRequestWithContext(ctx, met, u, body)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}
	r.Host = origHost

	return r, nil
}

// Run creates an instance of the FedBOX container type
func Run(ctx context.Context, t testing.TB, image string, opts ...containers.ContainerCustomizer) (*fedboxContainer, error) {

	logger := testLogger(t.Logf)
	req := containers.ContainerRequest{
		Image: image,
		LogConsumerCfg: &containers.LogConsumerConfig{
			Opts:      []containers.LogProductionOption{containers.WithLogProductionTimeout(10 * time.Second)},
			Consumers: []containers.LogConsumer{logger},
		},
		WaitingFor: wait.ForLog("Starting").WithStartupTimeout(500 * time.Millisecond),
	}

	rreq := containers.GenericContainerRequest{
		ContainerRequest: req,
		ProviderType:     containers.ProviderPodman,
		Started:          true,
	}

	opts = append(opts, WithLogger(logger))
	for _, opt := range opts {
		if err := opt.Customize(&rreq); err != nil {
			return nil, err
		}
	}

	fc, err := containers.GenericContainer(ctx, rreq)
	if err != nil {
		return nil, err
	}
	f := fedboxContainer{Container: fc}

	if err = f.Start(ctx); err != nil {
		return &f, err
	}

	initializers := [][]string{
		{"fedboxctl", "--env", "dev", "bootstrap"},
		{"fedboxctl", "--env", "dev", "pub", "import", "/storage/import.json"},
	}
	errs := make([]error, 0)
	for _, cmd := range initializers {
		st, out, err := f.Exec(ctx, cmd)
		if err != nil {
			errs = append(errs, err)
		}
		if st != 0 {
			// command didn't return success.
			errs = append(errs, fmt.Errorf("command failed"))
		}

		if _, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out); err != nil {
			errs = append(errs, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	return &f, errors.Join(errs...)
}

var envKeys = []string{
	"DISABLE_STORAGE_CACHE", "DISABLE_REQUEST_CACHE", "DISABLE_STORAGE_INDEX", "DISABLE_MASTODON_COMPATIBILITY",
	"STORAGE_PATH", "DISABLE_CACHE", "DB_PASSWORD", "LISTEN", "DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "STORAGE",
	"LOG_LEVEL", "TIME_OUT", "LOG_OUTPUT", "HOSTNAME", "HTTPS", "CERT_PATH", "KEY_PATH", "ENV",
}

func loadEnv() map[string]string {
	conf := make(map[string]string)
	for _, k := range envKeys {
		v := os.Getenv(k)
		if v == "" {
			continue
		}
		conf[k] = v
	}
	return conf
}

var defaultPort = 6669

func parseListen(s string) (string, int) {
	pieces := strings.Split(s, ":")
	port := defaultPort
	host := ""
	switch len(pieces) {
	case 1:
		if p, err := strconv.Atoi(pieces[0]); err == nil {
			port = p
		}
	case 2:
		if p, err := strconv.Atoi(pieces[1]); err == nil {
			port = p
		}
		host = pieces[0]
	}
	return host, port
}

func WithLogger(logFn log.Logger) containers.CustomizeRequestOption {
	return func(req *containers.GenericContainerRequest) error {
		req.Logger = logFn
		return nil
	}
}

func WithEnvFile(configFile string) containers.CustomizeRequestOption {
	_ = godotenv.Load(configFile)
	return func(req *containers.GenericContainerRequest) error {
		if req.Env == nil {
			req.Env = make(map[string]string)
		}
		for k, v := range loadEnv() {
			if v != "" {
				req.Env[k] = v
			}
		}
		if listen, ok := req.Env["LISTEN"]; ok {
			host, port := parseListen(listen)
			req.ContainerRequest.ExposedPorts = append(req.ContainerRequest.ExposedPorts, strconv.FormatInt(int64(port), 10))
			req.NetworkAliases = map[string][]string{host: {host}}
		}
		if host, ok := req.Env["HOSTNAME"]; ok {
			req.NetworkAliases = map[string][]string{host: {host}}
		}
		return nil
	}
}

func WithStorage(storage string) containers.CustomizeRequestOption {
	var files []containers.ContainerFile

	_ = filepath.WalkDir(storage, func(path string, d fs.DirEntry, err error) error {
		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}
		cf := containers.ContainerFile{
			HostFilePath:      path,
			ContainerFilePath: filepath.Join("/storage", strings.ReplaceAll(path, "mocks", "")),
			FileMode:          0o755,
		}
		files = append(files, cf)
		return nil
	})

	return func(req *containers.GenericContainerRequest) error {
		req.Files = append(req.Files, files...)
		return nil
	}
}
