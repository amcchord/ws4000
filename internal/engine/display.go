package engine

import (
	"sync"
	"time"
)

type Display interface {
	ID() string
	Name() string
	NavID() int
	Enabled() bool
	SetEnabled(bool)
	Status() LoadStatus
	SetStatus(LoadStatus)
	Timing() *Timing
	Fetch(params *WeatherParams) error
	Draw(canvas Canvas, screenIndex int) error
	DrawClock(canvas Canvas) error
	ShowClock() bool
	ShowTicker() bool
	BackgroundID() string
	OnShow(screenIndex int)
	OnHide()
	TickBaseCount(baseCount int) (next bool, response NavResponse)
	ResetNav()
	StartNav(speed float64)
	StopNav()
	NavNext(cmd NavCommand)
	NavPrev(cmd NavCommand)
	ScreenIndex() int
	SetScreenIndex(int)
	BaseCount() int
	SetBaseCount(int)
	SetNavCallback(func(id string, resp NavResponse))
}

type Canvas interface {
	Clear()
	DrawBackground(path string) error
	DrawText(font, text string, x, y int, color interface{}, shadow bool)
	DrawTextRight(font, text string, x, y int, color interface{}, shadow bool)
	DrawTextCentered(font, text string, x, y, w int, color interface{}, shadow bool)
	DrawImage(path string, x, y, w, h int) error
	DrawGIF(path string, x, y, w, h int, frame int) error
	DrawRect(x, y, w, h int, color interface{})
	DrawScanlines(enabled bool)
	Size() (int, int)
	Texture() interface{}
}

type WeatherParams struct {
	Latitude          float64
	Longitude         float64
	City              string
	State             string
	ZoneID            string
	RadarID           string
	StationID         string
	WeatherOffice     string
	TimeZone          string
	ForecastURL       string
	ForecastGridURL   string
	ObservationURL    string
	Units             string
	StationURLs       []string
}

type BaseDisplay struct {
	mu              sync.Mutex
	id              string
	name            string
	navID           int
	enabled         bool
	defaultEnabled  bool
	status          LoadStatus
	timing          Timing
	screenIndex     int
	navBaseCount    int
	speed           float64
	navTicker       *time.Ticker
	navStop         chan struct{}
	showClock       bool
	showTicker      bool
	okDrawTicker    bool
	params          *WeatherParams
	onNav           func(id string, resp NavResponse)
}

func NewBaseDisplay(navID int, id, name string, defaultEnabled bool) *BaseDisplay {
	d := &BaseDisplay{
		id:             id,
		name:           name,
		navID:          navID,
		defaultEnabled: defaultEnabled,
		enabled:        defaultEnabled,
		status:         StatusLoading,
		showClock:      true,
		showTicker:     true,
		okDrawTicker:   true,
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
func (d *BaseDisplay) Status() LoadStatus     { return d.status }
func (d *BaseDisplay) Timing() *Timing        { return &d.timing }
func (d *BaseDisplay) ShowClock() bool        { return d.showClock }
func (d *BaseDisplay) ShowTicker() bool       { return d.showTicker && d.okDrawTicker }
func (d *BaseDisplay) BackgroundID() string   { return d.id }
func (d *BaseDisplay) ScreenIndex() int       { return d.screenIndex }
func (d *BaseDisplay) BaseCount() int         { return d.navBaseCount }
func (d *BaseDisplay) Params() *WeatherParams { return d.params }

func (d *BaseDisplay) SetEnabled(v bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = v
	if v {
		if d.status == StatusDisabled {
			d.status = StatusLoading
		}
	} else {
		d.status = StatusDisabled
		d.StopNav()
	}
}

func (d *BaseDisplay) SetStatus(s LoadStatus) {
	d.mu.Lock()
	d.status = s
	d.mu.Unlock()
}

func (d *BaseDisplay) SetScreenIndex(v int) { d.screenIndex = v }
func (d *BaseDisplay) SetBaseCount(v int)   { d.navBaseCount = v }

func (d *BaseDisplay) SetNavCallback(fn func(id string, resp NavResponse)) {
	d.onNav = fn
}

func (d *BaseDisplay) SetShowClock(v bool)  { d.showClock = v }
func (d *BaseDisplay) SetShowTicker(v bool) { d.showTicker = v; d.okDrawTicker = v }

func (d *BaseDisplay) BeginFetch(params *WeatherParams, refresh bool) bool {
	d.params = params
	if !d.enabled {
		d.SetStatus(StatusDisabled)
		return false
	}
	if !refresh {
		d.SetStatus(StatusLoading)
	}
	d.timing.CalcNavTiming()
	return true
}

func (d *BaseDisplay) DrawClock(canvas Canvas) error {
	return nil
}

func (d *BaseDisplay) OnShow(screenIndex int) {
	if d.screenIndex < 0 {
		d.screenIndex = 0
	}
}

func (d *BaseDisplay) OnHide() {
	d.ResetNav()
}

func (d *BaseDisplay) ResetNav() {
	d.StopNav()
	d.navBaseCount = 0
	d.screenIndex = -1
}

func (d *BaseDisplay) StartNav(speed float64) {
	d.StopNav()
	if speed <= 0 {
		speed = 1.0
	}
	d.speed = speed
	interval := time.Duration(float64(d.timing.BaseDelayMS) / speed) * time.Millisecond
	d.navStop = make(chan struct{})
	d.navTicker = time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-d.navTicker.C:
				d.navBaseTime()
			case <-d.navStop:
				return
			}
		}
	}()
}

func (d *BaseDisplay) StopNav() {
	if d.navTicker != nil {
		d.navTicker.Stop()
		d.navTicker = nil
	}
	if d.navStop != nil {
		close(d.navStop)
		d.navStop = nil
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
	if d.screenIndex < 0 {
		d.screenIndex = 0
	} else {
		d.screenIndex = next
	}
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

func (d *BaseDisplay) TickBaseCount(baseCount int) (bool, NavResponse) {
	return false, NavRespNext
}

func (d *BaseDisplay) DrawStandardHeader(canvas Canvas, titleTop, titleBottom string) {
	canvas.DrawText("large", titleTop, 20, 18, ColorTitle(), true)
	canvas.DrawText("large", titleBottom, 20, 38, ColorTitle(), true)
}

func ColorTitle() interface{} {
	return struct{}{}
}

func (d *BaseDisplay) DrawStandardDateTime(canvas Canvas, date, timeStr string) {
	canvas.DrawTextRight("small", timeStr, 620, 8, nil, true)
	canvas.DrawTextRight("small", date, 620, 22, nil, true)
}
