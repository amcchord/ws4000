package displays

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"time"

	"github.com/amcchord/ws4000/internal/config"
	"github.com/amcchord/ws4000/internal/data/weather"
	"github.com/amcchord/ws4000/internal/engine"
	"github.com/amcchord/ws4000/internal/render"
)

// Upstream spc-outlook.mjs: categorical outlook geojson per day,
// point-in-polygon test, bar widths by risk category.
var spcBarSizes = map[string]int{
	"TSTM": 60,
	"MRGL": 150,
	"SLGT": 210,
	"ENH":  270,
	"MDT":  330,
	"HIGH": 390,
}

var spcRiskOrder = map[string]int{
	"TSTM": 0, "MRGL": 1, "SLGT": 2, "ENH": 3, "MDT": 4, "HIGH": 5,
}

type spcGeoJSON struct {
	Features []struct {
		Properties struct {
			Label string `json:"LABEL"`
			DN    int    `json:"DN"`
		} `json:"properties"`
		Geometry struct {
			Type        string          `json:"type"`
			Coordinates json.RawMessage `json:"coordinates"`
		} `json:"geometry"`
	} `json:"features"`
}

type spcDay struct {
	Name  string
	Label string // risk category or "" for none
}

type SPCOutlookDisplay struct {
	*engine.BaseDisplay
	svc  *weather.Service
	cfg  config.Config
	days []spcDay
}

func NewSPCOutlook(svc *weather.Service, cfg config.Config) *SPCOutlookDisplay {
	d := &SPCOutlookDisplay{
		BaseDisplay: engine.NewBaseDisplay(10, "spc-outlook", "SPC Outlook", true),
		svc:         svc,
		cfg:         cfg,
	}
	d.Timing().TotalScreens = 0
	return d
}

func (d *SPCOutlookDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params) {
		return nil
	}
	client := &http.Client{Timeout: 20 * time.Second}
	loc := params.TZ()
	now := time.Now().In(loc)

	d.days = nil
	anyRisk := false
	for day := 1; day <= 3; day++ {
		url := fmt.Sprintf("https://www.spc.noaa.gov/products/outlook/day%dotlk_cat.nolyr.geojson", day)
		label := ""
		resp, err := client.Get(url)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			var geo spcGeoJSON
			if json.Unmarshal(body, &geo) == nil {
				best := -1
				for _, f := range geo.Features {
					if !pointInGeometry(params.Longitude, params.Latitude, f.Geometry.Type, f.Geometry.Coordinates) {
						continue
					}
					if rank, ok := spcRiskOrder[f.Properties.Label]; ok && rank > best {
						best = rank
						label = f.Properties.Label
					}
				}
			}
		}
		if label != "" {
			anyRisk = true
		}
		d.days = append(d.days, spcDay{
			Name:  now.Add(time.Duration(day-1) * 24 * time.Hour).Format("Monday"),
			Label: label,
		})
	}

	if !anyRisk {
		d.Timing().TotalScreens = 0
		d.SetStatus(engine.StatusNoData)
		return nil
	}
	d.Timing().TotalScreens = 1
	d.Timing().Delay = 2
	d.Timing().CalcNavTiming()
	d.SetStatus(engine.StatusLoaded)
	return nil
}

// pointInGeometry implements ray-casting for Polygon and MultiPolygon coordinates.
func pointInGeometry(lon, lat float64, geomType string, raw json.RawMessage) bool {
	switch geomType {
	case "Polygon":
		var poly [][][]float64
		if json.Unmarshal(raw, &poly) != nil {
			return false
		}
		return pointInPolygon(lon, lat, poly)
	case "MultiPolygon":
		var multi [][][][]float64
		if json.Unmarshal(raw, &multi) != nil {
			return false
		}
		for _, poly := range multi {
			if pointInPolygon(lon, lat, poly) {
				return true
			}
		}
	}
	return false
}

func pointInPolygon(lon, lat float64, poly [][][]float64) bool {
	if len(poly) == 0 {
		return false
	}
	// outer ring must contain; holes must not
	if !pointInRing(lon, lat, poly[0]) {
		return false
	}
	for _, hole := range poly[1:] {
		if pointInRing(lon, lat, hole) {
			return false
		}
	}
	return true
}

func pointInRing(lon, lat float64, ring [][]float64) bool {
	inside := false
	n := len(ring)
	j := n - 1
	for i := 0; i < n; i++ {
		xi, yi := ring[i][0], ring[i][1]
		xj, yj := ring[j][0], ring[j][1]
		if (yi > lat) != (yj > lat) &&
			lon < (xj-xi)*(lat-yi)/(yj-yi)+xi {
			inside = !inside
		}
		j = i
	}
	return inside
}

func (d *SPCOutlookDisplay) Draw(c *render.Canvas, screenIndex int) error {
	_ = c.DrawBackground("backgrounds/6.png")
	drawHeaderDual(c, d.Params(), "Storm Prediction", "Center Outlook", false)

	// diagonal risk-level labels (Star4000 Small 32, white per upstream)
	riskStyle := render.Style{Family: render.FontStar4000Small, Size: 32, Color: colWhite, Shadow: true}
	labels := []string{"High", "Moderate", "Enhanced", "Slight", "Marginal", "T'Storm"}
	for i, label := range labels {
		x := 216 + (5-i)*20
		y := mainTop + i*20 - 14
		c.Text(riskStyle, label, x, y)
	}

	// day rows with risk bars
	grayLight := color.RGBA{R: 153, G: 153, B: 153, A: 255}
	grayMid := color.RGBA{R: 128, G: 128, B: 128, A: 255}
	grayDark := color.RGBA{R: 102, G: 102, B: 102, A: 255}

	dayStyle := render.Style{Family: render.FontStar4000, Size: 32, Color: colWhite, Shadow: true}
	baseY := mainTop + 120
	for i, day := range d.days {
		y := baseY + i*60 + 20
		c.TextRight(dayStyle, day.Name, 210, y)
		if day.Label == "" {
			continue
		}
		w := spcBarSizes[day.Label]
		// outset border bar with vertical gray gradient (approximated in 3 bands)
		c.FillRect(220, y, w, 40, grayLight)
		c.FillRect(223, y+3, w-6, 34, grayDark)
		c.FillRect(223, y+3, w-6, 17, grayMid)
		c.FillRect(223, y+12, w-6, 14, grayLight)
	}
	return nil
}
