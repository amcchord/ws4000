package render

import (
	"image"
	"image/png"
	"os"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type Window struct {
	window     *sdl.Window
	renderer   *sdl.Renderer
	canvas     *Canvas
	fonts      *FontManager
	fullscreen bool
	scale      int
}

type WindowConfig struct {
	Title      string
	Fullscreen bool
	Scale      int
	Headless   bool // render to CPU canvas only (for --screenshot without a window)
}

func NewWindow(cfg WindowConfig) (*Window, error) {
	if cfg.Title == "" {
		cfg.Title = "WeatherStar 4000+"
	}
	if cfg.Scale <= 0 {
		cfg.Scale = 2
	}

	fonts := NewFontManager()

	if cfg.Headless {
		if err := ttf.Init(); err != nil {
			return nil, err
		}
		canvas, err := NewCanvas(nil, fonts)
		if err != nil {
			return nil, err
		}
		return &Window{canvas: canvas, fonts: fonts, scale: cfg.Scale}, nil
	}

	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_TIMER); err != nil {
		return nil, err
	}
	if err := ttf.Init(); err != nil {
		sdl.Quit()
		return nil, err
	}

	flags := uint32(sdl.WINDOW_SHOWN | sdl.WINDOW_RESIZABLE | sdl.WINDOW_ALLOW_HIGHDPI)
	if cfg.Fullscreen {
		flags |= sdl.WINDOW_FULLSCREEN_DESKTOP
	}

	window, err := sdl.CreateWindow(cfg.Title, sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED,
		int32(LogicalWidth*cfg.Scale), int32(LogicalHeight*cfg.Scale), flags)
	if err != nil {
		ttf.Quit()
		sdl.Quit()
		return nil, err
	}

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		window.Destroy()
		ttf.Quit()
		sdl.Quit()
		return nil, err
	}
	renderer.SetLogicalSize(LogicalWidth, LogicalHeight)

	canvas, err := NewCanvas(renderer, fonts)
	if err != nil {
		renderer.Destroy()
		window.Destroy()
		ttf.Quit()
		sdl.Quit()
		return nil, err
	}

	return &Window{
		window:     window,
		renderer:   renderer,
		canvas:     canvas,
		fonts:      fonts,
		fullscreen: cfg.Fullscreen,
		scale:      cfg.Scale,
	}, nil
}

func (w *Window) Canvas() *Canvas { return w.canvas }

func (w *Window) ToggleFullscreen() {
	if w.window == nil {
		return
	}
	w.fullscreen = !w.fullscreen
	if w.fullscreen {
		w.window.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
	} else {
		w.window.SetFullscreen(0)
	}
}

func (w *Window) Present() error {
	if w.renderer == nil {
		return nil
	}
	if err := w.canvas.PresentToRenderer(); err != nil {
		return err
	}
	w.renderer.SetDrawColor(0, 0, 0, 255)
	w.renderer.Clear()
	if err := w.renderer.Copy(w.canvas.texture, nil, nil); err != nil {
		return err
	}
	w.renderer.Present()
	return nil
}

// PollEvent returns the next pending event and whether quit was requested.
func (w *Window) PollEvent() (sdl.Event, bool) {
	if w.window == nil {
		return nil, false
	}
	event := sdl.PollEvent()
	if event == nil {
		return nil, false
	}
	switch e := event.(type) {
	case *sdl.QuitEvent:
		return event, true
	case *sdl.KeyboardEvent:
		if e.Type == sdl.KEYDOWN && (e.Keysym.Sym == sdl.K_ESCAPE || e.Keysym.Sym == sdl.K_q) {
			return event, true
		}
	}
	return event, false
}

func (w *Window) SaveScreenshot(path string) error {
	return w.canvas.SavePNG(path)
}

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func (w *Window) Close() {
	if w.canvas != nil {
		w.canvas.Close()
	}
	if w.fonts != nil {
		w.fonts.Close()
	}
	if w.renderer != nil {
		w.renderer.Destroy()
	}
	if w.window != nil {
		w.window.Destroy()
	}
	ttf.Quit()
	if w.window != nil {
		sdl.Quit()
	}
}
