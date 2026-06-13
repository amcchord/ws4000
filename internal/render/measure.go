package render

import (
	"sync"

	"github.com/veandco/go-sdl2/ttf"
)

var (
	measureMu  sync.Mutex
	measureMgr *FontManager
)

// MeasureString returns the pixel width of text in the given font family and
// size. Safe to call from any goroutine (used by text wrapping during fetch).
func MeasureString(family string, size int, text string) int {
	measureMu.Lock()
	defer measureMu.Unlock()
	if !ttf.WasInit() {
		if err := ttf.Init(); err != nil {
			return len(text) * size / 2
		}
	}
	if measureMgr == nil {
		measureMgr = NewFontManager()
	}
	font, err := measureMgr.Get(family, size)
	if err != nil {
		return len(text) * size / 2
	}
	w, _, err := font.SizeUTF8(text)
	if err != nil {
		return len(text) * size / 2
	}
	return w
}
