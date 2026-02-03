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
	"time"

	"git.sr.ht/~mariusor/storage-all"
	vocab "github.com/go-ap/activitypub"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/exec"
	nw "github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

type Running struct {
	Containers []tc.Container
	Network    *tc.DockerNetwork
}

type Suite []ContainerInitializer

type testLogger struct {
	*testing.T
}

func (t testLogger) Printf(s string, args ...any) {
	t.Helper()
	t.Logf(s, args...)
}

func (t testLogger) Accept(l tc.Log) {
	t.Helper()
	t.Logf("%s", l.Content)
}

type image struct {
	name      string
	env       map[string]string
	startCmds []tc.Executable
}

func (i image) initFns() []tc.ContainerCustomizer {
	return []tc.ContainerCustomizer{WithImage(i.name), WithEnvFile(i.env), tc.WithMounts(tc.ContainerMount{
		Source: tc.GenericTmpfsMountSource{},
		Target: "/storage",
	})}
}

func (i image) Start(ctx context.Context, t testing.TB) (tc.Container, error) {
	return nil, fmt.Errorf("TODO generic containers")
}

func Init(ctx context.Context, t testing.TB, s ...ContainerInitializer) (Running, error) {
	// NOTE(marius): the docker host can come from multiple places.
	// @see github.com/testcontainers/testcontainers-go/internal/core.MustExtractDockerHost()
	newNetwork, err := nw.New(ctx)
	if err != nil {
		t.Fatalf("unable to initialize nw: %s", err)
	}
	m := Running{Containers: make([]tc.Container, 0), Network: newNetwork}
	for _, img := range s {
		c, err := img.Start(ctx, t)
		if err != nil {
			t.Fatalf("unable to initialize container %T: %s", img, err)
		}

		m.Containers = append(m.Containers, c)
	}

	m.Cleanup(t)
	return m, nil
}

func (m Running) Cleanup(t testing.TB) {
	for _, mm := range m.Containers {
		tc.CleanupContainer(t, mm)
	}
	tc.CleanupNetwork(t, m.Network)
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
			if strings.HasPrefix(pair, "HOSTNAME=") {
				if host := strings.TrimPrefix(pair, "HOSTNAME="); host == uu.Host {
					opts := cmd.Options()
					if IO != nil {
						opts = append(opts, WithIO(IO))
					}
					return execSSH(ctx, fc, cmd.AsCommand(), opts...)
				}
			}
		}
	}
	return nil, fmt.Errorf("no matching mock instance for the host: %s", host)
}

func (m Running) BuildRequest(ctx context.Context, met, u string, body io.Reader) (*http.Request, error) {
	uu, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("received invalid url: %w", err)
	}

	for _, fc := range m.Containers {
		info, err := fc.Inspect(ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to inspect container: %w", err)
		}
		for _, pair := range info.Config.Env {
			if strings.HasPrefix(pair, "HOSTNAME=") {
				if host := strings.TrimPrefix(pair, "HOSTNAME="); host == uu.Host {
					return buildRequest(ctx, fc, met, u, body)
				}
			}
		}
	}
	return nil, fmt.Errorf("no matching mock instance for the url: %s", u)
}

func buildRequest(ctx context.Context, fc tc.Container, met, u string, body io.Reader) (*http.Request, error) {
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

func defaultFedBOXRequest(name string) tc.GenericContainerRequest {
	envType := ExtractEnvTagFromBuild()
	return tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image: name,
			Cmd:   []string{"--env", envType, "--bootstrap"},
			WaitingFor: wait.ForAll(
				wait.ForLog("Started"),
				wait.ForListeningPort("4000"),
				wait.ForListeningPort("4044"),
			).WithDeadline(800 * time.Millisecond),
		},
		ProviderType: tc.ProviderPodman,
		Started:      true,
	}
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

func WithEnvFile(env map[string]string) tc.CustomizeRequestOption {
	storageType := extractStorageTagFromBuild()
	envType := ExtractEnvTagFromBuild()
	if storageType == "all" {
		storageType = string(storage.Default)
	}
	return func(req *tc.GenericContainerRequest) error {
		if req.Env == nil {
			req.Env = make(map[string]string)
		}
		for k, v := range env {
			if v != "" {
				req.Env[k] = v
			}
		}
		exposePort := func(portVal string) {
			if port, err := strconv.ParseUint(portVal, 10, 32); err == nil {
				req.ContainerRequest.ExposedPorts = append(req.ContainerRequest.ExposedPorts, strconv.FormatInt(int64(port), 10))
			}
		}

		req.Env["ENV"] = envType
		req.Env["STORAGE"] = storageType

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
