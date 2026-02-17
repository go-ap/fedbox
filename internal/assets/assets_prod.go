//go:build prod || qa

package assets

import "embed"

const TemplatesPath = "templates"

//go:embed templates/*
var Templates embed.FS

//go:embed assets/*
var Assets embed.FS
