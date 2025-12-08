package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/containers/buildah"
	"github.com/containers/common/pkg/config"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	"github.com/sirupsen/logrus"
)

const (
	//baseImage     = "cgr.dev/chainguard/static:latest"
	baseImage     = "gcr.io/distroless/static:latest"
	targetRepo    = "localhost"
	containerName = "fedbox"
	importPath    = "github.com/go-ap/fedbox"
	commitSHA     = "deadbeef"
)

var basePath = filepath.Join(os.TempDir(), "fedbox-test")

func buildImage(ctx context.Context, verbose bool) (string, error) {
	logrus.SetOutput(os.Stderr)
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableQuote: true, ForceColors: true})

	logger := logrus.New()

	buildah.InitReexec()

	buildStoreOptions, err := storage.DefaultStoreOptions()
	if err != nil {
		return "", err
	}

	buildStoreOptions.RunRoot = filepath.Join(basePath, "root")
	buildStoreOptions.GraphRoot = filepath.Join(basePath, "graph")
	buildStoreOptions.RootlessStoragePath = filepath.Join(basePath, "rootless")
	buildStoreOptions.RootAutoNsUser = filepath.Join(basePath, "rootless")
	//buildStoreOptions.GraphDriverName = "vfs"
	//buildStoreOptions.GraphDriverName = "btrfs"
	//buildStoreOptions.GraphDriverName = "aufs"
	buildStoreOptions.GraphDriverName = "overlay"
	//buildStoreOptions.GraphDriverOptions = []string{
	//"skip_mount_home=true",
	//"ignore_chown_errors=true",
	//"use_composefs=true", // error building: composefs is not supported in user namespaces
	//}
	//buildStoreOptions.DisableVolatile = true
	//buildStoreOptions.TransientStore = true
	//buildStoreOptions.UIDMap = stt.Transport.DefaultUIDMap()
	//buildStoreOptions.GIDMap = stt.Transport.DefaultGIDMap()

	/*
		chown /home/habarnam/.cache/podman/fedbox-test/graph/vfs/dir/bff7f7a9d44356d8784500366094c66399aa6a2edd990cc70e02e27c84402753: operation not permitted
	*/
	store, err := storage.GetStore(buildStoreOptions)
	if err != nil {
		return "", err
	}

	_ = os.Setenv(unshare.UsernsEnvName, "done")
	//netOpts := netavark.InitConfig{
	//	Config:           &config.Config{},
	//	NetworkConfigDir: filepath.Join(basePath, "net", "config"),
	//	NetworkRunDir:    filepath.Join(basePath, "net", "run"),
	//	NetavarkBinary:   "true",
	//}
	//net, err := netavark.NewNetworkInterface(&netOpts)
	//if err != nil {
	//	return "", err
	//}

	//commonBuildOpts := buildah.CommonBuildOptions{}
	//defaultEnv := []string{}

	// NOTE(marius): this fails with a mounting error.
	// The internet seems to suggest we need to force a user namespace creation when running rootless,
	// but I don't know how to do this programmatically, and they don't give any clues:
	// https://github.com/containers/buildah/issues/5744
	// https://github.com/containers/buildah/issues/3948
	// https://github.com/containers/buildah/issues/4489
	//namespaces, err := buildah.DefaultNamespaceOptions()
	//if err != nil {
	//	return "", err
	//}
	//namespaces.AddOrReplace(define.NamespaceOption{Name: string(specs.MountNamespace), Host: true})

	conf, err := config.Default()
	if err != nil {
		return "", err
	}
	//conf.Secrets.Opts["seccomp"] = "unconfined"
	//conf.Secrets.Opts["apparmor"] = "unconfined"

	uidStr := strconv.Itoa(os.Geteuid())
	capabilities, err := conf.Capabilities(uidStr, nil, nil)
	if err != nil {
		return "", err
	}

	buildOpts := buildah.BuilderOptions{
		//Args:         nil,
		FromImage:    baseImage,
		Capabilities: capabilities,
		Container:    containerName,
		Logger:       logger,
		//Mount:        true,
		ReportWriter: os.Stderr,
		//Isolation:    buildah.IsolationChroot,
		//NamespaceOptions: namespaces,
		//ConfigureNetwork: 0,
		//NetworkInterface: net,
		//IDMappingOptions: &define.IDMappingOptions{
		//	AutoUserNs: true,
		//	AutoUserNsOpts: types.AutoUserNsOptions{
		//		Size:        4096,
		//		InitialSize: 1024,
		//		AdditionalUIDMappings: []idtools.IDMap{
		//			{ContainerID: 10000, HostID: 1, Size: 4096},
		//		},
		//		AdditionalGIDMappings: []idtools.IDMap{
		//			{ContainerID: 10000, HostID: 1, Size: 4096},
		//		},
		//	},
		//},
		//CommonBuildOpts: &commonBuildOpts,
		//Format:                "",
		//Devices:               nil,
		//DeviceSpecs:           nil,
		//DefaultEnv: defaultEnv,
	}
	builder, err := buildah.NewBuilder(ctx, store, buildOpts)
	if err != nil {
		return "", err
	}

	img, err := alltransports.ParseImageName("localhost/fedbox/app:dev")
	if err != nil {
		return "", err
	}
	commitOpts := buildah.CommitOptions{
		//PreferredManifestType:       "",
		//Compression: archive.Gzip,
		//AdditionalTags:              nil,
		ReportWriter: os.Stderr,
		//HistoryTimestamp:            nil,
		//SystemContext:               nil,
		//IIDFile:                     "",
		//Squash:                      false,
		//SignBy:                      "",
		//Manifest:                    "",
		//ExtraImageContent:           nil,
	}
	hash, canonical, digest, err := builder.Commit(ctx, img, commitOpts)
	if err != nil {
		return "", err
	}
	fmt.Printf("hash: %s\n", hash)
	fmt.Printf("digest: %s\n", digest)
	fmt.Printf("canonical: %s\n", canonical)
	return hash, nil
}
