package integration

import (
	"context"
	"path/filepath"

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
	imageName  = "fedbox/app"
	importPath = "github.com/go-ap/fedbox"
	commitSHA  = "deadbeef"
)

func justName(s string, s2 string) string {
	return s2
}

func buildImage(ctx context.Context, imageName string, logger *logrus.Logger) (string, error) {
	builder, err := build.NewGo(ctx, "",
		//build.WithDebugger(),
		build.WithDefaultEnv([]string{
			"ENV=dev",
			"HOSTNAME=fedbox",
			"HTTP_PORT=4000",
			"SSH_PORT=4044",
			"HTTPS=true",
			"STORAGE=all",
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
				Flags: []string{"-tags storage_fs"},
			},
		}),
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
		Tags:        []string{"dev"},
		TagOnly:     true,
		Push:        true,
		Local:       true,
		//InsecureRegistry:    true,
		//PreserveImportPaths: true,
		//Bare:                true,
		ImageNamer: justName,
	}
	pub, err := commands.NewPublisher(&publishOpts)
	if err != nil {
		return "", err
	}
	ref, err := pub.Publish(ctx, res, "kofedbox")
	if err != nil {
		return "", err
	}
	return ref.String(), nil
}
