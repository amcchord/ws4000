package displays

import (
	"time"

	"github.com/amcchord/ws4000/internal/config"
	"github.com/amcchord/ws4000/internal/data/suncalc"
	"github.com/amcchord/ws4000/internal/data/weather"
	"github.com/amcchord/ws4000/internal/engine"
	"github.com/amcchord/ws4000/internal/render"
)

// Upstream almanac: background 3.png, sun grid (2 days sunrise/sunset)
// and the next 4 moon phases.
type AlmanacDisplay struct {
	*engine.BaseDisplay
	svc      *weather.Service
	cfg      config.Config
	dayNames [2]string
	sunrise  [2]string
	sunset   [2]string
	moon     []suncalc.PhaseEvent
}

func NewAlmanac(svc *weather.Service, cfg config.Config) *AlmanacDisplay {
	return &AlmanacDisplay{
		BaseDisplay: engine.NewBaseDisplay(9, "almanac", "Almanac", false),
		svc:         svc,
		cfg:         cfg,
	}
}

func (d *AlmanacDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params) {
		return nil
	}
	loc := params.TZ()
	now := time.Now().In(loc)
	for i := 0; i < 2; i++ {
		day := now.Add(time.Duration(i) * 24 * time.Hour)
		times := suncalc.GetTimes(params.Latitude, params.Longitude, day, loc)
		d.dayNames[i] = day.Format("Monday")
		if times.Valid {
			d.sunrise[i] = times.Sunrise.Format("3:04 PM")
			d.sunset[i] = times.Sunset.Format("3:04 PM")
		} else {
			d.sunrise[i] = "-"
			d.sunset[i] = "-"
		}
	}
	d.moon = suncalc.NextMoonPhases(now, 4)
	d.Timing().TotalScreens = 1
	d.Timing().CalcNavTiming()
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *AlmanacDisplay) Draw(c *render.Canvas, screenIndex int) error {
	_ = c.DrawBackground("backgrounds/3.png")
	drawHeaderSingle(c, d.Params(), "Almanac", true)

	styleHead := render.Style{Family: render.FontStar4000, Size: 32, Color: colTitle, Shadow: true}

	// sun grid: centered 3 columns (label, day0, day1) with 90px gaps
	col0Right := 170 // right edge of row labels
	col1Center := 320
	col2Center := 530
	y := mainTop + 3

	c.TextCenter(styleHead, d.dayNames[0], col1Center, y)
	c.TextCenter(styleHead, d.dayNames[1], col2Center, y)
	y += 30
	c.TextRight(styleBody, "Sunrise:", col0Right, y)
	c.TextCenter(styleBody, d.sunrise[0], col1Center, y)
	c.TextCenter(styleBody, d.sunrise[1], col2Center, y)
	y += 30
	c.TextRight(styleBody, "Sunset:", col0Right, y)
	c.TextCenter(styleBody, d.sunset[0], col1Center, y)
	c.TextCenter(styleBody, d.sunset[1], col2Center, y)

	// moon data
	y += 30 + 12
	c.Text(styleHead, "Moon Data:", 63, y)
	y += 36

	cellW := 132
	startX := (640 - cellW*4) / 2
	for i, phase := range d.moon {
		if i >= 4 {
			break
		}
		x := startX + i*cellW
		c.TextCenterIn(styleBody, phase.Name, x, cellW, y)
		_ = c.ImageCenteredIn(phase.Icon, x+5, y+32, cellW, 0)
		c.TextCenterIn(styleBody, phase.Date.Format("Jan 2"), x, cellW, y+122)
	}
	return nil
}
