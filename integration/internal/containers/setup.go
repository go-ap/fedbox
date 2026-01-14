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
	"net"
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
	"github.com/testcontainers/testcontainers-go/log"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/ssh"
)

type fboxContainer struct {
	tc.Container
}

type Running []tc.Container

type image struct {
	name      string
	env       map[string]string
	startCmds []tc.Executable
}

type ContainerInitializer interface {
	Name() string
	InitFns() []tc.ContainerCustomizer
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

func (i image) Name() string {
	return i.name
}

func (i image) InitFns() []tc.ContainerCustomizer {
	return []tc.ContainerCustomizer{WithImage(i.Name()), WithEnvFile(i.env)}
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

func (f fboxImage) Name() string {
	return f.name
}

func (f fboxImage) InitFns() []tc.ContainerCustomizer {
	initFns := []tc.ContainerCustomizer{WithImage(f.name), WithEnvFile(f.env)}
	if len(f.mocks) > 0 {
		// NOTE(marius): the default import for the mocks
		initFns = append(initFns, WithMocks(f.mocks...))
		if f.pw != nil || f.key != nil {
			// NOTE(marius): we also need to add an SSH command to import the mocks file
			importCmd := sshCmd{
				cmd:  []string{ /*ctlBin, "--env", envType, */ "pub", "import", "/storage/import.json"},
				user: f.user,
			}
			if len(f.user) > 0 {
			}
			if f.key != nil {
				importCmd.key = f.key
				initFns = append(initFns, WithPrivateKey(f.key))
			}
			if f.pw != nil {
				importCmd.pw = f.pw
				initFns = append(initFns, WithPassword(f.pw))
			}
			initFns = append(initFns, tc.WithAfterReadyCommand(importCmd))
		}
	}
	return initFns
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

type sshCmd struct {
	cmd  []string
	user string
	pw   []byte
	key  crypto.PrivateKey
}

func (c sshCmd) AsCommand() []string {
	return c.cmd
}

func (c sshCmd) Options() []exec.ProcessOption {
	prvEnc, err := x509.MarshalPKCS8PrivateKey(c.key)
	if err != nil {
		return nil
	}

	// NOTE(marius): we pass the private key and the password through env variables.
	env := []string{
		"_PK=" + string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: prvEnc})),
		"_PW=" + string(c.pw),
	}
	return []exec.ProcessOption{exec.WithEnv(env), exec.WithUser(c.user)}
}

var _ tc.Executable = sshCmd{}

func Init(ctx context.Context, t *testing.T, s ...ContainerInitializer) (Running, error) {
	// NOTE(marius): the docker host can come from multiple places.
	// @see github.com/testcontainers/testcontainers-go/internal/core.MustExtractDockerHost()
	m := make(Running, 0)
	for _, img := range s {
		c, err := startContainer(ctx, t, img.Name(), img.InitFns()...)
		if err != nil {
			return nil, fmt.Errorf("unable to initialize container %T: %w", img, err)
		}

		m = append(m, *c)
	}

	m.Cleanup(t)
	return m, nil
}

func (m Running) Cleanup(t *testing.T) {
	for _, mm := range m {
		tc.CleanupContainer(t, mm)
	}
}

func (m Running) BuildRequest(ctx context.Context, met, u string, body io.Reader) (*http.Request, error) {
	uu, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("received invalid url: %w", err)
	}

	for _, fc := range m {
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
	envType := extractEnvTagFromBuild()
	return tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image:      name,
			Cmd:        []string{"--env", envType, "--bootstrap"},
			WaitingFor: wait.ForLog("Started").WithStartupTimeout(800 * time.Millisecond),
		},
		ProviderType: tc.ProviderPodman,
		Started:      true,
	}
}

