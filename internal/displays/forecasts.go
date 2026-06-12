package displays

import (
	"strconv"
	"strings"

	"github.com/austinmcchord/ws4000/internal/config"
	"github.com/austinmcchord/ws4000/internal/data/nws"
	"github.com/austinmcchord/ws4000/internal/data/weather"
	"github.com/austinmcchord/ws4000/internal/engine"
)

type HazardsDisplay struct {
	*engine.BaseDisplay
	svc  *weather.Service
	cfg  config.Config
	text []string
}

func NewHazards(svc *weather.Service, cfg config.Config) *HazardsDisplay {
	d := &HazardsDisplay{
		BaseDisplay: engine.NewBaseDisplay(0, "hazards", "Hazards", true),
		svc:         svc,
		cfg:         cfg,
	}
	d.Timing().TotalScreens = 0
	return d
}

func (d *HazardsDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params, false) {
		return nil
	}
	alerts, err := d.svc.Client.GetAlerts(params.ZoneID)
	if err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	d.text = nil
	for _, f := range alerts.Features {
		line := strings.ToUpper(f.Properties.Event)
		if f.Properties.Headline != "" {
			line = strings.ToUpper(f.Properties.Headline)
		}
		d.text = append(d.text, line)
	}
	if len(d.text) == 0 {
		d.Timing().TotalScreens = 0
		d.SetStatus(engine.StatusNoData)
		return nil
	}
	d.Timing().TotalScreens = len(d.text)
	delays := make([]int, len(d.text))
	for i := range delays {
		delays[i] = 3
	}
	d.Timing().Delay = delays
	d.Timing().CalcNavTiming()
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *HazardsDisplay) Draw(canvas engine.Canvas, screenIndex int) error {
	drawBackground(canvas, "latest-observations")
	drawHeader(canvas, "Weather", "Warnings")
	drawDateTime(canvas, loadLocation(d.Params().TimeZone))
	if screenIndex < 0 || screenIndex >= len(d.text) {
		return nil
	}
	canvas.DrawText("regular", d.text[screenIndex], 72, 120, nil, true)
	return nil
}

type LocalForecastDisplay struct {
	*engine.BaseDisplay
	svc    *weather.Service
	cfg    config.Config
	periods []nws.ForecastPeriod
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
	if !d.BeginFetch(params, false) {
		return nil
	}
	forecast, err := d.svc.Client.GetForecast(params.ForecastURL, params.Units)
	if err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	limit := 6
	if len(forecast.Properties.Periods) < limit {
		limit = len(forecast.Properties.Periods)
	}
	d.periods = forecast.Properties.Periods[:limit]
	d.Timing().TotalScreens = len(d.periods)
	delays := make([]int, len(d.periods))
	for i := range delays {
		delays[i] = 1
	}
	d.Timing().Delay = delays
	d.Timing().CalcNavTiming()
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *LocalForecastDisplay) Draw(canvas engine.Canvas, screenIndex int) error {
	drawBackground(canvas, d.ID())
	drawHeader(canvas, "Local", "Forecast")
	drawDateTime(canvas, loadLocation(d.Params().TimeZone))
	if screenIndex < 0 || screenIndex >= len(d.periods) {
		return nil
	}
	p := d.periods[screenIndex]
	text := strings.ToUpper(p.Name + "... " + strings.ReplaceAll(p.DetailedForecast, "...", " "))
	canvas.DrawText("regular", text, 72, 120, nil, true)
	return nil
}

type ExtendedForecastDisplay struct {
	*engine.BaseDisplay
	svc     *weather.Service
	cfg     config.Config
	periods []nws.ForecastPeriod
}

func NewExtendedForecast(svc *weather.Service, cfg config.Config) *ExtendedForecastDisplay {
	return &ExtendedForecastDisplay{
		BaseDisplay: engine.NewBaseDisplay(8, "extended-forecast", "Extended Forecast", true),
		svc:         svc,
		cfg:         cfg,
	}
}

func (d *ExtendedForecastDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params, false) {
		return nil
	}
	forecast, err := d.svc.Client.GetForecast(params.ForecastURL, params.Units)
	if err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	d.periods = forecast.Properties.Periods
	if len(d.periods) > 10 {
		d.periods = d.periods[:10]
	}
	d.Timing().TotalScreens = 1
	d.Timing().Delay = 3
	d.Timing().CalcNavTiming()
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *ExtendedForecastDisplay) Draw(canvas engine.Canvas, screenIndex int) error {
	drawBackground(canvas, d.ID())
	drawHeader(canvas, "Extended", "Forecast")
	drawDateTime(canvas, loadLocation(d.Params().TimeZone))
	y := 100
	for i, p := range d.periods {
		if i >= 5 {
			break
		}
		hiLo := "LO"
		if p.IsDaytime {
			hiLo = "HI"
		}
		line := strings.ToUpper(p.Name)
		canvas.DrawText("regular", line, 80, y, nil, true)
		canvas.DrawText("regular", hiLo, 300, y, nil, true)
		temp := p.Temperature
		if paramsUnits(d) == "metric" && p.TemperatureUnit == "F" {
			temp = p.Temperature
		}
		canvas.DrawText("regular", formatTemp(temp), 360, y, nil, true)
		canvas.DrawText("regular", strings.ToUpper(p.ShortForecast), 420, y, nil, true)
		y += 28
	}
	return nil
}

func paramsUnits(d *ExtendedForecastDisplay) string {
	if d.Params() == nil {
		return "us"
	}
	return d.Params().Units
}

func formatTemp(v int) string {
	return strconv.Itoa(v)
}
