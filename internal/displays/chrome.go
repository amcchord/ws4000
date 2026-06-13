package displays

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/amcchord/ws4000/internal/engine"
	"github.com/amcchord/ws4000/internal/render"
)

// Upstream colors (shared/_colors.scss)
var (
	colTitle       = color.RGBA{R: 255, G: 255, B: 0, A: 255}   // yellow
	colWhite       = color.RGBA{R: 255, G: 255, B: 255, A: 255} // date-time and body text
	colColumnHead  = color.RGBA{R: 32, G: 0, B: 87, A: 255}     // purple column header bar
	colBlueBox     = color.RGBA{R: 38, G: 35, B: 90, A: 255}
	colExtendedLow = color.RGBA{R: 128, G: 128, B: 255, A: 255}
	colHeatIndex   = color.RGBA{R: 238, G: 0, B: 0, A: 255}
	colHazardRed   = color.RGBA{R: 112, G: 35, B: 35, A: 255}
	colStatusGreen = color.RGBA{R: 0, G: 255, B: 0, A: 255}
	colStatusGray  = color.RGBA{R: 192, G: 192, B: 192, A: 255}
)

// Text styles matching upstream CSS font assignments.
var (
	styleTitle      = render.Style{Family: render.FontStar4000, Size: 32, Color: colTitle, Shadow: true}
	styleClock      = render.Style{Family: render.FontStar4000Small, Size: 32, Color: colWhite, Shadow: true}
	styleBody       = render.Style{Family: render.FontStar4000, Size: 32, Color: colWhite, Shadow: true}
	styleBodyYellow = render.Style{Family: render.FontStar4000, Size: 32, Color: colTitle, Shadow: true}
	styleColHead    = render.Style{Family: render.FontStar4000Small, Size: 32, Color: colTitle, Shadow: true}
	styleTicker     = render.Style{Family: render.FontStar4000, Size: 32, Color: colWhite, Shadow: true}
	styleTickerHead = render.Style{Family: render.FontStar4000Small, Size: 26, Color: colWhite, Shadow: true}
)

// Standard layout constants (shared/_positions.scss and _weather-display.scss)
const (
	mainTop       = 90  // header height 60 + padding 30
	blueBoxMargin = 64  // $blue-box-margin
	blueBoxWidth  = 512 // 640 - 2*64
	scrollTop     = 403 // 480 - 77 ticker area height
)

// drawHeaderDual renders logo, dual-line yellow title, optional NOAA logo and clock.
func drawHeaderDual(c *render.Canvas, params *engine.WeatherParams, top, bottom string, noaa bool) {
	_ = c.Image("logos/logo-corner.png", 50, 30)
	c.Text(styleTitle, top, 170, 27)
	c.Text(styleTitle, bottom, 170, 56)
	if noaa {
		_ = c.Image("logos/noaa.gif", 356, 39)
	}
	drawClock(c, params)
}

// drawHeaderSingle renders logo and a single-line title.
func drawHeaderSingle(c *render.Canvas, params *engine.WeatherParams, title string, hasTime bool) {
	_ = c.Image("logos/logo-corner.png", 50, 30)
	c.Text(styleTitle, title, 170, 40)
	if hasTime {
		drawClock(c, params)
	}
}

// drawClock renders the date/time in the upper right (right edge x=585).
func drawClock(c *render.Canvas, params *engine.WeatherParams) {
	var loc *time.Location
	if params != nil {
		loc = params.TZ()
	} else {
		loc = time.Local
	}
	now := time.Now().In(loc)
	timeStr := strings.ToUpper(now.Format("3:04:05 PM"))
	dateStr := strings.ToUpper(now.Format("Mon Jan 2"))
	c.TextRight(styleClock, timeStr, 585, 30)
	c.TextRight(styleClock, dateStr, 585, 52)
}

// statusColor maps a load status to the upstream progress-screen color.
func statusColor(s engine.LoadStatus) color.RGBA {
	switch s {
	case engine.StatusLoading, engine.StatusRetrying:
		return colTitle
	case engine.StatusLoaded:
		return colStatusGreen
	case engine.StatusFailed:
		return colHeatIndex
	default:
		return colStatusGray
	}
}

func statusLabel(s engine.LoadStatus) string {
	if s == engine.StatusLoaded {
		return "Press Here"
	}
	return s.String()
}

// formatDegree renders e.g. 75 + degree sign.
func degree() string { return string(rune(176)) }

func fmtTemp(v int) string { return fmt.Sprintf("%d", v) }

// shortDayName gives the upstream 3-letter day, e.g. "MON".
func shortDayName(t time.Time) string {
	return strings.ToUpper(t.Format("Mon"))
}
