package render

import "image/color"

var (
	ColorTitle       = color.RGBA{R: 255, G: 255, B: 0, A: 255}
	ColorDateTime    = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	ColorTextShadow  = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	ColorColumnHead  = color.RGBA{R: 255, G: 255, B: 0, A: 255}
	ColorColumnBG    = color.RGBA{R: 32, G: 0, B: 87, A: 255}
	ColorBlueBox     = color.RGBA{R: 38, G: 35, B: 90, A: 255}
	ColorExtendedLow = color.RGBA{R: 128, G: 128, B: 255, A: 255}
	ColorWindChill   = color.RGBA{R: 128, G: 128, B: 255, A: 255}
	ColorHeatIndex   = color.RGBA{R: 238, G: 0, B: 0, A: 255}
	ColorGradient1   = color.RGBA{R: 16, G: 32, B: 128, A: 255}
	ColorGradient2   = color.RGBA{R: 0, G: 16, B: 64, A: 255}
)

const (
	LogicalWidth  = 640
	LogicalHeight = 480
	BlueBoxMargin = 64
	BlueBoxWidth  = LogicalWidth - 2*BlueBoxMargin
)
