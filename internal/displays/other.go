package displays

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/austinmcchord/ws4000/internal/assets"
	"github.com/austinmcchord/ws4000/internal/config"
	"github.com/austinmcchord/ws4000/internal/data/nws"
	"github.com/austinmcchord/ws4000/internal/data/suncalc"
	"github.com/austinmcchord/ws4000/internal/data/units"
	"github.com/austinmcchord/ws4000/internal/data/weather"
	"github.com/austinmcchord/ws4000/internal/engine"
)

type LatestObservationsDisplay struct {
	*engine.BaseDisplay
	svc  *weather.Service
	cfg  config.Config
	rows []string
}

func NewLatestObservations(svc *weather.Service, cfg config.Config) *LatestObservationsDisplay {
	return &LatestObservationsDisplay{
		BaseDisplay: engine.NewBaseDisplay(2, "latest-observations", "Latest Observations", true),
		svc:         svc,
		cfg:         cfg,
	}
}

func (d *LatestObservationsDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params, false) {
		return nil
	}
	conv := units.New(params.Units)
	d.rows = nil
	for i, url := range params.StationURLs {
		if i >= 6 {
			break
		}
		obs, err := d.svc.Client.GetObservations(url, 1)
		if err != nil || len(obs.Features) == 0 {
			continue
		}
		p := obs.Features[0].Properties
		line := fmt.Sprintf("%s  %s%s  %s",
			params.StationID,
			conv.TempC(p.Temperature.Value),
			conv.TempSymbol(),
			strings.ToUpper(p.TextDescription),
		)
		d.rows = append(d.rows, line)
		break
	}
	if len(d.rows) == 0 {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	d.Timing().TotalScreens = 1
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *LatestObservationsDisplay) Draw(canvas engine.Canvas, screenIndex int) error {
	drawBackground(canvas, d.ID())
	drawHeader(canvas, "Latest", "Observations")
	drawDateTime(canvas, loadLocation(d.Params().TimeZone))
	y := 120
	for _, row := range d.rows {
		canvas.DrawText("regular", row, 72, y, nil, true)
		y += 24
	}
	return nil
}

type HourlyDisplay struct {
	*engine.BaseDisplay
	svc      *weather.Service
	cfg      config.Config
	periods  []nws.HourlyPeriod
}

func NewHourly(svc *weather.Service, cfg config.Config) *HourlyDisplay {
	return &HourlyDisplay{
		BaseDisplay: engine.NewBaseDisplay(3, "hourly", "Hourly Forecast", false),
		svc:         svc,
		cfg:         cfg,
	}
}

func (d *HourlyDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params, false) {
		return nil
	}
	hourly, err := d.svc.Client.GetHourly(params.ForecastURL, params.Units)
	if err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	limit := 24
	if len(hourly.Properties.Periods) < limit {
		limit = len(hourly.Properties.Periods)
	}
	d.periods = hourly.Properties.Periods[:limit]
	screens := (limit + 5) / 6
	d.Timing().TotalScreens = screens
	d.Timing().Delay = screens
	d.Timing().CalcNavTiming()
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *HourlyDisplay) Draw(canvas engine.Canvas, screenIndex int) error {
	drawBackground(canvas, d.ID())
	drawHeader(canvas, "Hourly", "Forecast")
	drawDateTime(canvas, loadLocation(d.Params().TimeZone))
	start := screenIndex * 6
	y := 110
	canvas.DrawText("regular", "TEMP", 120, 90, nil, true)
	canvas.DrawText("regular", "LIKE", 220, 90, nil, true)
	canvas.DrawText("regular", "WIND", 320, 90, nil, true)
	for i := start; i < start+6 && i < len(d.periods); i++ {
		p := d.periods[i]
		t, _ := time.Parse(time.RFC3339, p.StartTime)
		line := fmt.Sprintf("%s   %d   %s", strings.ToUpper(t.Format("3 PM")), p.Temperature, p.WindSpeed)
		canvas.DrawText("regular", line, 72, y, nil, true)
		y += 28
	}
	return nil
}

