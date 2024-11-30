package integration

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	containers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type fedboxContainer struct {
	containers.Container
	ServiceURL string
}

// Run creates an instance of the FedBOX container type
func Run(ctx context.Context, image string, opts ...containers.ContainerCustomizer) (containers.Container, error) {
	req := containers.ContainerRequest{
		Image: image,
		LogConsumerCfg: &containers.LogConsumerConfig{
			Opts:      []containers.LogProductionOption{containers.WithLogProductionTimeout(10 * time.Second)},
			Consumers: []containers.LogConsumer{new(containers.StdoutLogConsumer)},
		},
		WaitingFor: wait.ForLog("Started").WithStartupTimeout(2 * time.Second),
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
		return fc, err
	}

	if err = fc.Start(ctx); err != nil {
		return fc, err
	}
	return fc, nil
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
			ContainerFilePath: filepath.Join("/storage", strings.ReplaceAll(path, storage, "")),
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
