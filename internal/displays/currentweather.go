package displays

import (
	"fmt"
	"strings"
	"time"

	"github.com/amcchord/ws4000/internal/config"
	"github.com/amcchord/ws4000/internal/data/icons"
	"github.com/amcchord/ws4000/internal/data/nws"
	"github.com/amcchord/ws4000/internal/data/units"
	"github.com/amcchord/ws4000/internal/data/weather"
	"github.com/amcchord/ws4000/internal/engine"
	"github.com/amcchord/ws4000/internal/render"
)

type CurrentWeatherDisplay struct {
	*engine.BaseDisplay
	svc  *weather.Service
	cfg  config.Config
	data *currentWeatherData
}

type currentWeatherData struct {
	Temperature    string // with degree sign
	TempUnit       string // F or C
	Condition      string
	WindDirSpeed   string // "SE  10" or "Calm"
	WindGust       string // "Gusts to 20" or ""
	Humidity       string
	Dewpoint       string
	Ceiling        string
	Visibility     string
	Pressure       string
	PressureDir    string
	HeatIndexLabel string
	HeatIndex      string
	Location       string
	Icon           string
	Stale          bool

	// raw values for the ticker
	tickerSegments []string
}

func NewCurrentWeather(svc *weather.Service, cfg config.Config) *CurrentWeatherDisplay {
	return &CurrentWeatherDisplay{
		BaseDisplay: engine.NewBaseDisplay(1, "current-weather", "Current Conditions", true),
		svc:         svc,
		cfg:         cfg,
	}
}

func (d *CurrentWeatherDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params) {
		return nil
	}
	conv := units.New(params.Units)

	var obs *nws.ObservationResponse
	for _, url := range params.StationURLs {
		candidate, err := d.svc.Client.GetObservations(url, 5)
		if err != nil || len(candidate.Features) == 0 {
			continue
		}
		props := candidate.Features[0].Properties
		if props.Temperature.Value == nil || props.TextDescription == "" {
			continue
		}
		obs = candidate
		break
	}
	if obs == nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}

	props := obs.Features[0].Properties
	pressureDir := ""
	if len(obs.Features) > 1 {
		cur := props.BarometricPressure.Value
		prev := obs.Features[1].Properties.BarometricPressure.Value
		if cur != nil && prev != nil {
			diff := *cur - *prev
			if diff > 150 {
				pressureDir = "R"
			}
			if diff < -150 {
				pressureDir = "F"
			}
		}
	}

	condition := props.TextDescription
	if len(condition) > 15 {
		condition = weather.ShortCondition(condition)
	}

	windSpeed := conv.WindKMH(props.WindSpeed.Value)
	windDirSpeed := "Calm"
	windDir := units.DirectionToNSEW(props.WindDirection.Value)
	if windSpeed != "Calm" && windSpeed != "-" {
		// upstream pads: direction padEnd(3) + speed padStart(3)
		windDirSpeed = fmt.Sprintf("%-3s%3s", windDir, windSpeed)
	}

	gust := ""
	gustVal := conv.WindKMH(props.WindGust.Value)
	if gustVal != "-" && gustVal != "Calm" {
		gust = "Gusts to " + gustVal
	}

	location := weather.CleanLocation(params.City)
	if len(location) > 20 {
		location = location[:20]
	}

	tempUnit := conv.TempUnit()

	heatLabel, heatValue := "", ""
	temp := conv.TempC(props.Temperature.Value)
	if props.HeatIndex.Value != nil {
		h := conv.TempC(props.HeatIndex.Value)
		if h != temp {
			heatLabel, heatValue = "Heat Index:", h+degree()
		}
	} else if props.WindChill.Value != nil {
		wc := conv.TempC(props.WindChill.Value)
		if wc != "" && wc != temp {
			heatLabel, heatValue = "Wind Chill:", wc+degree()
		}
	}

	ceiling := conv.CeilingM(ceilingValue(props))
	if ceiling != "Unlimited" {
		ceiling += conv.CeilingUnit()
	}

	d.data = &currentWeatherData{
		Temperature:    temp + degree(),
		TempUnit:       tempUnit,
		Condition:      condition,
		WindDirSpeed:   windDirSpeed,
		WindGust:       gust,
		Humidity:       formatPercent(props.RelativeHumidity.Value),
		Dewpoint:       conv.TempC(props.Dewpoint.Value) + degree(),
		Ceiling:        ceiling,
		Visibility:     conv.VisibilityM(props.Visibility.Value) + conv.VisibilityUnit(),
		Pressure:       conv.PressurePa(props.BarometricPressure.Value),
		PressureDir:    pressureDir,
		HeatIndexLabel: heatLabel,
		HeatIndex:      heatValue,
		Location:       location,
		Icon:           icons.LargeIcon(props.Icon),
	}

	if ts, err := time.Parse(time.RFC3339, props.Timestamp); err == nil && time.Since(ts) > 80*time.Minute {
		d.data.Stale = true
	}

	// ticker segments (ported from currentweatherscroll.mjs)
	segs := []string{
		fmt.Sprintf("Conditions at %s", location),
	}
	tempSeg := fmt.Sprintf("Temp: %s%s%s", temp, degree(), tempUnit)
	if heatLabel != "" {
		tempSeg += fmt.Sprintf("    %s %s%s", heatLabel, heatValue, tempUnit)
	}
	segs = append(segs, tempSeg)
	segs = append(segs, fmt.Sprintf("Humidity: %s   Dewpoint: %s%s", d.data.Humidity, d.data.Dewpoint, tempUnit))
	segs = append(segs, fmt.Sprintf("Barometric Pressure: %s %s", d.data.Pressure, pressureDir))
	if windDirSpeed != "Calm" {
		wind := fmt.Sprintf("Wind: %s %s %s", windDir, windSpeed, conv.WindUnit())
		if gustVal != "-" && gustVal != "Calm" {
			wind += "  Gusts to " + gustVal
		}
		segs = append(segs, wind)
	} else {
		segs = append(segs, "Wind: Calm")
	}
	tickerCeiling := ceiling
	if tickerCeiling != "Unlimited" {
		// the ticker formats ceiling with a space before the unit
		tickerCeiling = conv.CeilingM(ceilingValue(props)) + " " + conv.CeilingUnit()
	}
	segs = append(segs, fmt.Sprintf("Visib: %s  Ceiling: %s", d.data.Visibility, tickerCeiling))
	d.data.tickerSegments = segs

	d.Timing().TotalScreens = 1
	d.Timing().CalcNavTiming()
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func ceilingValue(props nws.ObservationProperties) *float64 {
	if len(props.CloudLayers) > 0 {
		return props.CloudLayers[0].Base.Value
	}
	v := float64(0)
	return &v
}

