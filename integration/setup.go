package integration

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/pkg/stdcopy"
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

	"github.com/joho/godotenv"
	containers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type fedboxContainer struct {
	containers.Container
}

type cntrs map[string]*fedboxContainer

var defaultFedBOXImage = "localhost/fedbox/app:dev"

func initMocks(ctx context.Context, suites ...string) (cntrs, error) {
	m := make(cntrs)

	for _, name := range suites {
		storage := filepath.Join(".", "mocks")
		env := filepath.Join(storage, ".env")

		c, err := Run(ctx, defaultFedBOXImage, WithEnvFile(env), WithStorage(storage))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize container %s: %w", name, err)
		}
		//i, err := c.Inspect(ctx)
		//if err != nil {
		//	return nil, fmt.Errorf("unable to inspect container %s: %w", name, err)
		//}
		m[name] = c
	}

	return m, nil
}

func (m cntrs) cleanup(t *testing.T) {
	for _, mm := range m {
		containers.CleanupContainer(t, mm.Container)
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
	//in, _ := fc.Inspect(ctx)

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
func Run(ctx context.Context, image string, opts ...containers.ContainerCustomizer) (*fedboxContainer, error) {
	req := containers.ContainerRequest{
		Image: image,
		LogConsumerCfg: &containers.LogConsumerConfig{
			Opts:      []containers.LogProductionOption{containers.WithLogProductionTimeout(10 * time.Second)},
			Consumers: []containers.LogConsumer{new(containers.StdoutLogConsumer)},
		},
		WaitingFor: wait.ForLog("Starting").WithStartupTimeout(500 * time.Millisecond),
	}

	rreq := containers.GenericContainerRequest{
		ContainerRequest: req,
		ProviderType:     containers.ProviderPodman,
		Started:          true,
	}

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
	eerrs := make([]error, 0)
	for _, cmd := range initializers {
		st, out, err := f.Exec(ctx, cmd)
		if err != nil {
			eerrs = append(eerrs, err)
		}
		if st != 0 {
			// command didn't return success.
		}

		if _, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out); err != nil {
			eerrs = append(eerrs, err)
		}
	}
	return &f, errors.Join(eerrs...)
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
