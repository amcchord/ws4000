// Package progress renders the "loading" screen per upstream progress.ejs/_progress.scss.
package progress

import (
	"image/color"
	"strings"
	"time"

	"github.com/amcchord/ws4000/internal/engine"
	"github.com/amcchord/ws4000/internal/render"
)

var (
	colTitle   = color.RGBA{R: 255, G: 255, B: 0, A: 255}
	colWhite   = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	colGreen   = color.RGBA{R: 0, G: 255, B: 0, A: 255}
	colRed     = color.RGBA{R: 255, G: 0, B: 0, A: 255}
	colGray    = color.RGBA{R: 192, G: 192, B: 192, A: 255}
	colBlueBox = color.RGBA{R: 38, G: 35, B: 90, A: 255}

	// loading bar gradient (shared/_colors.scss)
	gradLoading = []color.RGBA{
		{R: 0x09, G: 0x24, B: 0x6f, A: 255},
		{R: 0x36, G: 0x4a, B: 0xc0, A: 255},
		{R: 0x4f, G: 0x99, B: 0xf9, A: 255},
		{R: 0x8f, G: 0xfd, B: 0xfa, A: 255},
	}
)

type Display struct{}

func New() *Display { return &Display{} }

func statusColor(s engine.LoadStatus) color.RGBA {
	switch s {
	case engine.StatusLoading, engine.StatusRetrying:
		return colTitle
	case engine.StatusLoaded:
		return colGreen
	case engine.StatusFailed:
		return colRed
	default:
		return colGray
	}
}

func statusLabel(s engine.LoadStatus) string {
	if s == engine.StatusLoaded {
		// upstream shows "Press Here" (clickable) once a display loads
		return "Press Here"
	}
	return s.String()
}

// Draw renders the progress screen with per-display status and a progress bar.
func (p *Display) Draw(c *render.Canvas, displays []engine.Display, params *engine.WeatherParams) error {
	_ = c.DrawBackground("backgrounds/1.png")

	// header (dual title + clock)
	titleStyle := render.Style{Family: render.FontStar4000, Size: 32, Color: colTitle, Shadow: true}
	_ = c.Image("logos/logo-corner.png", 50, 30)
	c.Text(titleStyle, "WeatherStar", 170, 27)
	c.Text(titleStyle, "4000+", 170, 56)

	clockStyle := render.Style{Family: render.FontStar4000Small, Size: 32, Color: colWhite, Shadow: true}
	var loc *time.Location
	if params != nil {
		loc = params.TZ()
	} else {
		loc = time.Local
	}
	now := time.Now().In(loc)
	c.TextRight(clockStyle, strings.ToUpper(now.Format("3:04:05 PM")), 585, 30)
	c.TextRight(clockStyle, strings.ToUpper(now.Format("Mon Jan 2")), 585, 52)

	// items: Star4000 Extended 25px, line-height 28, container at main top + 15
	itemStyle := render.Style{Family: render.FontStar4000Extended, Size: 25, Color: colWhite, Shadow: true}
	dotStyle := itemStyle

	y := 90 + 15
	x := 64 + 10
	rightEdge := 576 - 10
	loaded := 0
	total := 0
	for _, d := range displays {
		if d.Enabled() {
			total++
			if d.Status() != engine.StatusLoading {
				loaded++
			}
		}
		name := d.Name()
		status := statusLabel(d.Status())
		stStyle := itemStyle
		stStyle.Color = statusColor(d.Status())

		c.Text(itemStyle, name, x, y)
		// dotted leader between name and status
		nameW := c.MeasureText(itemStyle, name)
		statusW := c.MeasureText(stStyle, status)
		dotStart := x + nameW + 4
		dotEnd := rightEdge - statusW - 8
		if dotEnd > dotStart {
			dots := strings.Repeat(".", (dotEnd-dotStart)/c.MeasureText(dotStyle, "."))
			c.Text(dotStyle, dots, dotStart, y)
		}
		// status with blue-box backing
		c.FillRect(rightEdge-statusW-4, y, statusW+8, 26, colBlueBox)
		c.Text(stStyle, status, rightEdge-statusW, y)
		y += 28
	}

	// progress bar (in the ticker area): white container 524px wide centered
	barX := (640 - 524) / 2
	barY := 423
	c.FillRect(barX, barY, 524, 28, color.RGBA{A: 255})        // border
	c.FillRect(barX+2, barY+2, 520, 24, colWhite)              // container
	fillW := 0
	if total > 0 {
		fillW = 516 * loaded / total
	}
	// striped gradient: repeating 40px pattern of the four loading colors
	pattern := []int{0, 1, 2, 3, 2, 1, 0, 0}
	for px := 0; px < fillW; px += 5 {
		idx := (px / 5) % len(pattern)
		w := 5
		if px+w > fillW {
			w = fillW - px
		}
		c.FillRect(barX+4+px, barY+4, w, 20, gradLoading[pattern[idx]])
	}
	return nil
}
