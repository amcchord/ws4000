package displays

import (
	"strconv"
	"strings"
	"time"

	"github.com/amcchord/ws4000/internal/assets"
	"github.com/amcchord/ws4000/internal/config"
	"github.com/amcchord/ws4000/internal/data/icons"
	"github.com/amcchord/ws4000/internal/data/nws"
	"github.com/amcchord/ws4000/internal/data/units"
	"github.com/amcchord/ws4000/internal/data/weather"
	"github.com/amcchord/ws4000/internal/engine"
)

func drawBackground(canvas engine.Canvas, displayID string) {
	_ = canvas.DrawBackground(assets.BackgroundPath(displayID))
}

func drawHeader(canvas engine.Canvas, top, bottom string) {
	canvas.DrawText("large", top, 20, 18, titleColor(), true)
	canvas.DrawText("large", bottom, 20, 42, titleColor(), true)
}

func titleColor() interface{} {
	return struct{ title bool }{true}
}

func drawDateTime(canvas engine.Canvas, loc *time.Location) {
	now := time.Now()
	if loc != nil {
		now = now.In(loc)
	}
	date := strings.ToUpper(now.Format("Mon Jan 02"))
	timeStr := strings.ToUpper(now.Format("03:04:05 PM"))
	canvas.DrawTextRight("small", timeStr, 620, 8, nil, true)
	canvas.DrawTextRight("small", date, 620, 22, nil, true)
}

func loadLocation(tz string) *time.Location {
	if tz == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Local
	}
	return loc
}

type CurrentWeatherDisplay struct {
	*engine.BaseDisplay
	svc  *weather.Service
	cfg  config.Config
	data *currentWeatherData
}

type currentWeatherData struct {
	Temperature    string
	Condition      string
	Wind           string
	WindGust       string
	Humidity       string
	Dewpoint       string
	Ceiling        string
	CeilingUnit    string
	Visibility     string
	VisibilityUnit string
	Pressure       string
	PressureDir    string
	HeatIndexLabel string
	HeatIndex      string
	Location       string
	Icon           string
	Stale          bool
	TickerText     string
}

func NewCurrentWeather(svc *weather.Service, cfg config.Config) *CurrentWeatherDisplay {
	return &CurrentWeatherDisplay{
		BaseDisplay: engine.NewBaseDisplay(1, "current-weather", "Current Conditions", true),
		svc:         svc,
		cfg:         cfg,
	}
}

func (d *CurrentWeatherDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params, false) {
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
	wind := conv.WindMS(props.WindSpeed.Value)
	if wind != "Calm" {
		wind = units.DirectionToNSEW(props.WindDirection.Value) + wind
	}
	location := weather.CleanLocation(params.City)
	if len(location) > 20 {
		location = location[:20]
	}
	heatLabel := ""
	heatValue := ""
	if props.HeatIndex.Value != nil {
		t := conv.TempC(props.Temperature.Value)
		h := conv.TempC(props.HeatIndex.Value)
		if t != h {
			heatLabel = "Heat Index:"
			heatValue = h + conv.TempSymbol()
		}
	}
	if heatLabel == "" && props.WindChill.Value != nil {
		t := conv.TempC(props.Temperature.Value)
		wc := conv.TempC(props.WindChill.Value)
		if wc != "" && wc < t {
			heatLabel = "Wind Chill:"
			heatValue = wc + conv.TempSymbol()
		}
	}
	gust := conv.WindMS(props.WindGust.Value)
	if gust == "Calm" || gust == "-" {
		gust = "-"
	}
	d.data = &currentWeatherData{
		Temperature:    conv.TempC(props.Temperature.Value) + conv.TempSymbol(),
		Condition:      strings.ToUpper(condition),
		Wind:           wind,
		WindGust:       gust,
		Humidity:       formatPercent(props.RelativeHumidity.Value),
		Dewpoint:       conv.TempC(props.Dewpoint.Value) + conv.TempSymbol(),
		Ceiling:        conv.CeilingM(ceilingValue(props)),
		CeilingUnit:    conv.CeilingUnit(),
		Visibility:     conv.VisibilityM(props.Visibility.Value),
		VisibilityUnit: conv.VisibilityUnit(),
		Pressure:       conv.PressurePa(props.BarometricPressure.Value),
		PressureDir:    pressureDir,
		HeatIndexLabel: heatLabel,
		HeatIndex:      heatValue,
		Location:       strings.ToUpper(location),
		Icon:           icons.LargeIcon(props.Icon),
	}
	if ts, err := time.Parse(time.RFC3339, props.Timestamp); err == nil {
		if time.Since(ts) > 80*time.Minute {
			d.data.Stale = true
		}
	}
	d.data.TickerText = strings.ToUpper(props.TextDescription)
	d.Timing().TotalScreens = 1
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
	return strconv.Itoa(int(*v)) + "%"
}

func (d *CurrentWeatherDisplay) Draw(canvas engine.Canvas, screenIndex int) error {
	drawBackground(canvas, d.ID())
	titleTop := "Current"
	if d.data != nil && d.data.Stale {
		titleTop = "Recent"
	}
	drawHeader(canvas, titleTop, "Conditions")
	drawDateTime(canvas, loadLocation(d.Params().TimeZone))
	if d.data == nil {
		return nil
	}
	_ = canvas.DrawImage(d.data.Icon, 90, 120, 0, 0)
	canvas.DrawText("extended", d.data.Temperature, 80, 200, nil, true)
	canvas.DrawText("extended", d.data.Condition, 64, 240, nil, true)
	canvas.DrawText("regular", d.data.Wind, 170, 280, nil, true)
	if d.data.WindGust != "-" {
		canvas.DrawText("large", "Gusts to "+d.data.WindGust, 64, 310, nil, true)
	}
	canvas.DrawText("regular", d.data.Location, 64, 350, nil, true)
	canvas.DrawText("large", "Humidity:", 360, 120, nil, true)
	canvas.DrawText("large", d.data.Humidity, 560, 120, nil, true)
	canvas.DrawText("large", "Dewpoint:", 360, 150, nil, true)
	canvas.DrawText("large", d.data.Dewpoint, 560, 150, nil, true)
	canvas.DrawText("large", "Ceiling:", 360, 180, nil, true)
	canvas.DrawText("large", d.data.Ceiling+d.data.CeilingUnit, 560, 180, nil, true)
	canvas.DrawText("large", "Visibility:", 360, 210, nil, true)
	canvas.DrawText("large", d.data.Visibility+d.data.VisibilityUnit, 560, 210, nil, true)
	canvas.DrawText("large", "Pressure:", 360, 240, nil, true)
	canvas.DrawText("large", d.data.Pressure+" "+d.data.PressureDir, 560, 240, nil, true)
	if d.data.HeatIndexLabel != "" {
		canvas.DrawText("large", d.data.HeatIndexLabel, 360, 280, nil, true)
		canvas.DrawText("large", d.data.HeatIndex, 560, 280, nil, true)
	}
	return nil
}

func (d *CurrentWeatherDisplay) TickerText() string {
	if d.data == nil {
		return ""
	}
	return d.data.TickerText
}
