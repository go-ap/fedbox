package integration

import (
	"bytes"
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

	"git.sr.ht/~mariusor/storage-all"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/joho/godotenv"
	containers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/log"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/ssh"
)

type fedboxContainer struct {
	containers.Container
}

type cntrs map[string]*fedboxContainer

type suite struct {
	name    string
	storage string
}

type testLogger struct {
	*testing.T //func(s string, args ...any)
}

func (t testLogger) Printf(s string, args ...any) {
	t.Helper()
	t.Logf(s, args...)
}

func (t testLogger) Accept(l containers.Log) {
	t.Helper()
	t.Logf("%s", l.Content)
}

func initMocks(ctx context.Context, t *testing.T, suites ...suite) (cntrs, error) {
	t.Helper()

	// NOTE(marius): the docker host can come from multiple places.
	// @see github.com/testcontainers/testcontainers-go/internal/core.MustExtractDockerHost()
	m := make(cntrs)
	for _, s := range suites {
		storagePath := filepath.Join(".", "mocks")
		env := filepath.Join(storagePath, ".env")

		//storagePath := t.TempDir()
		img := fedboxImageName
		t.Logf("Mock image: %s path %s", fedboxImageName, storagePath)
		c, err := Run(ctx, t, WithImageName(img), WithEnvFile(env), WithStorage(storagePath))
		if err != nil {
			return nil, fmt.Errorf("unable to initialize container %s: %w", s.name, err)
		}
		_, err = c.Inspect(ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to inspect container %s: %w", c.Container, err)
		}
		m[s.name] = c
	}

	m.cleanup(t)
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

func defaultFedBOXRequest(name string) containers.GenericContainerRequest {
	envType := extractEnvTagFromBuild()
	return containers.GenericContainerRequest{
		ContainerRequest: containers.ContainerRequest{
			Image:      name,
			Cmd:        []string{"--env", envType},
			WaitingFor: wait.ForLog("Started").WithStartupTimeout(800 * time.Millisecond),
		},
		ProviderType: containers.ProviderPodman,
		Started:      true,
	}
}

func (fc *fedboxContainer) RemoteExec(ctx context.Context, cmd []string) (string, error) {
	//key, err := ssh.ParsePrivateKey([]byte(privateKey))
	//if err != nil {
	//	return "", err
	//}

	inspect, err := fc.Inspect(ctx)
	if err != nil {
		return "", err
	}

	if len(inspect.NetworkSettings.Ports) <= 1 {
		return "", errors.New("invalid ports")
	}
	host := "fedbox:4044"
	//for label, ports := range inspect.NetworkSettings.Ports {
	//	if strings.HasPrefix(string(label), "4044") && len(ports) > 0 {
	//		host = net.JoinHostPort(host, ports[0].HostPort)
	//	}
	//}
	// Authentication
	user := "http://fedbox"
	config := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		// TODO(marius): make sure the password or the private key are available
		//   Currently for TEST environments password check is skipped
		Auth: []ssh.AuthMethod{ssh.Password(defaultPassword) /*, ssh.PublicKeys(key)*/},
	}
	// Connect
	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return "", err
	}
	// Create a session. It is one session per command.
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	var b bytes.Buffer  // import "bytes"
	session.Stdout = &b // get output
	// you can also pass what gets input to the stdin, allowing you to pipe
	// content from client to server
	//      session.Stdin = bytes.NewBufferString("My input")

	// Finally, run the command
	err = session.Run(strings.Join(cmd, " "))
	return b.String(), err
}

var defaultPassword = "asd"

const ctlBin = "fedbox"

// Run creates an instance of the FedBOX container type
func Run(ctx context.Context, t testing.TB, opts ...containers.ContainerCustomizer) (*fedboxContainer, error) {
	logger := testLogger{T: t.(*testing.T)}

	req := defaultFedBOXRequest(fedboxImageName)
	opts = append(opts, containers.WithLogConsumers(logger))
	opts = append(opts, WithLogger(logger))
	for _, opt := range opts {
		if err := opt.Customize(&req); err != nil {
			return nil, err
		}
	}

	fc, err := containers.GenericContainer(ctx, req)
	if err != nil {
		return nil, err
	}
	f := fedboxContainer{Container: fc}

	if err = f.Start(ctx); err != nil {
		return &f, err
	}

	envType := extractEnvTagFromBuild()
	//storageType := extractStorageTagFromBuild()
	//if storageType == "all" {
	//	storageType = string(storage.Default)
	//}
	initializers := [][]string{
		{ctlBin, "--env", envType, /**/ "storage", /*"--type", storageType,*/ "bootstrap", "--password", defaultPassword},
		{ctlBin, "--env", envType, /**/ "pub", "import", "/storage/import.json"},
		{ctlBin, "--env", envType, /**/ "accounts", "gen-keys", "--key-type", "ED25519"},
	}
	errs := make([]error, 0)
	for _, cmd := range initializers {
		if cmd[0] == ctlBin {
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
		} else {
			if _, err := f.RemoteExec(ctx, cmd); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return &f, errors.Join(errs...)
}

var envKeys = []string{
	"LISTEN_PATH", "HTTP_PORT", "SSH_PORT",
	"STORAGE_PATH", "STORAGE", "LOG_LEVEL", "LOG_OUTPUT",
	"HOSTNAME", "HTTPS", "CERT_PATH", "KEY_PATH",
	"ENV", "TIME_OUT",
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

func WithImageName(image string) containers.CustomizeRequestOption {
	return func(req *containers.GenericContainerRequest) error {
		if req.Image == "" {
			req.Image = image
		}
		return nil
	}
}

func WithLogger(logFn log.Logger) containers.CustomizeRequestOption {
	return func(req *containers.GenericContainerRequest) error {
		req.Logger = logFn
		return nil
	}
}

func WithEnvFile(configFile string) containers.CustomizeRequestOption {
	_ = godotenv.Load(configFile)

	storageType := extractStorageTagFromBuild()
	envType := extractEnvTagFromBuild()
	if storageType == "all" {
		storageType = string(storage.Default)
	}
	return func(req *containers.GenericContainerRequest) error {
		if req.Env == nil {
			req.Env = make(map[string]string)
		}
		for k, v := range loadEnv() {
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
		if Verbose {
			req.Env["LOG_LEVEL"] = "trace"
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
