package integration

import (
	"context"
	"path/filepath"
	"runtime/debug"
	"strings"

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

func extractStorageTagsFromBuild() string {
	storageType := "storage_fs"
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, bs := range info.Settings {
			if bs.Key == "-tags" {
				for _, tt := range strings.Split(bs.Value, " ") {
					if strings.HasPrefix(tt, "storage_") {
						storageType = tt
					}
				}
			}
		}
	}
	return "-tags " + storageType
}

func buildImage(ctx context.Context, imageName string, _ *logrus.Logger) (string, error) {
	builder, err := build.NewGo(ctx, "",
		//build.WithDebugger(), // NOTE(marius): we're using statically linked fedbox, and a minimal base image, so we can't use Delve
		build.WithDefaultEnv([]string{
			"ENV=dev",
			"HOSTNAME=fedbox",
			"HTTP_PORT=4000",
			"SSH_PORT=4044",
			"HTTPS=true",
			"STORAGE=fs",
		}),
		build.WithBaseImages(func(ctx context.Context, _ string) (name.Reference, build.Result, error) {
			ref := name.MustParseReference(baseImage)
			base, err := remote.Index(ref, remote.WithContext(ctx))
			return ref, base, err
		}),
		build.WithPlatforms("linux/amd64"),
		build.WithDefaultLdflags([]string{`-extldflags "-static"`}),
		build.WithConfig(map[string]build.Config{
			"cmd/fedbox": {
				Dir:   "cmd/fedbox",
				Flags: []string{extractStorageTagsFromBuild()},
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
		LocalDomain:         targetRepo,
		Tags:                []string{"dev"},
		TagOnly:             true,
		Push:                true,
		Local:               true,
		InsecureRegistry:    true,
		PreserveImportPaths: true,
		Bare:                true,
		ImageNamer:          justName,
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