func (fc fboxContainer) Exec(ctx context.Context, cmd []string, opts ...exec.ProcessOption) (int, io.Reader, error) {
	if cmd[0] == ctlBin {
		// NOTE(marius): if the command actually wants to call the "fedbox" binary,
		// we execute it using the docker exec mechanism
		// Otherwise, we treat this as a command that needs to be run IN SSH.
		return fc.Container.Exec(ctx, cmd, opts...)
	}

	host, _ := fc.Host(ctx)
	portsMap, err := fc.Ports(ctx)
	if err != nil {
		return 0, nil, err
	}
	conf := exec.ProcessOptions{}
	for _, opt := range opts {
		opt.Apply(&conf)
	}

	for label, ports := range portsMap {
		if strings.HasPrefix(string(label), "4044") && len(ports) > 0 {
			host = net.JoinHostPort(host, ports[0].HostPort)
		}
	}

	// NOTE(marius): extract authorization mechanisms from env variables
	initFns := make([]ssh.AuthMethod, 0)
	for _, env := range conf.ExecConfig.Env {
		if strings.HasPrefix(env, "_PK=") {
			prv, err := ssh.ParseRawPrivateKey([]byte(strings.TrimPrefix(env, "_PK=")))
			if err != nil {
				continue
			}
			sig, err := ssh.NewSignerFromKey(prv)
			if err != nil {
				continue
			}
			initFns = append(initFns, ssh.PublicKeys(sig))
		}
		if strings.HasPrefix(env, "_PW=") {
			pw := strings.TrimPrefix(env, "_PW=")
			initFns = append(initFns, ssh.Password(pw))
		}
	}
	if len(initFns) == 0 {
		return 0, nil, fmt.Errorf("no SSH authorization methods found")
	}
	if len(conf.ExecConfig.User) == 0 {
		return 0, nil, fmt.Errorf("no user for SSH authorization found")
	}
	config := &ssh.ClientConfig{
		User:            conf.ExecConfig.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth:            initFns,
	}

	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return 0, nil, err
	}
	// Create a session. It is one session per command.
	session, err := client.NewSession()
	if err != nil {
		return 0, nil, err
	}
	defer session.Close()
	errBuff := bytes.Buffer{}
	outBuff := bytes.Buffer{}
	session.Stderr = &errBuff
	session.Stdout = &outBuff

	// Finally, run the command
	if err = session.Run(strings.Join(cmd, " ")); err != nil {
		return 0, io.MultiReader(&outBuff, &errBuff), fmt.Errorf("%w: %s\n%s", err, outBuff.String(), errBuff.String())
	}
	return 0, io.MultiReader(&outBuff, &errBuff), nil
}

const ctlBin = "fedbox"

// startContainer creates an instance of the FedBOX container type
func startContainer(ctx context.Context, t testing.TB, image string, opts ...tc.ContainerCustomizer) (*fboxContainer, error) {
	logger := testLogger{T: t.(*testing.T)}

	req := defaultFedBOXRequest(image)
	opts = append(opts, tc.WithLogConsumers(logger))
	opts = append(opts, WithLogger(logger))
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
	f := fboxContainer{Container: fc}
	if err = f.Start(ctx); err != nil {
		return &f, err
	}
	if len(cmds) > 0 {
		name, _ := f.Name(ctx)
		for _, ex := range cmds {
			if err = ex(ctx, f); err != nil {
				return nil, fmt.Errorf("unable to run startup commands on container %s[%T]: %w", name, f, err)
			}
		}
	}

	return &f, nil
}

func WithImage(image string) tc.CustomizeRequestOption {
	return func(req *tc.GenericContainerRequest) error {
		if req.Image == "" {
			req.Image = image
		}
		return nil
	}
}

func WithLogger(logFn log.Logger) tc.CustomizeRequestOption {
	return func(req *tc.GenericContainerRequest) error {
		req.Logger = logFn
		return nil
	}
}

func WithEnvFile(env map[string]string) tc.CustomizeRequestOption {
	storageType := extractStorageTagFromBuild()
	envType := extractEnvTagFromBuild()
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
