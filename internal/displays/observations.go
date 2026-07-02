package displays

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amcchord/ws4000/internal/assets"
	"github.com/amcchord/ws4000/internal/config"
	"github.com/amcchord/ws4000/internal/data/icons"
	"github.com/amcchord/ws4000/internal/data/units"
	"github.com/amcchord/ws4000/internal/data/weather"
	"github.com/amcchord/ws4000/internal/engine"
	"github.com/amcchord/ws4000/internal/render"
)

// stationInfo matches stations.json entries.
type stationInfo struct {
	ID       string  `json:"id"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	State    string  `json:"state"`
	City     string  `json:"city"`
	Priority int     `json:"priority"`
}

func loadStations() (map[string]stationInfo, error) {
	data, err := assets.Read("data/stations.json")
	if err != nil {
		return nil, err
	}
	var out map[string]stationInfo
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func distanceSq(aLat, aLon, bLat, bLon float64) float64 {
	dx := aLon - bLon
	dy := aLat - bLat
	return dx*dx + dy*dy
}

// --- Latest Observations ---

// Upstream _latest-observations.scss: blue box content; columns relative to box:
// temp at +230, weather at +280, wind at +430; rows 40px tall, text top +8.
type LatestObservationsDisplay struct {
	*engine.BaseDisplay
	svc  *weather.Service
	cfg  config.Config
	rows []obsRow
}

type obsRow struct {
	City    string
	Temp    string
	Weather string
	Wind    string
}

func NewLatestObservations(svc *weather.Service, cfg config.Config) *LatestObservationsDisplay {
	return &LatestObservationsDisplay{
		BaseDisplay: engine.NewBaseDisplay(2, "latest-observations", "Latest Observations", true),
		svc:         svc,
		cfg:         cfg,
	}
}

func (d *LatestObservationsDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params) {
		return nil
	}
	stations, err := loadStations()
	if err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}

	// sort stations by distance from current location, like upstream
	var sorted []stationInfo
	for _, s := range stations {
		sorted = append(sorted, s)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return distanceSq(sorted[i].Lat, sorted[i].Lon, params.Latitude, params.Longitude) <
			distanceSq(sorted[j].Lat, sorted[j].Lon, params.Latitude, params.Longitude)
	})

	conv := units.New(params.Units)

	// fetch the nearest 24 candidate stations in parallel, then keep the
	// 7 closest with valid data (one per city)
	candidates := sorted
	if len(candidates) > 24 {
		candidates = candidates[:24]
	}
	type obsResult struct {
		row obsRow
		ok  bool
	}
	results := make([]obsResult, len(candidates))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, st := range candidates {
		if st.ID == params.StationID {
			continue
		}
		wg.Add(1)
		go func(i int, st stationInfo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			obs, err := d.svc.Client.GetObservations("https://api.weather.gov/stations/"+st.ID, 1)
			if err != nil || len(obs.Features) == 0 {
				return
			}
			p := obs.Features[0].Properties
			if p.Temperature.Value == nil || p.TextDescription == "" {
				return
			}
			wind := "Calm"
			spd := conv.WindKMH(p.WindSpeed.Value)
			if spd != "Calm" && spd != "-" {
				wind = fmt.Sprintf("%-3s%3s", units.DirectionToNSEW(p.WindDirection.Value), spd)
			}
			cond := p.TextDescription
			if len(cond) > 9 {
				cond = weather.ShortCondition(cond)
			}
			if len(cond) > 9 {
				cond = cond[:9]
			}
			city := st.City
			if len(city) > 14 {
				city = city[:14]
			}
			results[i] = obsResult{
				row: obsRow{
					City:    city,
					Temp:    conv.TempC(p.Temperature.Value),
					Weather: cond,
					Wind:    wind,
				},
				ok: true,
			}
		}(i, st)
	}
	wg.Wait()

	var rows []obsRow
	seen := map[string]bool{}
	for _, r := range results {
		if !r.ok || seen[r.row.City] {
			continue
		}
		seen[r.row.City] = true
		rows = append(rows, r.row)
		if len(rows) >= 7 {
			break
		}
	}
	d.rows = rows

	if len(d.rows) == 0 {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	d.Timing().TotalScreens = 1
	d.Timing().CalcNavTiming()
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *LatestObservationsDisplay) Draw(c *render.Canvas, screenIndex int) error {
	_ = c.DrawBackground("backgrounds/1.png")
	drawHeaderDual(c, d.Params(), "Latest", "Observations", true)

	unitLabel := degree() + "F"
	if d.Params() != nil && units.New(d.Params().Units).Units == "metric" {
		unitLabel = degree() + "C"
	}
	headStyle := render.Style{Family: render.FontStar4000Small, Size: 32, Color: colWhite, Shadow: true}
	c.Text(headStyle, unitLabel, blueBoxMargin+230, mainTop-14)
	c.Text(headStyle, "Weather", blueBoxMargin+280, mainTop-14)
	c.Text(headStyle, "Wind", blueBoxMargin+430, mainTop-14)

	y := mainTop + 10 + 8
	for _, row := range d.rows {
		c.Text(styleBody, row.City, blueBoxMargin, y)
		c.TextRight(styleBody, row.Temp, blueBoxMargin+230+45, y)
		c.Text(styleBody, row.Weather, blueBoxMargin+280, y)
		c.Text(styleBody, row.Wind, blueBoxMargin+430, y)
		y += 40
	}
	return nil
}

// --- Hourly ---

// Upstream _hourly.scss: full-width; purple sticky header bar 20px; columns:
// hour 25, icon 255 (70 wide centered), temp 355, like 425, wind right in 505..605;
// rows 72px Star4000 Large yellow.
type HourlyDisplay struct {
	*engine.BaseDisplay
	svc  *weather.Service
	cfg  config.Config
	rows []hourlyRow
}

type hourlyRow struct {
	Hour string
	Icon string
	Temp string
	Like string
	Wind string
}

func NewHourly(svc *weather.Service, cfg config.Config) *HourlyDisplay {
	return &HourlyDisplay{
		BaseDisplay: engine.NewBaseDisplay(3, "hourly", "Hourly Forecast", false),
		svc:         svc,
		cfg:         cfg,
	}
}

func (d *HourlyDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params) {
		return nil
	}
	hourly, err := d.svc.Client.GetHourly(params.ForecastURL, params.Units)
	if err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	periods := hourly.Properties.Periods
	if len(periods) > 24 {
		periods = periods[:24]
	}
	loc := params.TZ()
	conv := units.New(params.Units)
	var rows []hourlyRow
	for _, p := range periods {
		t, err := time.Parse(time.RFC3339, p.StartTime)
		if err != nil {
			continue
		}
		// wind: "SSE 10" from direction + first number of "10 mph"
		windSpeed := 0
		if fields := strings.Fields(p.WindSpeed); len(fields) > 0 {
			windSpeed, _ = strconv.Atoi(fields[0])
		}
		wind := "Calm"
		if windSpeed > 0 {
			wind = fmt.Sprintf("%s %d", p.WindDirection, windSpeed)
		}

		humidity := 50.0
		if p.RelativeHumidity.Value != nil {
			humidity = *p.RelativeHumidity.Value
		}
		like := apparentTemperature(p.Temperature, humidity, windSpeed, conv.Units == "metric")

		rows = append(rows, hourlyRow{
			Hour: strings.ToUpper(t.In(loc).Format("3 PM")),
			Icon: icons.SmallIcon(p.Icon, !p.IsDaytime),
			Temp: fmtTemp(p.Temperature),
			Like: fmtTemp(like),
			Wind: wind,
		})
	}
	if len(rows) == 0 {
		d.SetStatus(engine.StatusNoData)
		return nil
	}
	d.rows = rows
	screens := (len(d.rows) + 3) / 4
	d.Timing().TotalScreens = screens
	d.Timing().Delay = 1
	d.Timing().CalcNavTiming()
	d.SetStatus(engine.StatusLoaded)
	return nil
}

// apparentTemperature computes the "LIKE" column: NWS heat index when hot,
// wind chill when cold, otherwise the air temperature. temp is in display
// units; formulas run in Fahrenheit.
func apparentTemperature(temp int, humidity float64, windMPH int, metric bool) int {
	tempF := float64(temp)
	if metric {
		tempF = tempF*9/5 + 32
	}
	feelsF := tempF
	switch {
	case tempF >= 80 && humidity >= 40:
		// Rothfusz heat index regression
		t, rh := tempF, humidity
		feelsF = -42.379 + 2.04901523*t + 10.14333127*rh -
			0.22475541*t*rh - 0.00683783*t*t - 0.05481717*rh*rh +
			0.00122874*t*t*rh + 0.00085282*t*rh*rh - 0.00000199*t*t*rh*rh
	case tempF <= 50 && windMPH > 3:
		v := math.Pow(float64(windMPH), 0.16)
		feelsF = 35.74 + 0.6215*tempF - 35.75*v + 0.4275*tempF*v
	}
	if metric {
		return int(math.Round((feelsF - 32) * 5 / 9))
	}
	return int(math.Round(feelsF))
}

func (d *HourlyDisplay) Draw(c *render.Canvas, screenIndex int) error {
	_ = c.DrawBackground("backgrounds/1.png")
	drawHeaderDual(c, d.Params(), "Hourly", "Forecast", false)

	// purple header bar
	c.FillRect(0, mainTop, 640, 20, colColumnHead)
	c.Text(styleColHead, "TEMP", 355, mainTop-14)
	c.Text(styleColHead, "LIKE", 435, mainTop-14)
	c.Text(styleColHead, "WIND", 535, mainTop-14)

	if screenIndex < 0 {
		screenIndex = 0
	}
	rowStyle := render.Style{Family: render.FontStar4000Large, Size: 32, Color: colTitle, Shadow: true}
	start := screenIndex * 4
	for i := 0; i < 4; i++ {
		idx := start + i
		if idx >= len(d.rows) {
			break
		}
		row := d.rows[idx]
		y := mainTop + 20 + 10 + i*72 + 8
		c.Text(rowStyle, row.Hour, 25, y)
		if row.Icon != "" {
			_ = c.ImageCenteredIn(row.Icon, 255, y-8, 70, 60)
		}
		c.Text(rowStyle, row.Temp, 355, y)
		c.Text(rowStyle, row.Like, 425, y)
		c.TextRight(rowStyle, row.Wind, 605, y)
	}
	return nil
}

// --- Hourly Graph ---

// Upstream hourly-graph.mjs: chart 532x285 at x=50 (within main at y=90);
// x-axis labels at chart x + k*133; lines: temperature red, dewpoint green,
// cloud lightgrey, rain aqua, each with black under-stroke.
type HourlyGraphDisplay struct {
	*engine.BaseDisplay
	svc       *weather.Service
	cfg       config.Config
	temps     []float64
	dewpoints []float64
	clouds    []float64
	rains     []float64
	times     []time.Time
}

func NewHourlyGraph(svc *weather.Service, cfg config.Config) *HourlyGraphDisplay {
	d := &HourlyGraphDisplay{
		BaseDisplay: engine.NewBaseDisplay(4, "hourly-graph", "Hourly Graph", true),
		svc:         svc,
		cfg:         cfg,
	}
	return d
}

func (d *HourlyGraphDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params) {
		return nil
	}
	hourly, err := d.svc.Client.GetHourly(params.ForecastURL, params.Units)
	if err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	periods := hourly.Properties.Periods
	if len(periods) > 36 {
		periods = periods[:36]
	}
	if len(periods) < 2 {
		d.SetStatus(engine.StatusNoData)
		return nil
	}
	loc := params.TZ()
	conv := units.New(params.Units)
	var temps, dewpoints, clouds, rains []float64
	var times []time.Time
	for _, p := range periods {
		temps = append(temps, float64(p.Temperature))
		if p.Dewpoint.Value != nil {
			// hourly dewpoint is degC regardless of the units query parameter
			dewpoints = append(dewpoints, conv.CToF(*p.Dewpoint.Value))
		} else {
			dewpoints = append(dewpoints, math.NaN())
		}
		prob := 0.0
		if p.ProbabilityOfPrecipitation.Value != nil {
			prob = *p.ProbabilityOfPrecipitation.Value
		}
		rains = append(rains, prob)
		clouds = append(clouds, math.NaN())
		if t, err := time.Parse(time.RFC3339, p.StartTime); err == nil {
			times = append(times, t.In(loc))
		} else {
			times = append(times, time.Time{})
		}
	}

	// cloud cover comes from the gridpoint data (skyCover series)
	if grid, err := d.svc.Client.GetGridData(params.ForecastGridURL); err == nil {
		cloudAt := func(t time.Time) float64 {
			best := math.NaN()
			for _, v := range grid.Properties.SkyCover.Values {
				// validTime like "2026-06-12T18:00:00+00:00/PT1H"
				parts := strings.SplitN(v.ValidTime, "/", 2)
				vt, err := time.Parse(time.RFC3339, parts[0])
				if err != nil || v.Value == nil {
					continue
				}
				if !vt.After(t) {
					best = *v.Value
				}
			}
			return best
		}
		for i, t := range times {
			if !t.IsZero() {
				clouds[i] = cloudAt(t)
			}
		}
	}
	d.temps, d.dewpoints, d.clouds, d.rains, d.times = temps, dewpoints, clouds, rains, times
	d.Timing().TotalScreens = 1
	d.Timing().Delay = 4
	d.Timing().CalcNavTiming()
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *HourlyGraphDisplay) Draw(c *render.Canvas, screenIndex int) error {
	_ = c.DrawBackground("backgrounds/1-chart.png")
	drawHeaderSingle(c, d.Params(), "Hourly Graph", false)

	// legend (upstream right-aligned at right:60 in header, tight stack)
	tempUnit := degree() + "F"
	if d.Params() != nil && units.New(d.Params().Units).Units == "metric" {
		tempUnit = degree() + "C"
	}
	legend := render.Style{Family: render.FontStar4000Small, Size: 28, Shadow: true}
	legendItems := []struct {
		label string
		col   color.RGBA
	}{
		{"Temperature " + tempUnit, color.RGBA{R: 255, A: 255}},
		{"Dewpoint " + tempUnit, color.RGBA{R: 0, G: 128, B: 0, A: 255}},
		{"Cloud %", color.RGBA{R: 211, G: 211, B: 211, A: 255}},
		{"Precip %", color.RGBA{R: 0, G: 255, B: 255, A: 255}},
	}
	ly := 17
	for _, item := range legendItems {
		s := legend
		s.Color = item.col
		c.TextRight(s, item.label, 580, ly)
		ly += 14
	}

	if len(d.temps) < 2 {
		return nil
	}

	const chartX, chartW, chartH = 50, 532, 285
	chartY := mainTop

	minT, maxT := d.temps[0], d.temps[0]
	for _, t := range d.temps {
		minT = math.Min(minT, t)
		maxT = math.Max(maxT, t)
	}
	for _, dp := range d.dewpoints {
		if !math.IsNaN(dp) {
			minT = math.Min(minT, dp)
			maxT = math.Max(maxT, dp)
		}
	}
	// scale spans the data range directly (matches upstream chart scaling)
	minS := math.Floor(minT)
	maxS := math.Ceil(maxT)
	if maxS-minS < 4 {
		maxS = minS + 4
	}

	tempY := func(v float64) int {
		return chartY + 10 + int((maxS-v)/(maxS-minS)*float64(chartH-20))
	}
	pctY := func(v float64) int {
		return chartY + 10 + int((100-v)/100*float64(chartH-20))
	}
	xAt := func(i int) int {
		return chartX + 5 + i*(chartW-10)/(len(d.temps)-1)
	}

	// y-axis labels (yellow with degree sign, Star4000 Small)
	for k := 0; k < 4; k++ {
		v := math.Round(maxS - (maxS-minS)*float64(k)/3)
		c.TextRight(styleColHead, fmt.Sprintf("%.0f%s", v, degree()), chartX-4, tempY(v)-10)
	}
	// x-axis labels every ~9 hours: hour as "7P", with day abbreviation after midnight
	lastDay := -1
	if !d.times[0].IsZero() {
		lastDay = d.times[0].Day()
	}
	for i := 0; i < len(d.times); i += 9 {
		if d.times[i].IsZero() {
			continue
		}
		t := d.times[i]
		hour := t.Format("3")
		ap := "A"
		if t.Format("PM") == "PM" {
			ap = "P"
		}
		label := hour + ap
		if t.Day() != lastDay {
			label = strings.ToUpper(t.Format("Mon")) + " " + label
			lastDay = t.Day()
		}
		c.TextCenter(styleColHead, label, xAt(i), chartY+chartH+2)
	}

	drawLine := func(vals []float64, yOf func(float64) int, col color.RGBA) {
		// black under-stroke then colored line (upstream chart style)
		for pass := 0; pass < 2; pass++ {
			for i := 1; i < len(vals); i++ {
				if math.IsNaN(vals[i-1]) || math.IsNaN(vals[i]) {
					continue
				}
				x0, y0 := xAt(i-1), yOf(vals[i-1])
				x1, y1 := xAt(i), yOf(vals[i])
				if pass == 0 {
					drawThickLine(c, x0, y0+2, x1, y1+2, 5, color.RGBA{A: 255})
				} else {
					drawThickLine(c, x0, y0, x1, y1, 3, col)
				}
			}
		}
	}
	drawLine(d.clouds, pctY, color.RGBA{R: 211, G: 211, B: 211, A: 255})
	drawLine(d.rains, pctY, color.RGBA{R: 0, G: 255, B: 255, A: 255})
	drawLine(d.dewpoints, tempY, color.RGBA{R: 0, G: 128, B: 0, A: 255})
	drawLine(d.temps, tempY, color.RGBA{R: 255, A: 255})
	return nil
}

// drawThickLine draws a simple thick line segment on the canvas.
func drawThickLine(c *render.Canvas, x0, y0, x1, y1, thickness int, col color.RGBA) {
	dx := math.Abs(float64(x1 - x0))
	dy := math.Abs(float64(y1 - y0))
	steps := int(math.Max(dx, dy))
	if steps == 0 {
		steps = 1
	}
	half := thickness / 2
	for s := 0; s <= steps; s++ {
		t := float64(s) / float64(steps)
		x := int(float64(x0) + t*float64(x1-x0))
		y := int(float64(y0) + t*float64(y1-y0))
		c.FillRect(x-half, y-half, thickness, thickness, col)
	}
}
