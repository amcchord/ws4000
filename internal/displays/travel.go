package displays

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/amcchord/ws4000/internal/assets"
	"github.com/amcchord/ws4000/internal/config"
	"github.com/amcchord/ws4000/internal/data/icons"
	"github.com/amcchord/ws4000/internal/data/weather"
	"github.com/amcchord/ws4000/internal/engine"
	"github.com/amcchord/ws4000/internal/render"
)

type travelCity struct {
	Name      string  `json:"Name"`
	Latitude  float64 `json:"Latitude"`
	Longitude float64 `json:"Longitude"`
	Point     struct {
		X   int    `json:"x"`
		Y   int    `json:"y"`
		WFO string `json:"wfo"`
	} `json:"point"`
}

type travelRow struct {
	City string
	Icon string
	Low  string
	High string
}

// Upstream _travel.scss: full width; purple header bar with LOW at 455, HIGH at 510;
// rows 72px Star4000 Large yellow; city x=80, icon centered 330..400, temps centered.
type TravelForecastDisplay struct {
	*engine.BaseDisplay
	svc     *weather.Service
	cfg     config.Config
	rows    []travelRow
	dayName string
}

func NewTravelForecast(svc *weather.Service, cfg config.Config) *TravelForecastDisplay {
	d := &TravelForecastDisplay{
		BaseDisplay: engine.NewBaseDisplay(5, "travel", "Travel Forecast", false),
		svc:         svc,
		cfg:         cfg,
	}
	// upstream scrolls 4 cities per screen
	return d
}

func (d *TravelForecastDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params) {
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

	type result struct {
		idx int
		row travelRow
		ok  bool
	}
	results := make([]result, len(cities))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for i, city := range cities {
		wg.Add(1)
		go func(i int, city travelCity) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			url := fmt.Sprintf("https://api.weather.gov/gridpoints/%s/%d,%d/forecast", city.Point.WFO, city.Point.X, city.Point.Y)
			forecast, err := d.svc.Client.GetForecast(url, params.Units)
			if err != nil {
				return
			}
			periods := forecast.Properties.Periods
			// find first day period and following night (upstream uses periods 1/2 relative to evening)
			var dayP, nightP *int
			for j := range periods {
				if periods[j].IsDaytime {
					dayP = &j
					break
				}
			}
			if dayP == nil || *dayP+1 >= len(periods) {
				return
			}
			n := *dayP + 1
			nightP = &n
			results[i] = result{
				idx: i,
				row: travelRow{
					City: city.Name,
					Icon: icons.LargeIcon(periods[*dayP].Icon),
					High: fmtTemp(periods[*dayP].Temperature),
					Low:  fmtTemp(periods[*nightP].Temperature),
				},
				ok: true,
			}
		}(i, city)
	}
	wg.Wait()

	d.rows = nil
	for _, r := range results {
		if r.ok {
			d.rows = append(d.rows, r.row)
		}
	}
	if len(d.rows) == 0 {
		d.SetStatus(engine.StatusNoData)
		d.Timing().TotalScreens = 0
		return nil
	}

	// day name for the title ("For Friday")
	tomorrow := time.Now().In(params.TZ()).Add(24 * time.Hour)
	d.dayName = tomorrow.Format("Monday")

	screens := (len(d.rows) + 3) / 4
	d.Timing().TotalScreens = screens
	d.Timing().Delay = 1
	d.Timing().CalcNavTiming()
	d.SetStatus(engine.StatusLoaded)
	return nil
}

func (d *TravelForecastDisplay) Draw(c *render.Canvas, screenIndex int) error {
	_ = c.DrawBackground("backgrounds/1.png")
	drawHeaderDual(c, d.Params(), "Travel Forecast", "For "+d.dayName, false)

	// purple sticky column header bar
	c.FillRect(0, mainTop, 640, 20, colColumnHead)
	c.TextCenter(styleColHead, "LOW", 455+25, mainTop-14)
	c.TextCenter(styleColHead, "HIGH", 510+30, mainTop-14)

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
		c.Text(rowStyle, row.City, 80, y)
		_ = c.ImageCenteredIn(row.Icon, 330, y-8, 70, 60)
		c.TextCenter(rowStyle, row.Low, 455+25, y)
		c.TextCenter(rowStyle, row.High, 510+30, y)
	}
	return nil
}