type HourlyGraphDisplay struct {
	*engine.BaseDisplay
	svc     *weather.Service
	cfg     config.Config
	periods []nws.HourlyPeriod
}

func NewHourlyGraph(svc *weather.Service, cfg config.Config) *HourlyGraphDisplay {
	return &HourlyGraphDisplay{
		BaseDisplay: engine.NewBaseDisplay(4, "hourly-graph", "Hourly Graph", true),
		svc:         svc,
		cfg:         cfg,
	}
}

func (d *HourlyGraphDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params, false) {
		return nil
	}
	hourly, err := d.svc.Client.GetHourly(params.ForecastURL, params.Units)
	if err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	limit := 36
	if len(hourly.Properties.Periods) < limit {
		limit = len(hourly.Properties.Periods)
	}
	d.periods = hourly.Properties.Periods[:limit]
	d.Timing().TotalScreens = 1
	d.Timing().Delay = 4
	d.Timing().CalcNavTiming()
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *HourlyGraphDisplay) Draw(canvas engine.Canvas, screenIndex int) error {
	drawBackground(canvas, d.ID())
	drawHeader(canvas, "Hourly", "Graph")
	drawDateTime(canvas, loadLocation(d.Params().TimeZone))
	canvas.DrawText("regular", "Temperature", 80, 90, nil, true)
	maxTemp := 0
	minTemp := 999
	for _, p := range d.periods {
		if p.Temperature > maxTemp {
			maxTemp = p.Temperature
		}
		if p.Temperature < minTemp {
			minTemp = p.Temperature
		}
	}
	span := maxTemp - minTemp
	if span <= 0 {
		span = 1
	}
	x := 80
	for i, p := range d.periods {
		if i >= 18 {
			break
		}
		h := 120 * (p.Temperature - minTemp) / span
		if h < 4 {
			h = 4
		}
		canvas.DrawRect(x, 300-int(h), 8, int(h), nil)
		x += 28
	}
	return nil
}

type travelCity struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

type TravelForecastDisplay struct {
	*engine.BaseDisplay
	svc     *weather.Service
	cfg     config.Config
	entries []string
}

func NewTravelForecast(svc *weather.Service, cfg config.Config) *TravelForecastDisplay {
	return &TravelForecastDisplay{
		BaseDisplay: engine.NewBaseDisplay(5, "travel", "Travel Forecast", false),
		svc:         svc,
		cfg:         cfg,
	}
}

func (d *TravelForecastDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params, false) {
		return nil
	}
	data, err := assets.Read("data/travelcities.json")
	if err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	var cities []travelCity
	if err := json.Unmarshal(data, &cities); err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	d.entries = nil
	for i, city := range cities {
		if i >= 8 {
			break
		}
		point, err := d.svc.Client.GetPoint(city.Lat, city.Lon)
		if err != nil {
			continue
		}
		forecast, err := d.svc.Client.GetForecast(point.Properties.Forecast, params.Units)
		if err != nil || len(forecast.Properties.Periods) == 0 {
			continue
		}
		p := forecast.Properties.Periods[0]
		d.entries = append(d.entries, fmt.Sprintf("%s  %d  %s", strings.ToUpper(city.Name), p.Temperature, strings.ToUpper(p.ShortForecast)))
	}
	if len(d.entries) == 0 {
		d.SetStatus(engine.StatusNoData)
		return nil
	}
	d.Timing().TotalScreens = 1
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *TravelForecastDisplay) Draw(canvas engine.Canvas, screenIndex int) error {
	drawBackground(canvas, d.ID())
	drawHeader(canvas, "Travel", "Forecast")
	drawDateTime(canvas, loadLocation(d.Params().TimeZone))
	y := 110
	for _, e := range d.entries {
		canvas.DrawText("regular", e, 72, y, nil, true)
		y += 28
	}
	return nil
}

type RegionalForecastDisplay struct {
	*engine.BaseDisplay
	svc     *weather.Service
	cfg     config.Config
	entries []string
}

func NewRegionalForecast(svc *weather.Service, cfg config.Config) *RegionalForecastDisplay {
	return &RegionalForecastDisplay{
		BaseDisplay: engine.NewBaseDisplay(6, "regional-forecast", "Regional Forecast", true),
		svc:         svc,
		cfg:         cfg,
	}
}

