package app

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
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

	playing     bool
	autoStarted bool

	tickerIndex    int
	tickerLastFlip time.Time
}

func New(cfg config.Config, headless bool) (*App, error) {
	scale := cfg.Scale
	if scale <= 0 {
		scale = 2
	}
	win, err := render.NewWindow(render.WindowConfig{
		Title:      "WeatherStar 4000+",
		Fullscreen: cfg.Fullscreen,
		Scale:      scale,
		Headless:   headless,
	})
	if err != nil {
		return nil, err
	}

	cacheDir := filepath.Join(os.TempDir(), "ws4000-cache")
	svc := weather.NewService(cacheDir, cfg.FixtureDir)
	nav := engine.NewNavigator(cfg.Speed)
	displays.RegisterAll(nav, svc, cfg)

	var player *music.Player
	if !headless {
		player, err = music.NewPlayer(cfg.Volume, cfg.MusicDir)
		if err != nil {
			// audio is non-fatal (e.g. no audio device on a headless Pi)
			player = nil
		}
	}

	return &App{
		cfg:      cfg,
		window:   win,
		nav:      nav,
		svc:      svc,
		progress: progress.New(),
		music:    player,
	}, nil
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

// allEnabledSettled reports whether every enabled display finished loading (or failed).
func (a *App) allEnabledSettled() bool {
	for _, d := range a.nav.Displays() {
		if d.Enabled() && d.Status() == engine.StatusLoading {
			return false
		}
	}
	return true
}

func (a *App) anyLoaded() bool {
	for _, d := range a.nav.Displays() {
		if d.Enabled() && d.Status() == engine.StatusLoaded {
			return true
		}
	}
	return false
}

// Run is the main loop. If screenshotPath is set, waits for data, renders the
// requested display (screenshotDisplay or first playable) once and exits.
func (a *App) Run(screenshotPath, screenshotDisplay string) error {
	if err := a.LoadWeather(); err != nil {
		return err
	}

	if screenshotPath != "" {
		return a.runScreenshot(screenshotPath, screenshotDisplay)
	}

	frame := time.NewTicker(33 * time.Millisecond) // ~30fps
	defer frame.Stop()

	for {
		// auto-start playback once everything settles (kiosk behavior)
		if !a.autoStarted && a.allEnabledSettled() && a.anyLoaded() {
			a.autoStarted = true
			a.playing = true
			a.nav.SetPlaying(true)
			if a.music != nil {
				_ = a.music.Play()
			}
		}

		a.renderFrame()
		if err := a.window.Present(); err != nil {
			return err
		}

		// drain events
		for {
			event, quit := a.window.PollEvent()
			if quit {
				return nil
			}
			if event == nil {
				break
			}
			a.handleEvent(event)
		}
		<-frame.C
	}
}

func (a *App) runScreenshot(path, displayID string) error {
	// wait up to 60s for displays to settle
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if a.allEnabledSettled() {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	for _, d := range a.nav.Displays() {
		fmt.Fprintf(os.Stderr, "%-22s %s\n", d.ID(), d.Status())
	}
	a.playing = true
	a.nav.SetPlaying(true)

	// navigate to the requested display
	if displayID != "" {
		if !a.nav.ShowDisplayByID(displayID) {
			fmt.Fprintf(os.Stderr, "display %q not found\n", displayID)
		}
	}
	// let the display settle on its first screen
	time.Sleep(100 * time.Millisecond)
	a.renderFrame()
	_ = a.window.Present()
	return a.window.SaveScreenshot(path)
}

func (a *App) renderFrame() {
	canvas := a.window.Canvas()
	canvas.Clear()

	current := a.nav.CurrentDisplay()
	if current == nil {
		_ = a.progress.Draw(canvas, a.nav.Displays(), a.params)
	} else {
		_ = current.Draw(canvas, current.ScreenIndex())
		if current.OkToDrawTicker() {
			a.drawTicker(canvas)
		}
	}

	if a.cfg.Scanlines {
		canvas.ApplyScanlines()
	}
}

// drawTicker renders the bottom scroll area (currentweatherscroll.mjs):
// segments cycle every 4 seconds.
func (a *App) drawTicker(canvas *render.Canvas) {
	segments := a.tickerSegments()
	if len(segments) == 0 {
		return
	}
	if time.Since(a.tickerLastFlip) > 4*time.Second {
		a.tickerIndex++
		a.tickerLastFlip = time.Now()
	}
	seg := segments[a.tickerIndex%len(segments)]

	style := render.Style{Family: render.FontStar4000, Size: 32, Color: color.RGBA{R: 255, G: 255, B: 255, A: 255}, Shadow: true}
	canvas.Text(style, seg, 55, 419)
}

func (a *App) tickerSegments() []string {
	var segments []string
	for _, d := range a.nav.Displays() {
		if hz, ok := d.(*displays.HazardsDisplay); ok {
			for _, t := range hz.HazardTexts() {
				if len(t) > 80 {
					t = t[:80]
				}
				segments = append(segments, strings.ToUpper(t))
				break // one hazard segment like upstream screen 0
			}
		}
		if cw, ok := d.(*displays.CurrentWeatherDisplay); ok {
			segments = append(segments, cw.TickerSegments()...)
		}
	}
	if a.cfg.CustomScroll != "" {
		for _, part := range strings.Split(a.cfg.CustomScroll, "|") {
			part = strings.TrimSpace(part)
			if part != "" {
				segments = append(segments, part)
			}
		}
	}
	return segments
}

func (a *App) handleEvent(event sdl.Event) {
	key, ok := event.(*sdl.KeyboardEvent)
	if !ok || key.Type != sdl.KEYDOWN {
		return
	}
	switch key.Keysym.Sym {
	case sdl.K_SPACE:
		a.playing = !a.playing
		a.nav.SetPlaying(a.playing)
		if a.music != nil {
			if a.playing {
				_ = a.music.Play()
			} else {
				a.music.Stop()
			}
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
	a.nav.SetPlaying(false)
	if a.music != nil {
		a.music.Close()
	}
	if a.window != nil {
		a.window.Close()
	}
}

func (a *App) String() string {
	return fmt.Sprintf("ws4000 playing=%v", a.playing)
}
