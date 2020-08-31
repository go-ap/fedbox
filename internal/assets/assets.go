package assets

import (
	"bufio"
	"bytes"
	"os"
)

const (
	templateDir = "templates/"
)

// generated with broccoli - see ./assets_gen.go
var walkFsFn = assets.Walk
var openFsFn = assets.Open

type AssetFiles map[string][]string

// Files returns asset names necessary for unrolled.Render
func Files() []string {
	names := make([]string, 0)
	walkFsFn(templateDir, func(path string, info os.FileInfo, err error) error {
		if info != nil && !info.IsDir() {
			names = append(names, path)
		}
		return nil
	})
	return names
}

func getFileContent(name string) ([]byte, error) {
	f, err := openFsFn(name)
	if err != nil {
		return nil, err
	}
	r := bufio.NewReader(f)
	b := bytes.Buffer{}
	_, err = r.WriteTo(&b)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Template returns an asset by path for unrolled.Render
func Template(name string) ([]byte, error) {
	return getFileContent(name)
}
