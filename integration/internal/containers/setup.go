package containers

import (
	"bytes"
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"git.sr.ht/~mariusor/storage-all"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/moby/moby/api/types/container"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/exec"
	nw "github.com/testcontainers/testcontainers-go/network"
)

type ContainerInitializer interface {
	Name() string
	Start(ctx context.Context, t testing.TB, extra ...tc.ContainerCustomizer) ([]tc.Container, error)
	Hostname() string
}

type Running struct {
	Containers []tc.Container
	Network    *tc.DockerNetwork
}

func Suite(container ...ContainerInitializer) []ContainerInitializer {
	return append([]ContainerInitializer{}, container...)
}

func Init(ctx context.Context, t testing.TB, s ...ContainerInitializer) (Running, error) {
	netInitFns := []nw.NetworkCustomizer{nw.WithDriver("bridge")}
	newNetwork, err := nw.New(ctx, netInitFns...)
	if err != nil {
		t.Fatalf("unable to initialize network: %v", err)
	}
	m := Running{Containers: make([]tc.Container, 0), Network: newNetwork}
	for _, img := range s {
		c, err := img.Start(context.WithoutCancel(ctx), t, nw.WithNetworkName([]string{img.Hostname()}, newNetwork.Name))
		if err != nil {
			t.Fatalf("unable to initialize container %T: %v", img, err)
		}

		m.Containers = append(m.Containers, c...)
	}

	return m, nil
}

func (m Running) Cleanup(t testing.TB) {
	if m.Network != nil {
		tc.CleanupNetwork(t, m.Network)
	}
	for _, mm := range m.Containers {
		tc.CleanupContainer(t, mm)
	}
}

func (m Running) RunCommand(ctx context.Context, host string, cmd tc.Executable, IO io.ReadWriter) (io.Reader, error) {
	uu, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("received invalid url: %w", err)
	}

	for _, fc := range m.Containers {
		info, err := fc.Inspect(ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to inspect container: %w", err)
		}
		for _, pair := range info.Config.Env {
			if hostname, found := strings.CutPrefix(pair, "HOSTNAME="); found {
				if hostname == uu.Host {
					opts := cmd.Options()
					if IO != nil {
						opts = append(opts, WithIO(IO))
					}
					_, r, err := fc.Exec(ctx, cmd.AsCommand(), opts...)
					return r, err
				}
			}
		}
	}
	return nil, fmt.Errorf("no matching mock instance for the host: %s", host)
}

func (m Running) RedirectRequest(ctx context.Context, req *http.Request) error {
	for _, fc := range m.Containers {
		info, err := fc.Inspect(ctx)
		if err != nil {
			return fmt.Errorf("unable to inspect container: %w", err)
		}
		for _, pair := range info.Config.Env {
			if host, found := strings.CutPrefix(pair, "HOSTNAME="); found && host == req.URL.Host {
				return rewriteRequestHost(ctx, fc, req)
			}
		}
	}
	return fmt.Errorf("no matching mock instance for the url: %s", req.URL)
}

func rewriteRequestHost(ctx context.Context, fc tc.Container, r *http.Request) error {
	host, err := fc.Endpoint(ctx, "http")
	if err != nil {
		return fmt.Errorf("unable to compose container end-point: %w", err)
	}
	uh, err := url.Parse(host)
	if err != nil {
		return fmt.Errorf("invalid container url: %w", err)
	}

	origHost := r.URL.Host

	r.URL.Host = uh.Host
	r.Host = origHost

	return nil
}

func WithIO(in io.ReadWriter) exec.ProcessOption {
	return exec.ProcessOptionFunc(func(opts *exec.ProcessOptions) {
		opts.Reader = in
	})
}

func WithImage(image string) tc.CustomizeRequestOption {
	return func(req *tc.GenericContainerRequest) error {
		if req.Image == "" {
			req.Image = image
		}
		return nil
	}
}

func WithEnvFromConfig(options config.Options) tc.CustomizeRequestOption {
	return func(req *tc.GenericContainerRequest) error {
		if req.Env == nil {
			req.Env = make(map[string]string)
		}

		exposePort := func(port int) {
			req.ContainerRequest.ExposedPorts = append(req.ContainerRequest.ExposedPorts, strconv.Itoa(port))
		}

		// Disable cache support for the requests handlers and for the storage backends that support it
		req.Env["DISABLE_CACHE"] = strconv.FormatBool(!options.StorageCache)
		// Disable cache support strictly for requests handlers
		req.Env["DISABLE_STORAGE_CACHE"] = strconv.FormatBool(!options.StorageCache)
		req.Env["ENV"] = string(options.Env)
		req.Env["STORAGE"] = string(options.Storage)
		req.Env["STORAGE_PATH"] = options.StoragePath
		if options.HTTPPort > 0 {
			req.Env["HTTP_PORT"] = strconv.Itoa(options.HTTPPort)
			exposePort(options.HTTPPort)
		}
		if options.SSHPort > 0 {
			req.Env["SSH_PORT"] = strconv.Itoa(options.SSHPort)
			exposePort(options.SSHPort)
		}
		if listenHost := options.ListenHost; listenHost != "" {
			req.Env["LISTEN_HOST"] = options.ListenHost
			req.NetworkAliases = map[string][]string{listenHost: {listenHost}}
		}
		if host := options.Hostname; host != "" {
			req.Env["HOSTNAME"] = options.Hostname
			req.NetworkAliases = map[string][]string{host: {host}}
		}
		req.Env["LOG_LEVEL"] = options.LogLevel.String()

		req.ConfigModifier = func(c *container.Config) {
			c.Hostname = options.Hostname
		}
		return nil
	}
}

