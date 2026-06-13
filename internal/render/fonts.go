package render

import (
	"fmt"
	"os"
	"sync"

	"github.com/amcchord/ws4000/internal/assets"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

// Font families matching upstream CSS font-family names.
const (
	FontStar4000         = "star4000"
	FontStar4000Large    = "star4000-large"
	FontStar4000Small    = "star4000-small"
	FontStar4000Extended = "star4000-extended"
	FontArialBold        = "arial-bold"
)

var fontAssetPaths = map[string]string{
	FontStar4000:         "fonts/ttf/Star4000.ttf",
	FontStar4000Large:    "fonts/ttf/Star4000 Large.ttf",
	FontStar4000Small:    "fonts/ttf/Star4000 Small.ttf",
	FontStar4000Extended: "fonts/ttf/Star4000 Extended.ttf",
}

// Candidate system paths for Arial Bold (radar title uses Arial per upstream CSS).
var arialBoldPaths = []string{
	"/System/Library/Fonts/Supplemental/Arial Bold.ttf",
	"/Library/Fonts/Arial Bold.ttf",
	"C:\\Windows\\Fonts\\arialbd.ttf",
	"/usr/share/fonts/truetype/msttcorefonts/Arial_Bold.ttf",
	"/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf",
	"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
}

type fontKey struct {
	family string
	size   int
}

// FontManager caches ttf.Font handles per (family, pixel size).
type FontManager struct {
	mu    sync.Mutex
	fonts map[fontKey]*ttf.Font
	// font file bytes must outlive the ttf.Font that references them
	data map[string][]byte
}

func NewFontManager() *FontManager {
	return &FontManager{
		fonts: make(map[fontKey]*ttf.Font),
		data:  make(map[string][]byte),
	}
}

func (m *FontManager) Get(family string, size int) (*ttf.Font, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fontKey{family, size}
	if f, ok := m.fonts[key]; ok {
		return f, nil
	}

	raw, ok := m.data[family]
	if !ok {
		var err error
		raw, err = m.loadFontBytes(family)
		if err != nil {
			return nil, err
		}
		m.data[family] = raw
	}

	rw, err := sdl.RWFromMem(raw)
	if err != nil {
		return nil, err
	}
	font, err := ttf.OpenFontRW(rw, 1, size)
	if err != nil {
		return nil, err
	}
	m.fonts[key] = font
	return font, nil
}

func (m *FontManager) loadFontBytes(family string) ([]byte, error) {
	if path, ok := fontAssetPaths[family]; ok {
		return assets.Read(path)
	}
	if family == FontArialBold {
		for _, p := range arialBoldPaths {
			if data, err := os.ReadFile(p); err == nil {
				return data, nil
			}
		}
		// fall back to Star4000 if no system Arial available
		return assets.Read(fontAssetPaths[FontStar4000])
	}
	return nil, fmt.Errorf("unknown font family %q", family)
}

func (m *FontManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, f := range m.fonts {
		f.Close()
	}
	m.fonts = make(map[fontKey]*ttf.Font)
}
