//go:build !prod && !qa

package assets

import "os"

var Templates = os.DirFS("./internal/assets")
