package assets

import (
	"io/fs"
	"path/filepath"
	"strings"
)

const TemplatesPath = "templates"

// Files returns asset names necessary for unrolled.Render
func Files() []string {
	names := make([]string, 0)
	fs.WalkDir(Templates, ".", func(path string, d fs.DirEntry, err error) error {
		if d == nil || d.IsDir() || !strings.HasSuffix(filepath.Dir(path), TemplatesPath) {
			return nil
		}
		names = append(names, path)
		return nil
	})
	return names
}

// Template returns an asset by path for unrolled.Render
func Template(name string) ([]byte, error) {
	return fs.ReadFile(Templates, filepath.Clean(name))
}
