package render

import (
	"fmt"
	"image/png"
	"os"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type Window struct {
	window     *sdl.Window
	renderer   *sdl.Renderer
	canvas     *Canvas
	fonts      *FontSet
	fullscreen bool
	scale      int
	logicalW   int
	logicalH   int
}

type WindowConfig struct {
	Title      string
	Fullscreen bool
	Scale      int
	Scanlines  bool
}

func NewWindow(cfg WindowConfig) (*Window, error) {
	if cfg.Title == "" {
		cfg.Title = "WeatherStar 4000+"
	}
	if cfg.Scale <= 0 {
		cfg.Scale = 2
	}

	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_TIMER); err != nil {
		return nil, err
	}
	if err := ttfInit(); err != nil {
		sdl.Quit()
		return nil, err
	}

	flags := uint32(sdl.WINDOW_SHOWN | sdl.WINDOW_RESIZABLE)
	if cfg.Fullscreen {
		flags |= sdl.WINDOW_FULLSCREEN_DESKTOP
	}

	winW := LogicalWidth * cfg.Scale
	winH := LogicalHeight * cfg.Scale

	window, err := sdl.CreateWindow(cfg.Title, sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED, int32(winW), int32(winH), flags)
	if err != nil {
		ttfQuit()
		sdl.Quit()
		return nil, err
	}

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		window.Destroy()
		ttfQuit()
		sdl.Quit()
		return nil, err
	}
	renderer.SetLogicalSize(LogicalWidth, LogicalHeight)
	renderer.SetIntegerScale(true)

	fonts, err := LoadFonts()
	if err != nil {
		renderer.Destroy()
		window.Destroy()
		ttfQuit()
		sdl.Quit()
		return nil, err
	}

	canvas, err := NewCanvas(renderer, fonts)
	if err != nil {
		fonts.Close()
		renderer.Destroy()
		window.Destroy()
		ttfQuit()
		sdl.Quit()
		return nil, err
	}
	canvas.DrawScanlines(cfg.Scanlines)

	return &Window{
		window:     window,
		renderer:   renderer,
		canvas:     canvas,
		fonts:      fonts,
		fullscreen: cfg.Fullscreen,
		scale:      cfg.Scale,
		logicalW:   LogicalWidth,
		logicalH:   LogicalHeight,
	}, nil
}

func ttfInit() error {
	return ttf.Init()
}

func ttfQuit() {
	ttf.Quit()
}

func (w *Window) Canvas() *Canvas {
	return w.canvas
}

func (w *Window) ToggleFullscreen() {
	w.fullscreen = !w.fullscreen
	if w.fullscreen {
		w.window.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
	} else {
		w.window.SetFullscreen(0)
	}
}

func (w *Window) IsFullscreen() bool {
	return w.fullscreen
}

func (w *Window) Present() error {
	if err := w.canvas.PresentToRenderer(); err != nil {
		return err
	}
	w.renderer.SetRenderTarget(nil)
	w.renderer.SetDrawColor(0, 0, 0, 255)
	w.renderer.Clear()
	if err := w.renderer.Copy(w.canvas.texture, nil, nil); err != nil {
		return err
	}
	w.renderer.Present()
	return nil
}

func (w *Window) PollEvent() (sdl.Event, bool) {
	for {
		event := sdl.PollEvent()
		if event == nil {
			return nil, false
		}
		switch e := event.(type) {
		case *sdl.QuitEvent:
			return event, true
		case *sdl.KeyboardEvent:
			if e.Type == sdl.KEYDOWN {
				if e.Keysym.Sym == sdl.K_ESCAPE || e.Keysym.Sym == sdl.K_q {
					return event, true
				}
				return event, false
			}
		}
	}
}

func (w *Window) SaveScreenshot(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, w.canvas.Snapshot())
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
	ttfQuit()
	sdl.Quit()
}

func (f *FontSet) Close() {
	if f.Regular != nil {
		f.Regular.Close()
	}
	if f.Extended != nil {
		f.Extended.Close()
	}
	if f.Large != nil {
		f.Large.Close()
	}
	if f.Small != nil {
		f.Small.Close()
	}
}

func (w *Window) String() string {
	return fmt.Sprintf("window scale=%d fullscreen=%v", w.scale, w.fullscreen)
}