func (d *RegionalForecastDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params, false) {
		return nil
	}
	data, err := assets.Read("data/regionalcities.json")
	if err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	var raw map[string][]travelCity
	if err := json.Unmarshal(data, &raw); err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	cities, ok := raw[params.State]
	if !ok || len(cities) == 0 {
		d.SetStatus(engine.StatusNoData)
		return nil
	}
	d.entries = nil
	for i, city := range cities {
		if i >= 8 {
			break
		}
		point, err := d.svc.Client.GetPoint(city.Lat, city.Lon)
		if err != nil {
			continue
		}
		forecast, err := d.svc.Client.GetForecast(point.Properties.Forecast, params.Units)
		if err != nil || len(forecast.Properties.Periods) == 0 {
			continue
		}
		p := forecast.Properties.Periods[0]
		d.entries = append(d.entries, fmt.Sprintf("%s  %d", strings.ToUpper(city.Name), p.Temperature))
	}
	if len(d.entries) == 0 {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	d.Timing().TotalScreens = 1
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *RegionalForecastDisplay) Draw(canvas engine.Canvas, screenIndex int) error {
	drawBackground(canvas, d.ID())
	drawHeader(canvas, "Regional", "Observations")
	drawDateTime(canvas, loadLocation(d.Params().TimeZone))
	y := 110
	for _, e := range d.entries {
		canvas.DrawText("regular", e, 72, y, nil, true)
		y += 28
	}
	return nil
}

type AlmanacDisplay struct {
	*engine.BaseDisplay
	svc  *weather.Service
	cfg  config.Config
	sun  suncalc.Times
	moon suncalc.Moon
}

func NewAlmanac(svc *weather.Service, cfg config.Config) *AlmanacDisplay {
	return &AlmanacDisplay{
		BaseDisplay: engine.NewBaseDisplay(9, "almanac", "Almanac", false),
		svc:         svc,
		cfg:         cfg,
	}
}

func (d *AlmanacDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params, false) {
		return nil
	}
	loc := loadLocation(params.TimeZone)
	now := time.Now().In(loc)
	d.sun = suncalc.GetTimes(params.Latitude, params.Longitude, now, loc)
	d.moon = suncalc.GetMoon(now)
	d.Timing().TotalScreens = 1
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *AlmanacDisplay) Draw(canvas engine.Canvas, screenIndex int) error {
	drawBackground(canvas, d.ID())
	drawHeader(canvas, "Almanac", "Data")
	drawDateTime(canvas, loadLocation(d.Params().TimeZone))
	canvas.DrawText("regular", "Sunrise:", 80, 120, nil, true)
	canvas.DrawText("regular", strings.ToUpper(d.sun.Sunrise.Format("3:04 PM")), 220, 120, nil, true)
	canvas.DrawText("regular", "Sunset:", 80, 150, nil, true)
	canvas.DrawText("regular", strings.ToUpper(d.sun.Sunset.Format("3:04 PM")), 220, 150, nil, true)
	canvas.DrawText("regular", "Moon Data:", 80, 190, nil, true)
	canvas.DrawText("regular", strings.ToUpper(d.moon.PhaseName), 220, 190, nil, true)
	_ = canvas.DrawImage(d.moon.Icon, 400, 180, 0, 0)
	return nil
}

type SPCOutlookDisplay struct {
	*engine.BaseDisplay
	svc  *weather.Service
	cfg  config.Config
	text string
}

func NewSPCOutlook(svc *weather.Service, cfg config.Config) *SPCOutlookDisplay {
	return &SPCOutlookDisplay{
		BaseDisplay: engine.NewBaseDisplay(10, "spc-outlook", "SPC Outlook", true),
		svc:         svc,
		cfg:         cfg,
	}
}

