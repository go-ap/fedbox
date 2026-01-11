package integration

import (
	"context"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"

	"git.sr.ht/~mariusor/storage-all"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/ko/pkg/build"
	"github.com/google/ko/pkg/commands"
	"github.com/google/ko/pkg/commands/options"
	"github.com/sirupsen/logrus"
)

const (
	//baseImage     = "cgr.dev/chainguard/static:latest"
	baseImage  = "gcr.io/distroless/static:latest"
	targetRepo = "localhost"

	importPath = "github.com/go-ap/fedbox"
)

func justName(s string, s2 string) string {
	return s2
}

var validEnvs = []string{"dev", "prod", "qa", "test"}

var buildInfo, buildOk = debug.ReadBuildInfo()

func extractEnvTagFromBuild() string {
	env := "dev"
	if buildOk {
		for _, bs := range buildInfo.Settings {
			if bs.Key == "-tags" {
				for _, tt := range strings.Split(bs.Value, " ") {
					if slices.Contains(validEnvs, tt) {
						env = tt
						break
					}
				}
			}
		}
	}
	return env
}

func extractStorageTagFromBuild() string {
	storageType := "all"
	if buildOk {
		for _, bs := range buildInfo.Settings {
			if bs.Key == "-tags" {
				for _, tt := range strings.Split(bs.Value, " ") {
					if strings.HasPrefix(tt, "storage_") {
						storageType = strings.TrimPrefix(tt, "storage_")
					}
				}
			}
		}
	}
	return storageType
}

func buildImage(ctx context.Context, imageName string, _ *logrus.Logger) (string, error) {
	storageType := extractStorageTagFromBuild()
	envType := extractEnvTagFromBuild()
	tags := `-tag "ssh storage_` + storageType + " " + envType + `" `
	if storageType == "all" {
		storageType = string(storage.Default)
	}

	builder, err := build.NewGo(ctx, "",
		//build.WithDebugger(), // NOTE(marius): we're using a minimal base image, requiring a statically compiled app, so we can't use Delve
		build.WithAnnotation("storage", storageType),
		build.WithAnnotation("env", envType),
		build.WithBaseImages(func(ctx context.Context, _ string) (name.Reference, build.Result, error) {
			ref := name.MustParseReference(baseImage)
			base, err := remote.Index(ref, remote.WithContext(ctx))
			return ref, base, err
		}),
		build.WithPlatforms("linux/amd64"),
		build.WithConfig(map[string]build.Config{
			"cmd/fedbox": {
				Dir:     "cmd/fedbox",
				Flags:   []string{tags},
				Ldflags: []string{`-extldflags "-static"`},
			},
		}),
		build.WithTrimpath(true),
		build.WithDisabledSBOM(),
	)
	if err != nil {
		return "", err
	}
	res, err := builder.Build(ctx, filepath.Join(importPath, "cmd/fedbox"))
	if err != nil {
		return "", err
	}
	publishOpts := options.PublishOptions{
		LocalDomain: targetRepo,
		DockerRepo:  targetRepo,
		Tags:        []string{storageType},
		//TagOnly:     true,
		//Push:        false,
		Local: true,
		//InsecureRegistry:    true,
		//PreserveImportPaths: true,
		//Bare:                true,
		ImageNamer: justName,
	}
	pub, err := commands.NewPublisher(&publishOpts)
	if err != nil {
		return "", err
	}
	ref, err := pub.Publish(ctx, res, strings.TrimPrefix(imageName, targetRepo+"/"))
	if err != nil {
		return "", err
	}
	return ref.String(), nil
}