func formatPercent(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%.0f%%", *v)
}

// Draw renders per upstream current-weather.ejs / _current-weather.scss:
// blue box content area x=64..576 starting y=90; left col 255px wide, right col
// 255px wide anchored right; rows pitch lh24+12.
func (d *CurrentWeatherDisplay) Draw(c *render.Canvas, screenIndex int) error {
	_ = c.DrawBackground("backgrounds/1.png")
	titleTop := "Current"
	if d.data != nil && d.data.Stale {
		titleTop = "Recent"
	}
	drawHeaderDual(c, d.Params(), titleTop, "Conditions", true)
	if d.data == nil {
		return nil
	}

	leftX := blueBoxMargin       // 64
	leftW := 255                 //
	rightX := 640 - 64 - 255     // 321
	rightW := 255                //
	top := mainTop + 20          // col margin-top 10 + padding-top 10

	styleTemp := render.Style{Family: render.FontStar4000Large, Size: 32, Color: colWhite, Shadow: true}
	styleCond := render.Style{Family: render.FontStar4000Extended, Size: 32, Color: colWhite, Shadow: true}
	styleRow := render.Style{Family: render.FontStar4000Large, Size: 20, Color: colWhite, Shadow: true}
	styleLoc := render.Style{Family: render.FontStar4000Large, Size: 20, Color: colTitle, Shadow: true}

	// left column: temp, condition, icon, wind, gusts
	y := top
	c.TextCenterIn(styleTemp, d.data.Temperature, leftX, leftW, y)
	y += 40
	c.TextCenterIn(styleCond, d.data.Condition, leftX, leftW, y)
	y += 42
	_ = c.ImageCenteredIn(d.data.Icon, leftX, y, leftW, 75)
	y += 85
	// wind row: label left, value right (wind-container margin-left 10)
	c.Text(styleCond, "Wind:", leftX+10, y)
	c.TextRight(styleCond, strings.TrimSpace(d.data.WindDirSpeed), leftX+leftW-10, y)
	y += 42
	if d.data.WindGust != "" {
		c.TextRight(styleCond, d.data.WindGust, leftX+leftW-10, y)
	}

	// right column: location + data rows
	ry := top
	c.Text(styleLoc, d.data.Location, rightX, ry+4)
	ry += 4 + 32 + 10 // location padding-top 4, height 32, margin-bottom 10

	row := func(label, value string, style render.Style) {
		c.Text(style, label, rightX+20, ry)
		c.TextRight(style, value, rightX+rightW-10, ry)
		ry += 36 // line-height 24 + margin-bottom 12
	}
	row("Humidity:", d.data.Humidity, styleRow)
	row("Dewpoint:", d.data.Dewpoint, styleRow)
	row("Ceiling:", d.data.Ceiling, styleRow)
	row("Visibility:", d.data.Visibility, styleRow)
	row("Pressure:", d.data.Pressure+" "+d.data.PressureDir, styleRow)
	if d.data.HeatIndexLabel != "" {
		heatStyle := styleRow
		if d.data.HeatIndexLabel == "Heat Index:" {
			heatStyle.Color = colHeatIndex
		} else {
			heatStyle.Color = colExtendedLow
		}
		row(d.data.HeatIndexLabel, d.data.HeatIndex, heatStyle)
	}
	return nil
}

// TickerSegments exposes the bottom-bar text segments.
func (d *CurrentWeatherDisplay) TickerSegments() []string {
	if d.data == nil {
		return nil
	}
	return d.data.tickerSegments
}
