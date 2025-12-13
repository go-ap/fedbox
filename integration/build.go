package integration

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/moby/sys/capability"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"go.podman.io/image/v5/transports/alltransports"
	"go.podman.io/storage"
	"go.podman.io/storage/pkg/idtools"
	"go.podman.io/storage/pkg/unshare"
	"go.podman.io/storage/types"
)

const (
	//baseImage     = "cgr.dev/chainguard/static:latest"
	baseImage     = "gcr.io/distroless/static:latest"
	targetRepo    = "localhost"
	containerName = "fedbox"
	importPath    = "github.com/go-ap/fedbox"
	commitSHA     = "deadbeef"
)

var cachePath, _ = os.UserCacheDir()

//var basePath = filepath.Join(os.TempDir(), "fedbox-test")
var basePath = filepath.Join(cachePath, "fedbox-test")

func buildImage(ctx context.Context, imageName string, logger *logrus.Logger) (string, error) {
	buildah.InitReexec()
	//unshare.MaybeReexecUsingUserNamespace(true)
	_ = os.Setenv(unshare.UsernsEnvName, "done")
	if err := MaybeReexecUsingUserNamespace(); err != nil {
		return "", err
	}

	buildStoreOptions, err := storage.DefaultStoreOptions()
	if err != nil {
		return "", err
	}

	buildStoreOptions.RunRoot = filepath.Join(basePath, "root")
	buildStoreOptions.GraphRoot = filepath.Join(basePath, "graph")
	buildStoreOptions.RootlessStoragePath = filepath.Join(basePath, "rootless")
	buildStoreOptions.RootAutoNsUser = filepath.Join(basePath, "rootless")
	buildStoreOptions.GraphDriverName = "vfs"
	//buildStoreOptions.GraphDriverName = "btrfs"
	//buildStoreOptions.GraphDriverName = "aufs" // not supported
	//buildStoreOptions.GraphDriverName = "overlay"
	//buildStoreOptions.GraphDriverName = "overlay2"
	//buildStoreOptions.GraphDriverOptions = []string{ // Overlay options
	//"mountopt=nodev",
	//"mount_program=/usr/bin/fuse-overlayfs",
	//"force_mask=shared",
	//"skip_mount_home=true",
	//"ignore_chown_errors=true",
	//"use_composefs=true", // error building: composefs is not supported in user namespaces
	//}
	//buildStoreOptions.DisableVolatile = true
	//buildStoreOptions.TransientStore = true
	//buildStoreOptions.UIDMap = []idtools.IDMap{}
	//buildStoreOptions.GIDMap = []idtools.IDMap{}

	/*
		chown /home/habarnam/.cache/podman/fedbox-test/graph/vfs/dir/bff7f7a9d44356d8784500366094c66399aa6a2edd990cc70e02e27c84402753: operation not permitted
	*/
	store, err := storage.GetStore(buildStoreOptions)
	if err != nil {
		return "", err
	}

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
	namespaces, err := buildah.DefaultNamespaceOptions()
	if err != nil {
		return "", err
	}
	namespaces.AddOrReplace(define.NamespaceOption{Name: string(specs.MountNamespace), Host: true})

	//conf, err := config.Default()
	//if err != nil {
	//	return "", err
	//}
	//conf.Secrets.Opts["seccomp"] = "unconfined"
	//conf.Secrets.Opts["apparmor"] = "unconfined"

	//uidStr := strconv.Itoa(os.Geteuid())
	//capabilities, err := conf.Capabilities(uidStr, nil, nil)
	//if err != nil {
	//	return "", err
	//}

	buildOpts := buildah.BuilderOptions{
		//Args:         nil,
		//FromImage: baseImage,
		//Capabilities: capabilities,
		Container:        containerName,
		Logger:           logger,
		Mount:            true,
		ReportWriter:     os.Stderr,
		Isolation:        buildah.IsolationChroot,
		NamespaceOptions: namespaces,
		//ConfigureNetwork: 0,
		//NetworkInterface: net,
		IDMappingOptions: &define.IDMappingOptions{
			AutoUserNs: true,
			AutoUserNsOpts: types.AutoUserNsOptions{
				Size:        4096,
				InitialSize: 1024,
				AdditionalUIDMappings: []idtools.IDMap{
					{ContainerID: 10000, HostID: 1, Size: 4096},
				},
				AdditionalGIDMappings: []idtools.IDMap{
					{ContainerID: 10000, HostID: 1, Size: 4096},
				},
			},
		},
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

// MaybeReexecUsingUserNamespace re-exec the process in a new namespace
func MaybeReexecUsingUserNamespace() error {
	// If we've already been through this once, no need to try again.
	if os.Geteuid() == 0 && unshare.GetRootlessUID() > 0 {
		return nil
	}

	var uidNum, gidNum uint64
	// Figure out who we are.
	me, err := user.Current()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("error determining current user: %w", err)
		}
		uidNum, err = strconv.ParseUint(me.Uid, 10, 32)
		if err != nil {
			return fmt.Errorf("error parsing current UID %s: %w", me.Uid, err)
		}
		gidNum, err = strconv.ParseUint(me.Gid, 10, 32)
		if err != nil {
			return fmt.Errorf("error parsing current GID %s: %w", me.Gid, err)
		}
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// ID mappings to use to reexec ourselves.
	var uidmap, gidmap []specs.LinuxIDMapping
	if uidNum != 0 || false {
		// Read the set of ID mappings that we're allowed to use.  Each
		// range in /etc/subuid and /etc/subgid file is a starting host
		// ID and a range size.
		uidmap, gidmap, err = unshare.GetSubIDMappings(me.Username, me.Username)
		if err != nil {
			logrus.Warnf("Reading allowed ID mappings: %v", err)
		}
		if len(uidmap) == 0 {
			logrus.Warnf("Found no UID ranges set aside for user %q in /etc/subuid.", me.Username)
		}
		if len(gidmap) == 0 {
			logrus.Warnf("Found no GID ranges set aside for user %q in /etc/subgid.", me.Username)
		}
		// Map our UID and GID, then the subuid and subgid ranges,
		// consecutively, starting at 0, to get the mappings to use for
		// a copy of ourselves.
		uidmap = append([]specs.LinuxIDMapping{{HostID: uint32(uidNum), ContainerID: 0, Size: 1}}, uidmap...)
		gidmap = append([]specs.LinuxIDMapping{{HostID: uint32(gidNum), ContainerID: 0, Size: 1}}, gidmap...)
		var rangeStart uint32
		for i := range uidmap {
			uidmap[i].ContainerID = rangeStart
			rangeStart += uidmap[i].Size
		}
		rangeStart = 0
		for i := range gidmap {
			gidmap[i].ContainerID = rangeStart
			rangeStart += gidmap[i].Size
		}
	} else {
		// If we have CAP_SYS_ADMIN, then we don't need to create a new namespace in order to be able
		// to use unshare(), so don't bother creating a new user namespace at this point.
		capabilities, err := capability.NewPid2(0)
		if err != nil {
			return fmt.Errorf("Initializing a new Capabilities object of pid 0: %w", err)
		}
		err = capabilities.Load()
		if err != nil {
			return fmt.Errorf("Reading the current capabilities sets: %w", err)
		}

		if capabilities.Get(capability.EFFECTIVE, capability.CAP_SYS_ADMIN) {
			return nil
		}
		// Read the set of ID mappings that we're currently using.
		uidmap, gidmap, err = unshare.GetHostIDMappings("")
		if err != nil {
			return fmt.Errorf("Reading current ID mappings: %w", err)
		}

		// Just reuse them.
		for i := range uidmap {
			uidmap[i].HostID = uidmap[i].ContainerID
		}
		for i := range gidmap {
			gidmap[i].HostID = gidmap[i].ContainerID
		}
	}

	// If, somehow, we don't become UID 0 in our child, indicate that the child shouldn't try again.
	//err = os.Setenv(unshare.UsernsEnvName, "1")
	//if err != nil {
	//	return fmt.Errorf("error setting %s=1 in environment: %w", unshare.UsernsEnvName, err)
	//}

	// Set the default isolation type to use the "rootless" method.
	if _, present := os.LookupEnv("BUILDAH_ISOLATION"); !present {
		if err = os.Setenv("BUILDAH_ISOLATION", "rootless"); err != nil {
			if err := os.Setenv("BUILDAH_ISOLATION", "rootless"); err != nil {
				return fmt.Errorf("Setting BUILDAH_ISOLATION=rootless in environment: %w", err)
			}
		}
	}

	return nil
}
