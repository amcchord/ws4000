package progress

import (
	"fmt"

	"github.com/amcchord/ws4000/internal/assets"
	"github.com/amcchord/ws4000/internal/engine"
)

type Display struct {
	loaded int
	total  int
	lines  []string
}

func New(total int) *Display {
	return &Display{total: total}
}

func (p *Display) Update(disps []engine.Display, loaded int) {
	p.loaded = loaded
	p.lines = nil
	for _, d := range disps {
		line := fmt.Sprintf("%-22s %s", d.Name(), d.Status().String())
		p.lines = append(p.lines, line)
	}
}

func (p *Display) Draw(canvas engine.Canvas) error {
	_ = canvas.DrawBackground(assets.BackgroundPath("progress"))
	canvas.DrawText("large", "Please Stand By", 180, 180, titleColor(), true)
	canvas.DrawText("regular", fmt.Sprintf("Loading %d/%d", p.loaded, len(p.lines)), 250, 220, nil, true)
	y := 260
	for _, line := range p.lines {
		canvas.DrawText("small", line, 120, y, nil, true)
		y += 14
	}
	return nil
}

func titleColor() interface{} {
	return struct{ title bool }{true}
}
