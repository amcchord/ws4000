package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/amcchord/ws4000/internal/config"
	"github.com/amcchord/ws4000/internal/data/weather"
	"github.com/amcchord/ws4000/internal/displays"
	"github.com/amcchord/ws4000/internal/engine"
	"github.com/amcchord/ws4000/internal/music"
	"github.com/amcchord/ws4000/internal/progress"
	"github.com/amcchord/ws4000/internal/render"
	"github.com/veandco/go-sdl2/sdl"
)

type App struct {
	cfg      config.Config
	window   *render.Window
	nav      *engine.Navigator
	svc      *weather.Service
	progress *progress.Display
	music    *music.Player
	params   *engine.WeatherParams
	playing  bool
	tickerX  int
	ticker   string
	showProgress bool
}

func New(cfg config.Config) (*App, error) {
	scale := cfg.Scale
	if scale <= 0 {
		scale = 2
	}
	win, err := render.NewWindow(render.WindowConfig{
		Title:      "WeatherStar 4000+",
		Fullscreen: cfg.Fullscreen,
		Scale:      scale,
		Scanlines:  cfg.Scanlines,
	})
	if err != nil {
		return nil, err
	}

	cacheDir := filepath.Join(os.TempDir(), "ws4000-cache")
	svc := weather.NewService(cacheDir, cfg.FixtureDir)
	nav := engine.NewNavigator(cfg.Speed)
	displays.RegisterAll(nav, svc, cfg)

	player, err := music.NewPlayer(cfg.Volume, cfg.MusicDir)
	if err != nil {
		win.Close()
		return nil, err
	}

	a := &App{
		cfg:          cfg,
		window:       win,
		nav:          nav,
		svc:          svc,
		progress:     progress.New(len(nav.Displays())),
		music:        player,
		showProgress: true,
	}
	nav.SetStatusCallback(func() {
		a.progress.Update(nav.Displays(), nav.LoadedCount())
		if nav.LoadedCount() >= countEnabled(nav.Displays()) {
			a.showProgress = false
		}
	})
	return a, nil
}

func countEnabled(disps []engine.Display) int {
	n := 0
	for _, d := range disps {
		if d.Enabled() {
			n++
		}
	}
	return n
}

func (a *App) LoadWeather() error {
	lat, lon, _, err := a.svc.ResolveLocation(a.cfg.Location, a.cfg.Latitude, a.cfg.Longitude)
	if err != nil {
		return err
	}
	params, err := a.svc.BuildParams(lat, lon, a.cfg.Units)
	if err != nil {
		return err
	}
	a.params = params
	a.nav.FetchAll(params)
	return nil
}

func (a *App) Run(screenshotPath string) error {
	if err := a.LoadWeather(); err != nil {
		return err
	}

	for {
		if err := a.renderFrame(); err != nil {
			return err
		}
		if screenshotPath != "" {
			if err := a.window.SaveScreenshot(screenshotPath); err != nil {
				return err
			}
			return nil
		}
		event, quit := a.window.PollEvent()
		if quit {
			return nil
		}
		a.handleEvent(event)
		sdl.Delay(16)
	}
}

func (a *App) renderFrame() error {
	canvas := a.window.Canvas()
	canvas.Clear()
	canvas.DrawScanlines(a.cfg.Scanlines)

	if a.showProgress {
		a.progress.Draw(canvas)
	} else {
		d := a.nav.CurrentDisplay()
		if d == nil {
			a.progress.Draw(canvas)
		} else {
			_ = d.Draw(canvas, d.ScreenIndex())
			if d.ShowClock() {
				a.drawClock(canvas)
			}
			if d.ShowTicker() {
				a.drawTicker(canvas)
			}
		}
	}
	return a.window.Present()
}

func (a *App) drawClock(canvas engine.Canvas) {
	if a.params == nil {
		return
	}
	loc, err := time.LoadLocation(a.params.TimeZone)
	if err != nil {
		loc = time.Local
	}
	now := time.Now().In(loc)
	canvas.DrawTextRight("small", now.Format("03:04:05 PM"), 620, 8, nil, true)
	canvas.DrawTextRight("small", now.Format("Mon Jan 02"), 620, 22, nil, true)
}

func (a *App) drawTicker(canvas engine.Canvas) {
	canvas.DrawRect(64, 430, 512, 28, nil)
	text := a.tickerText()
	if text == "" {
		text = "WEATHERSTAR 4000+"
	}
	a.tickerX--
	if a.tickerX < -len(text)*8 {
		a.tickerX = 640
	}
	canvas.DrawText("regular", text, a.tickerX, 438, nil, true)
}

func (a *App) tickerText() string {
	for _, d := range a.nav.Displays() {
		if cw, ok := d.(interface{ TickerText() string }); ok {
			if t := cw.TickerText(); t != "" {
				return t
			}
		}
	}
	if a.cfg.CustomScroll != "" {
		parts := splitScroll(a.cfg.CustomScroll)
		if len(parts) > 0 {
			return parts[time.Now().Unix()%int64(len(parts))]
		}
	}
	return ""
}

func splitScroll(s string) []string {
	var out []string
	for _, p := range splitPipe(s) {
		p = trimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitPipe(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '|' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func (a *App) handleEvent(event sdl.Event) {
	if event == nil {
		return
	}
	key, ok := event.(*sdl.KeyboardEvent)
	if !ok || key.Type != sdl.KEYDOWN {
		return
	}
	switch key.Keysym.Sym {
	case sdl.K_SPACE:
		a.playing = !a.playing
		a.nav.SetPlaying(a.playing)
		if a.playing {
			_ = a.music.Play()
		} else {
			a.music.Stop()
		}
	case sdl.K_RIGHT:
		a.nav.NavNextManual()
	case sdl.K_LEFT:
		a.nav.NavPrevManual()
	case sdl.K_f:
		a.window.ToggleFullscreen()
	}
}

func (a *App) Close() {
	if a.music != nil {
		a.music.Close()
	}
	if a.window != nil {
		a.window.Close()
	}
}

func (a *App) String() string {
	return fmt.Sprintf("ws4000 app playing=%v", a.playing)
}