func WithEnv(env map[string]string) tc.CustomizeRequestOption {
	storageType := ExtractStorageTagFromBuild()
	envType := ExtractEnvTagFromBuild()
	if storageType == "all" {
		storageType = storage.Default
	}
	return func(req *tc.GenericContainerRequest) error {
		if req.Env == nil {
			req.Env = make(map[string]string)
		}
		exposePort := func(portVal string) {
			req.ContainerRequest.ExposedPorts = append(req.ContainerRequest.ExposedPorts, portVal)
		}

		// Disable cache support for the requests handlers and for the storage backends that support it
		req.Env["FEDBOX_DISABLE_CACHE"] = "true"
		// Disable cache support strictly for requests handlers
		req.Env["FEDBOX_DISABLE_STORAGE_CACHE"] = "true"
		req.Env["ENV"] = string(envType)
		req.Env["STORAGE"] = string(storageType)
		req.Env["STORAGE_PATH"] = "/storage"
		if storageType == config.StoragePostgres {
			req.Env["STORAGE_PATH"] = "TBD"
		}
		if httpPort, ok := req.Env["HTTP_PORT"]; ok {
			exposePort(httpPort)
		}
		if sshPort, ok := req.Env["SSH_PORT"]; ok {
			exposePort(sshPort)
		}
		if listenHost, ok := req.Env["LISTEN_HOST"]; ok {
			req.NetworkAliases = map[string][]string{listenHost: {listenHost}}
		}
		if host, ok := req.Env["HOSTNAME"]; ok {
			req.NetworkAliases = map[string][]string{host: {host}}
			req.ConfigModifier = func(c *container.Config) {
				c.Hostname = host
			}
		}
		for k, v := range env {
			if v != "" {
				req.Env[k] = v
			}
		}
		return nil
	}
}

func WithPrivateKey(prv crypto.PrivateKey) tc.CustomizeRequestOption {
	return func(req *tc.GenericContainerRequest) error {
		hostname, ok := req.Env["HOSTNAME"]
		if !ok {
			return nil
		}
		prvEnc, err := x509.MarshalPKCS8PrivateKey(prv)
		if err != nil {
			return err
		}
		keyReader := bytes.NewReader(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: prvEnc}))
		keyFile := tc.ContainerFile{
			ContainerFilePath: filepath.Join("/storage", hostname+".key"),
			Reader:            keyReader,
			FileMode:          0600,
		}
		req.Files = append(req.Files, keyFile)
		return nil
	}
}

func WithPassword(pw []byte) tc.CustomizeRequestOption {
	return func(req *tc.GenericContainerRequest) error {
		hostname, ok := req.Env["HOSTNAME"]
		if !ok {
			return nil
		}
		keyFile := tc.ContainerFile{
			ContainerFilePath: filepath.Join("/storage", hostname+".pw"),
			Reader:            bytes.NewReader(pw),
			FileMode:          0600,
		}
		req.Files = append(req.Files, keyFile)
		return nil
	}
}

func WithInitScript() tc.CustomizeRequestOption {
	initScript := `CREATE USER storage;
GRANT ALL PRIVILEGES ON DATABASE storage TO storage;
`
	return func(req *tc.GenericContainerRequest) error {
		filePath := filepath.Join("/docker-entrypoint-initdb.d", "init-db.sql")
		cf := tc.ContainerFile{
			Reader:            bytes.NewBufferString(initScript),
			ContainerFilePath: filePath,
			FileMode:          0644,
		}
		req.Files = append(req.Files, cf)
		return nil
	}
}

func WithMocks(items ...vocab.Item) tc.CustomizeRequestOption {
	return func(req *tc.GenericContainerRequest) error {
		raw, err := vocab.MarshalJSON(vocab.ItemCollection(items))
		if err != nil {
			return err
		}
		cf := tc.ContainerFile{
			Reader:            bytes.NewBuffer(raw),
			ContainerFilePath: filepath.Join("/storage", "import.json"),
			FileMode:          0600,
		}
		req.Files = append(req.Files, cf)
		return nil
	}
}

func WithStorage(storage string) tc.CustomizeRequestOption {
	base := filepath.Dir(storage)
	return func(req *tc.GenericContainerRequest) error {
		_ = filepath.WalkDir(storage, func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				return nil
			}
			cf := tc.ContainerFile{
				HostFilePath:      path,
				ContainerFilePath: filepath.Join("/storage", strings.TrimPrefix(path, base)),
				FileMode:          0600,
			}
			req.Files = append(req.Files, cf)
			return nil
		})
		return nil
	}
}
