//go:build prod || qa

package assets

import "embed"

//go:embed templates/*
var Templates embed.FS
