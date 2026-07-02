package displays

import (
	"strings"
	"time"

	"github.com/amcchord/ws4000/internal/config"
	"github.com/amcchord/ws4000/internal/data/icons"
	"github.com/amcchord/ws4000/internal/data/nws"
	"github.com/amcchord/ws4000/internal/data/weather"
	"github.com/amcchord/ws4000/internal/engine"
	"github.com/amcchord/ws4000/internal/render"
)

// --- Local Forecast ---

// Upstream: blue box text pages, Star4000 32px, line-height 40, 7 lines per page
// (280px page height), container at main+15 with 10px side margins.
type LocalForecastDisplay struct {
	*engine.BaseDisplay
	svc   *weather.Service
	cfg   config.Config
	pages [][]string // wrapped lines per screen
}

func NewLocalForecast(svc *weather.Service, cfg config.Config) *LocalForecastDisplay {
	d := &LocalForecastDisplay{
		BaseDisplay: engine.NewBaseDisplay(7, "local-forecast", "Local Forecast", true),
		svc:         svc,
		cfg:         cfg,
	}
	d.Timing().BaseDelayMS = 5000
	return d
}

func (d *LocalForecastDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params) {
		return nil
	}
	forecast, err := d.svc.Client.GetForecast(params.ForecastURL, params.Units)
	if err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	periods := filterExpired(forecast.Properties.Periods)
	if len(periods) > 6 {
		periods = periods[:6]
	}
	if len(periods) == 0 {
		d.SetStatus(engine.StatusNoData)
		return nil
	}

	// Each period becomes "NAME...TEXT", wrapped to the box width, split into
	// pages of up to 7 lines (280px / 40px line height).
	const maxLines = 7
	var pages [][]string
	var delays []int
	for _, p := range periods {
		text := p.Name + "..." + strings.ReplaceAll(p.DetailedForecast, "...", " ")
		lines := wrapTextWidth(text, 492) // blue box 512 - 2*10 margins
		for start := 0; start < len(lines); start += maxLines {
			end := start + maxLines
			if end > len(lines) {
				end = len(lines)
			}
			page := lines[start:end]
			pages = append(pages, page)
			// upstream content-aware timing: 1 line 0.6x, 2 lines 0.8x, 6+ 1.4x, else 1x of 5s
			var mult float64
			switch {
			case len(page) == 1:
				mult = 0.6
			case len(page) == 2:
				mult = 0.8
			case len(page) >= 6:
				mult = 1.4
			default:
				mult = 1.0
			}
			delays = append(delays, int(mult*5000/250))
		}
	}
	d.pages = pages
	d.Timing().BaseDelayMS = 250
	d.Timing().Delay = delays
	d.Timing().CalcNavTiming()
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *LocalForecastDisplay) Draw(c *render.Canvas, screenIndex int) error {
	_ = c.DrawBackground("backgrounds/1.png")
	drawHeaderDual(c, d.Params(), "Local", "Forecast", true)
	if screenIndex < 0 || screenIndex >= len(d.pages) {
		return nil
	}
	y := mainTop + 15
	for _, line := range d.pages[screenIndex] {
		c.Text(styleBody, line, blueBoxMargin+10, y)
		y += 40
	}
	return nil
}

