package assets

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	rootOnce sync.Once
	rootPath string
	rootErr  error
)

func Root() (string, error) {
	rootOnce.Do(func() {
		// an explicit WS4000_ASSETS always wins
		candidates := []string{
			os.Getenv("WS4000_ASSETS"),
			"assets/upstream",
			"/usr/share/ws4000/assets",
		}
		if exe, err := os.Executable(); err == nil {
			candidates = append(candidates, filepath.Join(filepath.Dir(exe), "assets", "upstream"))
		}
		for _, c := range candidates {
			if c == "" {
				continue
			}
			if st, err := os.Stat(c); err == nil && st.IsDir() {
				rootPath = c
				return
			}
		}
		rootErr = fmt.Errorf("assets not found; run ./tools/vendor-assets.sh")
	})
	return rootPath, rootErr
}

func Read(path string) ([]byte, error) {
	root, err := Root()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
}

func MustRead(path string) []byte {
	data, err := Read(path)
	if err != nil {
		panic(err)
	}
	return data
}

func Walk(fn func(path string, data []byte) error) error {
	root, err := Root()
	if err != nil {
		return err
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return fn(filepath.ToSlash(rel), data)
	})
}

func FontPath(variant string) string {
	name := "Star4000.ttf"
	switch strings.ToLower(variant) {
	case "extended":
		name = "Star4000 Extended.ttf"
	case "large":
		name = "Star4000 Large.ttf"
	case "small":
		name = "Star4000 Small.ttf"
	}
	return filepath.ToSlash(filepath.Join("fonts", "ttf", name))
}

func BackgroundPath(displayID string) string {
	mapping := map[string]string{
		"current-weather":     "1.png",
		"latest-observations": "2.png",
		"hourly":              "3.png",
		"hourly-graph":        "1-chart.png",
		"travel":              "4.png",
		"regional-forecast":   "5.png",
		"local-forecast":      "6.png",
		"extended-forecast":   "7.png",
		"almanac":             "3.png",
		"spc-outlook":         "4.png",
		"radar":               "1.png",
		"hazards":             "2.png",
		"progress":            "1.png",
	}
	file, ok := mapping[displayID]
	if !ok {
		file = "1.png"
	}
	return filepath.ToSlash(filepath.Join("backgrounds", file))
}
