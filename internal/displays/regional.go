package displays

import (
	"encoding/json"
	"fmt"
	"image"
	"math"
	"sort"
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

type regionalCity struct {
	City  string  `json:"city"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	Point struct {
		X   int    `json:"x"`
		Y   int    `json:"y"`
		WFO string `json:"wfo"`
	} `json:"point"`
}

type regionalCityData struct {
	Name string
	X, Y int // position in the 640x312 map viewport
	// per screen: 0 = current obs, 1..2 = forecast periods
	Temps [3]string
	Icons [3]string
}

// Upstream regionalforecast.mjs: basemap 2550x1600 cropped at sourceXY
// (offsets 240x117 = half the source viewport), scaled by 640/480 to the
// 640-wide display area; 3 screens (observations + 2 forecast periods).
type RegionalForecastDisplay struct {
	*engine.BaseDisplay
	svc         *weather.Service
	cfg         config.Config
	srcX, srcY  float64
	cities      []regionalCityData
	screenNames [3]string
}

func NewRegionalForecast(svc *weather.Service, cfg config.Config) *RegionalForecastDisplay {
	d := &RegionalForecastDisplay{
		BaseDisplay: engine.NewBaseDisplay(6, "regional-forecast", "Regional Forecast", true),
		svc:         svc,
		cfg:         cfg,
	}
	d.Timing().TotalScreens = 3
	d.Timing().Delay = 1
	d.Timing().CalcNavTiming()
	return d
}

const (
	regionalOffsetX = 240.0
	regionalOffsetY = 117.0
)

func regionalSourceXY(lat, lon float64) (float64, float64) {
	y := (50.5-lat)*55.2 - regionalOffsetY
	y = math.Max(0, math.Min(1600-regionalOffsetY*2, y))
	x := -((-127.5-lon)*41.775) - regionalOffsetX
	x = math.Max(0, math.Min(2550-regionalOffsetX*2, x))
	return x, y
}

func (d *RegionalForecastDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params) {
		return nil
	}
	data, err := assets.Read("data/regionalcities.json")
	if err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	var allCities []regionalCity
	if err := json.Unmarshal(data, &allCities); err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}

	d.srcX, d.srcY = regionalSourceXY(params.Latitude, params.Longitude)

	// visible lat/lon box (upstream getMinMaxLatitudeLongitude)
	maxLat := -(d.srcY/55.2 - 50.5)
	minLat := -((d.srcY+regionalOffsetY*2)/55.2 - 50.5)
	minLon := -(-d.srcX/41.775 + 127.5)
	maxLon := -(-(d.srcX+regionalOffsetX*2)/41.775 + 127.5)

	stations, _ := loadStations()

	// combine regional cities (spacing 1 degree) with stations (spacing 2.4),
	// cities first so they get priority, matching upstream
	type candidate struct {
		city       regionalCity
		targetDist float64
	}
	var candidates []candidate
	for _, city := range allCities {
		candidates = append(candidates, candidate{city: city, targetDist: 1.0})
	}
	if stations != nil {
		var ids []string
		for id := range stations {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			st := stations[id]
			candidates = append(candidates, candidate{
				city: regionalCity{
					City: st.City,
					Lat:  st.Lat,
					Lon:  st.Lon,
				},
				targetDist: 2.4,
			})
		}
	}

	var picked []regionalCity
	for _, cand := range candidates {
		city := cand.city
		if !(city.Lat > minLat && city.Lat < maxLat && city.Lon > minLon && city.Lon < maxLon-1) {
			continue
		}
		ok := true
		for _, p := range picked {
			dist := math.Sqrt(distanceSq(city.Lat, city.Lon, p.Lat, p.Lon))
			if dist < cand.targetDist {
				ok = false
				break
			}
		}
		if ok {
			picked = append(picked, city)
		}
		if len(picked) >= 10 {
			break
		}
	}
	conv := units.New("us") // upstream regional always shows US units on the map

	type cityResult struct {
		data regionalCityData
		ok   bool
	}
	results := make([]cityResult, len(picked))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for i, city := range picked {
		wg.Add(1)
		go func(i int, city regionalCity) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			cd := regionalCityData{Name: formatRegionalCity(city.City)}

			// position on the viewport (upstream getXYForCity against 640x312 area)
			x := (city.Lon - minLon) * 57
			y := (maxLat - city.Lat) * 70
			x = math.Max(40, math.Min(580, x))
			y = math.Max(30, math.Min(282, y))
			cd.X, cd.Y = int(x), int(y)

			// forecast for screens 1 and 2 (stations have no precomputed point,
			// so resolve one via the points API)
			var url string
			if city.Point.WFO != "" {
				url = fmt.Sprintf("https://api.weather.gov/gridpoints/%s/%d,%d/forecast", city.Point.WFO, city.Point.X, city.Point.Y)
			} else {
				point, err := d.svc.Client.GetPoint(city.Lat, city.Lon)
				if err != nil {
					return
				}
				url = point.Properties.Forecast
			}
			forecast, err := d.svc.Client.GetForecast(url, "us")
			if err != nil || len(forecast.Properties.Periods) < 2 {
				return
			}
			periods := filterExpired(forecast.Properties.Periods)
			if len(periods) < 2 {
				return
			}
			for s := 0; s < 2; s++ {
				cd.Temps[s+1] = fmtTemp(periods[s].Temperature)
				night := !periods[s].IsDaytime
				cd.Icons[s+1] = icons.SmallIcon(periods[s].Icon, night)
			}

			// current observation from nearest station for screen 0
			if stations != nil {
				best := ""
				bestDist := math.MaxFloat64
				for id, st := range stations {
					dist := distanceSq(st.Lat, st.Lon, city.Lat, city.Lon)
					if dist < bestDist {
						bestDist = dist
						best = id
					}
				}
				if best != "" {
					obs, err := d.svc.Client.GetObservations("https://api.weather.gov/stations/"+best, 1)
					if err == nil && len(obs.Features) > 0 && obs.Features[0].Properties.Temperature.Value != nil {
						p := obs.Features[0].Properties
						cd.Temps[0] = conv.TempC(p.Temperature.Value)
						hour := time.Now().Hour()
						cd.Icons[0] = icons.SmallIcon(p.Icon, hour < 6 || hour > 19)
					}
				}
			}
			if cd.Temps[0] == "" {
				// fall back to forecast period 0
				cd.Temps[0] = cd.Temps[1]
				cd.Icons[0] = cd.Icons[1]
			}
			results[i] = cityResult{data: cd, ok: true}
		}(i, city)
	}
	wg.Wait()

	var cities []regionalCityData
	for _, r := range results {
		if r.ok {
			cities = append(cities, r.data)
		}
	}
	if len(cities) == 0 {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	d.cities = cities

	// screen titles: current observations, then the next two period names
	d.screenNames = [3]string{"Observations", "Forecast", "Forecast"}
	url := params.ForecastURL
	if forecast, err := d.svc.Client.GetForecast(url, "us"); err == nil {
		periods := filterExpired(forecast.Properties.Periods)
		for s := 0; s < 2 && s < len(periods); s++ {
			d.screenNames[s+1] = periods[s].Name
		}
	}

	d.SetStatus(engine.StatusLoaded)
	return nil
}

func formatRegionalCity(name string) string {
	// upstream formatCity strips suffixes but keeps full names
	for _, suffix := range []string{" International Airport", " Airport", " Regional", " International"} {
		name = strings.TrimSuffix(name, suffix)
	}
	return name
}

func (d *RegionalForecastDisplay) Draw(c *render.Canvas, screenIndex int) error {
	_ = c.DrawBackground("backgrounds/5.png")
	if screenIndex < 0 {
		screenIndex = 0
	}
	if screenIndex > 2 {
		screenIndex = 2
	}
	// screen 0: "Regional / Observations"; screens 1-2: "Forecast for / <period>"
	if screenIndex == 0 {
		drawHeaderDual(c, d.Params(), "Regional", "Observations", false)
	} else {
		drawHeaderDual(c, d.Params(), "Forecast for", d.screenNames[screenIndex], false)
	}

	// draw cropped/scaled basemap: source 480x234 -> 640x312 at y=90
	baseMap, err := c.LoadImage("maps/basemap.webp")
	if err == nil {
		src := image.Rect(int(d.srcX), int(d.srcY), int(d.srcX)+480, int(d.srcY)+234)
		c.DrawGoImageScaled(baseMap, src, image.Rect(0, mainTop, 640, mainTop+312))
	}

	cityStyle := render.Style{Family: render.FontStar4000, Size: 20, Color: colWhite, Shadow: true}
	tempStyle := render.Style{Family: render.FontStar4000Large, Size: 28, Color: colTitle, Shadow: true}

	for _, city := range d.cities {
		// location container is at (x-40, y-35) relative to the map top
		bx := city.X - 40
		by := mainTop + city.Y - 35
		c.Text(cityStyle, city.Name, bx, by)
		if city.Icons[screenIndex] != "" {
			_ = c.ImageCenteredIn(city.Icons[screenIndex], bx+44, by+26, 47, 32)
		}
		c.TextRight(tempStyle, city.Temps[screenIndex], bx+40, by+28)
	}
	return nil
}
