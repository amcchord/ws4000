package engine

import (
	"sync"
	"time"

	"github.com/amcchord/ws4000/internal/render"
)

type Display interface {
	ID() string
	Name() string
	NavID() int
	Enabled() bool
	SetEnabled(bool)
	Status() LoadStatus
	Timing() *Timing
	Fetch(params *WeatherParams) error
	Draw(canvas *render.Canvas, screenIndex int) error
	// ShowClock reports whether the standard date/time should be drawn (drawn by displays themselves).
	OkToDrawTicker() bool
	OnShow()
	OnHide()
	StartNav(speed float64)
	StopNav()
	NavNext(cmd NavCommand)
	NavPrev(cmd NavCommand)
	ScreenIndex() int
	SetNavCallback(func(id string, resp NavResponse))
}

type WeatherParams struct {
	Latitude        float64
	Longitude       float64
	City            string
	State           string
	ZoneID          string
	RadarID         string
	StationID       string
	StationName     string
	WeatherOffice   string
	GridX           int
	GridY           int
	TimeZone        string
	ForecastURL     string
	ForecastGridURL string
	ObservationURL  string
	Units           string
	StationURLs     []string
	Location        *time.Location
}

func (p *WeatherParams) TZ() *time.Location {
	if p == nil || p.Location == nil {
		return time.Local
	}
	return p.Location
}

type BaseDisplay struct {
	mu             sync.Mutex
	id             string
	name           string
	navID          int
	enabled        bool
	status         LoadStatus
	hasData        bool // has successfully loaded at least once
	timing         Timing
	screenIndex    int
	navBaseCount   int
	navTicker      *time.Ticker
	navStop        chan struct{}
	okToDrawTicker bool
	params         *WeatherParams
	onNav          func(id string, resp NavResponse)
}

func NewBaseDisplay(navID int, id, name string, defaultEnabled bool) *BaseDisplay {
	d := &BaseDisplay{
		id:             id,
		name:           name,
		navID:          navID,
		enabled:        defaultEnabled,
		status:         StatusLoading,
		okToDrawTicker: true,
		screenIndex:    -1,
		timing: Timing{
			TotalScreens: 1,
			BaseDelayMS:  9000,
			Delay:        1,
		},
	}
	d.timing.CalcNavTiming()
	if !defaultEnabled {
		d.status = StatusDisabled
	}
	return d
}

func (d *BaseDisplay) ID() string             { return d.id }
func (d *BaseDisplay) Name() string           { return d.name }
func (d *BaseDisplay) NavID() int             { return d.navID }
func (d *BaseDisplay) Enabled() bool          { return d.enabled }
func (d *BaseDisplay) Timing() *Timing        { return &d.timing }
func (d *BaseDisplay) OkToDrawTicker() bool   { return d.okToDrawTicker }
func (d *BaseDisplay) ScreenIndex() int       { return d.screenIndex }
func (d *BaseDisplay) Params() *WeatherParams { return d.params }

func (d *BaseDisplay) SetOkToDrawTicker(v bool) { d.okToDrawTicker = v }

func (d *BaseDisplay) Status() LoadStatus {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.status
}

func (d *BaseDisplay) SetEnabled(v bool) {
	d.mu.Lock()
	d.enabled = v
	if v {
		if d.status == StatusDisabled {
			d.status = StatusLoading
		}
	} else {
		d.status = StatusDisabled
	}
	d.mu.Unlock()
	if !v {
		d.StopNav()
	}
}

func (d *BaseDisplay) SetStatus(s LoadStatus) {
	d.mu.Lock()
	defer d.mu.Unlock()
	// silent-refresh semantics (like upstream): once a display has data,
	// a failed re-fetch keeps the old data on screen instead of dropping
	// the display out of the rotation
	if s == StatusFailed && d.hasData {
		return
	}
	if s == StatusLoaded {
		d.hasData = true
	}
	d.status = s
}

func (d *BaseDisplay) SetNavCallback(fn func(id string, resp NavResponse)) {
	d.onNav = fn
}

// BeginFetch stores params and resets status; returns false if display disabled.
// Once a display has loaded data, subsequent fetches are silent refreshes: the
// status stays Loaded so the rotation is not interrupted.
func (d *BaseDisplay) BeginFetch(params *WeatherParams) bool {
	d.params = params
	if !d.enabled {
		d.mu.Lock()
		d.status = StatusDisabled
		d.mu.Unlock()
		return false
	}
	d.mu.Lock()
	if !d.hasData {
		d.status = StatusLoading
	}
	d.mu.Unlock()
	d.timing.CalcNavTiming()
	return true
}

func (d *BaseDisplay) OnShow() {
	if d.screenIndex < 0 {
		d.screenIndex = 0
	}
}

func (d *BaseDisplay) OnHide() {
	d.StopNav()
	d.navBaseCount = 0
	d.screenIndex = -1
}

func (d *BaseDisplay) StartNav(speed float64) {
	d.StopNav()
	if speed <= 0 {
		speed = 1.0
	}
	interval := time.Duration(float64(d.timing.BaseDelayMS)*speed) * time.Millisecond
	stop := make(chan struct{})
	ticker := time.NewTicker(interval)
	d.mu.Lock()
	d.navStop = stop
	d.navTicker = ticker
	d.mu.Unlock()
	go func() {
		for {
			select {
			case <-ticker.C:
				d.navBaseTime()
			case <-stop:
				return
			}
		}
	}()
}

func (d *BaseDisplay) StopNav() {
	d.mu.Lock()
	ticker := d.navTicker
	stop := d.navStop
	d.navTicker = nil
	d.navStop = nil
	d.mu.Unlock()
	if ticker != nil {
		ticker.Stop()
	}
	if stop != nil {
		close(stop)
	}
}

func (d *BaseDisplay) navBaseTime() {
	if !d.enabled {
		return
	}
	d.navBaseCount++
	d.updateScreenFromBaseCount(false)
}

func (d *BaseDisplay) updateScreenFromBaseCount(force bool) {
	next, ok := d.timing.ScreenIndexFromBaseCount(d.navBaseCount)
	if !ok {
		if d.onNav != nil {
			d.onNav(d.id, NavRespNext)
		}
		return
	}
	if !force && next == d.screenIndex && d.screenIndex >= 0 {
		return
	}
	d.screenIndex = next
}

func (d *BaseDisplay) NavNext(cmd NavCommand) {
	if cmd == NavFirstFrame {
		d.navBaseCount = 0
	} else {
		d.navBaseCount = d.timing.NavNextBaseCount(d.navBaseCount)
	}
	d.updateScreenFromBaseCount(true)
}

func (d *BaseDisplay) NavPrev(cmd NavCommand) {
	if cmd == NavLastFrame {
		d.navBaseCount = d.timing.LastFrameBaseCount()
	} else {
		prev := d.timing.NavPrevBaseCount(d.navBaseCount)
		if prev == 0 && d.navBaseCount == 0 {
			if d.onNav != nil {
				d.onNav(d.id, NavRespPrevious)
			}
			return
		}
		d.navBaseCount = prev
	}
	d.updateScreenFromBaseCount(true)
}