func (d *SPCOutlookDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params, false) {
		return nil
	}
	url := fmt.Sprintf("https://mapservices.weather.noaa.gov/vector/rest/services/outlook/SPC_wx_outlk/MapServer/1/query?geometry=%f,%f&geometryType=esriGeometryPoint&inSR=4326&spatialRel=esriSpatialRelIntersects&outFields=*&f=json", params.Longitude, params.Latitude)
	resp, err := http.Get(url)
	if err != nil {
		d.SetStatus(engine.StatusNoData)
		return nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "features") || strings.Contains(string(body), `"features":[]`) {
		d.SetStatus(engine.StatusNoData)
		d.Timing().TotalScreens = 0
		return nil
	}
	d.text = "SEVERE WEATHER OUTLOOK IN EFFECT FOR YOUR AREA"
	d.Timing().TotalScreens = 1
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *SPCOutlookDisplay) Draw(canvas engine.Canvas, screenIndex int) error {
	drawBackground(canvas, d.ID())
	drawHeader(canvas, "Storm Prediction", "Center Outlook")
	drawDateTime(canvas, loadLocation(d.Params().TimeZone))
	canvas.DrawText("regular", d.text, 72, 140, nil, true)
	canvas.DrawText("regular", "Categorical Outlook", 72, 200, nil, true)
	return nil
}

type RadarDisplay struct {
	*engine.BaseDisplay
	svc    *weather.Service
	cfg    config.Config
	frames []string
	times  []string
}

func NewRadar(svc *weather.Service, cfg config.Config) *RadarDisplay {
	d := &RadarDisplay{
		BaseDisplay: engine.NewBaseDisplay(11, "radar", "Local Radar", true),
		svc:         svc,
		cfg:         cfg,
	}
	d.SetShowClock(false)
	d.SetShowTicker(false)
	d.Timing().BaseDelayMS = 350
	d.Timing().Delay = []engine.ScreenDelay{
		{Time: 4, ScreenIndex: 5}, {Time: 1, ScreenIndex: 0}, {Time: 1, ScreenIndex: 1},
		{Time: 1, ScreenIndex: 2}, {Time: 1, ScreenIndex: 3}, {Time: 1, ScreenIndex: 4},
		{Time: 4, ScreenIndex: 5}, {Time: 1, ScreenIndex: 0}, {Time: 1, ScreenIndex: 1},
		{Time: 1, ScreenIndex: 2}, {Time: 1, ScreenIndex: 3}, {Time: 1, ScreenIndex: 4},
		{Time: 12, ScreenIndex: 5},
	}
	d.Timing().CalcNavTiming()
	return d
}

func (d *RadarDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params, false) {
		return nil
	}
	if params.State == "AK" || params.State == "HI" {
		d.SetStatus(engine.StatusNoData)
		d.Timing().TotalScreens = 0
		return nil
	}
	d.frames = []string{"maps/radar-stretched.webp", "maps/radar-stretched.webp"}
	d.times = []string{"", ""}
	for i := 0; i < 6; i++ {
		d.frames = append(d.frames, "maps/radar-stretched.webp")
		d.times = append(d.times, time.Now().Add(time.Duration(-i*5)*time.Minute).Format("3:04 PM"))
	}
	d.Timing().TotalScreens = len(d.frames)
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *RadarDisplay) Draw(canvas engine.Canvas, screenIndex int) error {
	drawBackground(canvas, d.ID())
	canvas.DrawText("large", "Local", 20, 18, nil, true)
	canvas.DrawText("large", "Radar", 20, 42, nil, true)
	if screenIndex >= 0 && screenIndex < len(d.frames) {
		_ = canvas.DrawImage(d.frames[screenIndex], 64, 80, 512, 280)
	}
	if screenIndex >= 0 && screenIndex < len(d.times) && d.times[screenIndex] != "" {
		canvas.DrawTextRight("regular", d.times[screenIndex], 620, 18, nil, true)
	}
	return nil
}

var pngTimestamp = regexp.MustCompile(`_(\d{12})\.png`)

func parseRadarTimestamp(name string) string {
	m := pngTimestamp.FindStringSubmatch(name)
	if len(m) < 2 {
		return ""
	}
	t, err := time.Parse("200601021504", m[1])
	if err != nil {
		return ""
	}
	return t.Format("3:04 PM")
}