//go:build !(prod || qa)

package assets

import "os"

const TemplatesPath = "."

var Templates = os.DirFS("./internal/assets/templates")