// wrapTextWidth wraps text into lines no wider than maxPx, measured with the
// actual Star4000 32px font metrics.
func wrapTextWidth(text string, maxPx int) []string {
	words := strings.Fields(text)
	var lines []string
	var cur string
	for _, w := range words {
		candidate := w
		if cur != "" {
			candidate = cur + " " + w
		}
		if cur != "" && render.MeasureString(render.FontStar4000, 32, candidate) > maxPx {
			lines = append(lines, cur)
			cur = w
		} else {
			cur = candidate
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

func filterExpired(periods []nws.ForecastPeriod) []nws.ForecastPeriod {
	now := time.Now()
	var out []nws.ForecastPeriod
	for _, p := range periods {
		if end, err := time.Parse(time.RFC3339, p.EndTime); err == nil && end.Before(now) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// --- Extended Forecast ---

// Upstream: background 2.png, full-width day cards. Day i content box at
// x = 27 + i*195 + 20, width 155. Date yellow centered; icon centered (max-h 75);
// condition centered; Lo/Hi blocks with Star4000 Large values.
type ExtendedForecastDisplay struct {
	*engine.BaseDisplay
	svc  *weather.Service
	cfg  config.Config
	days []extendedDay
}

type extendedDay struct {
	Name      string
	Icon      string
	Condition string
	Low       string
	High      string
	HasLow    bool
}

func NewExtendedForecast(svc *weather.Service, cfg config.Config) *ExtendedForecastDisplay {
	d := &ExtendedForecastDisplay{
		BaseDisplay: engine.NewBaseDisplay(8, "extended-forecast", "Extended Forecast", true),
		svc:         svc,
		cfg:         cfg,
	}
	d.Timing().TotalScreens = 2
	d.Timing().Delay = 2
	d.Timing().CalcNavTiming()
	return d
}

func (d *ExtendedForecastDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params) {
		return nil
	}
	forecast, err := d.svc.Client.GetForecast(params.ForecastURL, params.Units)
	if err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	periods := filterExpired(forecast.Properties.Periods)

	// pair day/night periods into days (upstream extendedforecast.mjs parse)
	var days []extendedDay
	i := 0
	// skip a leading night period so days start with daytime
	if len(periods) > 0 && !periods[0].IsDaytime {
		i = 1
	}
	for ; i+1 < len(periods) && len(days) < 6; i += 2 {
		day := periods[i]
		night := periods[i+1]
		t, _ := time.Parse(time.RFC3339, day.StartTime)
		cond := shortenCondition(day.ShortForecast)
		days = append(days, extendedDay{
			Name:      strings.ToUpper(t.Format("Mon")),
			Icon:      icons.LargeIcon(day.Icon),
			Condition: cond,
			High:      fmtTemp(day.Temperature),
			Low:       fmtTemp(night.Temperature),
			HasLow:    true,
		})
	}
	if len(days) == 0 {
		d.SetStatus(engine.StatusNoData)
		return nil
	}
	d.days = days
	screens := (len(d.days) + 2) / 3
	d.Timing().TotalScreens = screens
	d.Timing().Delay = screens
	d.Timing().CalcNavTiming()
	d.SetStatus(engine.StatusLoaded)
	return nil
}

// shortenCondition ports upstream's shortenExtendedForecastText (first word combos).
func shortenCondition(condition string) string {
	c := condition
	c = strings.ReplaceAll(c, "Slight ", "")
	c = strings.ReplaceAll(c, "Chance ", "")
	c = strings.ReplaceAll(c, "Very ", "")
	c = strings.ReplaceAll(c, "Patchy ", "")
	c = strings.ReplaceAll(c, "Areas ", "")
	c = strings.ReplaceAll(c, "Isolated ", "")
	c = strings.ReplaceAll(c, "Scattered ", "")
	if idx := strings.Index(c, " then "); idx >= 0 {
		c = c[:idx]
	}
	words := strings.SplitN(c, " ", 3)
	if len(words) > 2 {
		c = words[0] + " " + words[1]
	}
	if len(c) > 10 {
		// upstream truncates each condition word to 10 chars
		parts := strings.Split(c, " ")
		for j := range parts {
			if len(parts[j]) > 10 {
				parts[j] = parts[j][:10]
			}
		}
		c = strings.Join(parts, " ")
	}
	return c
}

func (d *ExtendedForecastDisplay) Draw(c *render.Canvas, screenIndex int) error {
	_ = c.DrawBackground("backgrounds/2.png")
	drawHeaderDual(c, d.Params(), "Extended", "Forecast", false)
	if screenIndex < 0 {
		screenIndex = 0
	}
	start := screenIndex * 3

	styleDate := render.Style{Family: render.FontStar4000, Size: 32, Color: colTitle, Shadow: true}
	styleCond := render.Style{Family: render.FontStar4000, Size: 32, Color: colWhite, Shadow: true}
	styleLabelLo := render.Style{Family: render.FontStar4000, Size: 32, Color: colExtendedLow, Shadow: true}
	styleLabelHi := render.Style{Family: render.FontStar4000, Size: 32, Color: colTitle, Shadow: true}
	styleValue := render.Style{Family: render.FontStar4000Large, Size: 32, Color: colWhite, Shadow: true}

	for i := 0; i < 3; i++ {
		idx := start + i
		if idx >= len(d.days) {
			break
		}
		day := d.days[idx]
		// day-container margin-left 27 + day margin 15 + padding 5
		x := 27 + i*195 + 20
		w := 155
		y := mainTop + 16 + 5

		c.TextCenterIn(styleDate, day.Name, x, w, y)
		y += 38
		_ = c.ImageCenteredIn(day.Icon, x, y, w, 75)
		y += 75 + 5
		// condition: up to 2 lines centered in 74px block
		condLines := wrapTextWidth(day.Condition, w)
		for li, line := range condLines {
			if li >= 2 {
				break
			}
			c.TextCenterIn(styleCond, line, x, w, y)
			y += 36
		}
		y = mainTop + 16 + 5 + 38 + 80 + 74 + 5
		// temperature blocks: two 44% blocks
		blockW := w * 44 / 100
		loX := x + (w/2 - blockW)
		hiX := x + w/2
		c.TextCenterIn(styleLabelLo, "Lo", loX, blockW, y)
		c.TextCenterIn(styleLabelHi, "Hi", hiX, blockW, y)
		c.TextCenterIn(styleValue, day.Low, loX, blockW, y+36)
		c.TextCenterIn(styleValue, day.High, hiX, blockW, y+36)
	}
	return nil
}

// --- Hazards ---

// Upstream: background 7.png, no header, full-screen dark red box with
// uppercase scrolling text (margin 80px each side).
type HazardsDisplay struct {
	*engine.BaseDisplay
	svc   *weather.Service
	cfg   config.Config
	lines []string
	texts []string // raw hazard texts for the ticker
}

func NewHazards(svc *weather.Service, cfg config.Config) *HazardsDisplay {
	d := &HazardsDisplay{
		BaseDisplay: engine.NewBaseDisplay(0, "hazards", "Hazards", true),
		svc:         svc,
		cfg:         cfg,
	}
	d.SetOkToDrawTicker(false)
	d.Timing().TotalScreens = 0
	d.Timing().CalcNavTiming()
	return d
}

func (d *HazardsDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params) {
		return nil
	}
	alerts, err := d.svc.Client.GetAlerts(params.ZoneID)
	if err != nil {
		// keep whatever state we had; TotalScreens stays 0 until a load succeeds
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	var lines, texts []string
	for _, f := range alerts.Features {
		// only significant alerts get the full-screen treatment upstream
		if f.Properties.Severity != "Extreme" && f.Properties.Severity != "Severe" {
			continue
		}
		text := f.Properties.Event + " " + f.Properties.Description
		texts = append(texts, text)
		lines = append(lines, wrapTextWidth(strings.ToUpper(text), 480)...)
	}
	d.lines = lines
	d.texts = texts
	if len(lines) == 0 {
		d.Timing().TotalScreens = 0
		d.SetStatus(engine.StatusNoData)
		return nil
	}
	// scroll through pages of 9 lines
	const perPage = 9
	pages := (len(lines) + perPage - 1) / perPage
	d.Timing().TotalScreens = pages
	d.Timing().Delay = 2
	d.Timing().CalcNavTiming()
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *HazardsDisplay) Draw(c *render.Canvas, screenIndex int) error {
	_ = c.DrawBackground("backgrounds/7.png")
	// full-height red box, no header
	c.FillRect(0, 0, 640, 480, colHazardRed)
	if screenIndex < 0 {
		screenIndex = 0
	}
	styleHazard := render.Style{Family: render.FontStar4000, Size: 32, Color: colWhite, Shadow: false}
	const perPage = 9
	start := screenIndex * perPage
	y := 20
	for i := start; i < start+perPage && i < len(d.lines); i++ {
		c.Text(styleHazard, d.lines[i], 80, y)
		y += 44
	}
	return nil
}

// HazardTexts exposes hazard strings for the ticker.
func (d *HazardsDisplay) HazardTexts() []string { return d.texts }
