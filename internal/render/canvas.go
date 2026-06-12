package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"image/gif"
	"image/png"
	"unsafe"

	"github.com/amcchord/ws4000/internal/assets"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

type FontSet struct {
	Regular  *ttf.Font
	Extended *ttf.Font
	Large    *ttf.Font
	Small    *ttf.Font
}

type Canvas struct {
	renderer  *sdl.Renderer
	fonts     *FontSet
	texture   *sdl.Texture
	pixels    *image.RGBA
	scanlines bool
}

func LoadFonts() (*FontSet, error) {
	load := func(variant string, size int) (*ttf.Font, error) {
		data, err := assets.Read(assets.FontPath(variant))
		if err != nil {
			return nil, err
		}
		rw, err := sdl.RWFromMem(data)
		if err != nil {
			return nil, err
		}
		font, err := ttf.OpenFontRW(rw, 1, size)
		if err != nil {
			return nil, err
		}
		return font, nil
	}

	regular, err := load("", 16)
	if err != nil {
		return nil, err
	}
	extended, err := load("extended", 16)
	if err != nil {
		return nil, err
	}
	large, err := load("large", 24)
	if err != nil {
		return nil, err
	}
	small, err := load("small", 12)
	if err != nil {
		return nil, err
	}

	return &FontSet{
		Regular:  regular,
		Extended: extended,
		Large:    large,
		Small:    small,
	}, nil
}


func NewCanvas(renderer *sdl.Renderer, fonts *FontSet) (*Canvas, error) {
	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, LogicalWidth, LogicalHeight)
	if err != nil {
		return nil, err
	}
	c := &Canvas{
		renderer: renderer,
		fonts:    fonts,
		texture:  texture,
		pixels:   image.NewRGBA(image.Rect(0, 0, LogicalWidth, LogicalHeight)),
	}
	c.Clear()
	return c, nil
}

func (c *Canvas) Texture() interface{} { return c.texture }

func (c *Canvas) Size() (int, int) { return LogicalWidth, LogicalHeight }

func (c *Canvas) Clear() {
	imagedraw.Draw(c.pixels, c.pixels.Bounds(), &image.Uniform{C: ColorGradient2}, image.Point{}, imagedraw.Src)
	grad := image.NewUniform(ColorGradient1)
	top := image.Rect(0, 0, LogicalWidth, LogicalHeight/2)
	imagedraw.DrawMask(c.pixels, top, grad, image.Point{}, image.NewUniform(color.Alpha{A: 128}), image.Point{}, imagedraw.Over)
}

func (c *Canvas) DrawScanlines(enabled bool) {
	c.scanlines = enabled
	if !enabled {
		return
	}
	for y := 0; y < LogicalHeight; y += 2 {
		line := image.Rect(0, y, LogicalWidth, y+1)
		imagedraw.Draw(c.pixels, line, &image.Uniform{C: color.RGBA{A: 40}}, image.Point{}, imagedraw.Over)
	}
}

func (c *Canvas) fontFor(name string) *ttf.Font {
	switch name {
	case "large":
		return c.fonts.Large
	case "small":
		return c.fonts.Small
	case "extended":
		return c.fonts.Extended
	default:
		return c.fonts.Regular
	}
}

func (c *Canvas) DrawText(fontName, text string, x, y int, fg interface{}, shadow bool) {
	c.drawTextInternal(c.fontFor(fontName), text, x, y, fg, shadow, 0)
}

func (c *Canvas) DrawTextRight(fontName, text string, x, y int, fg interface{}, shadow bool) {
	font := c.fontFor(fontName)
	w, _, err := font.SizeUTF8(text)
	if err != nil {
		return
	}
	c.drawTextInternal(font, text, x-w, y, fg, shadow, 0)
}

func (c *Canvas) DrawTextCentered(fontName, text string, x, y, width int, fg interface{}, shadow bool) {
	font := c.fontFor(fontName)
	w, _, err := font.SizeUTF8(text)
	if err != nil {
		return
	}
	c.drawTextInternal(font, text, x+(width-w)/2, y, fg, shadow, 0)
}

func (c *Canvas) drawTextInternal(font *ttf.Font, text string, x, y int, fg interface{}, shadow bool, _ int) {
	color := ColorDateTime
	if fg != nil {
		color = ColorTitle
	}
	if shadow {
		surf, err := font.RenderUTF8Solid(text, sdl.Color{R: 0, G: 0, B: 0, A: 255})
		if err == nil {
			defer surf.Free()
			c.blitSurface(surf, x+1, y+1)
		}
	}
	surf, err := font.RenderUTF8Solid(text, sdl.Color{R: color.R, G: color.G, B: color.B, A: color.A})
	if err != nil {
		return
	}
	defer surf.Free()
	c.blitSurface(surf, x, y)
}

func (c *Canvas) blitSurface(surf *sdl.Surface, x, y int) {
	src := surf.Pixels()
	w := int(surf.W)
	h := int(surf.H)
	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			px := row*int(surf.Pitch) + col*4
			if px+3 >= len(src) {
				continue
			}
			a := src[px+3]
			if a == 0 {
				continue
			}
			dstX := x + col
			dstY := y + row
			if dstX < 0 || dstY < 0 || dstX >= LogicalWidth || dstY >= LogicalHeight {
				continue
			}
			c.pixels.SetRGBA(dstX, dstY, color.RGBA{
				R: src[px],
				G: src[px+1],
				B: src[px+2],
				A: a,
			})
		}
	}
}

func (c *Canvas) DrawRect(x, y, w, h int, _ interface{}) {
	rect := image.Rect(x, y, x+w, y+h)
	imagedraw.Draw(c.pixels, rect, &image.Uniform{C: ColorBlueBox}, image.Point{}, imagedraw.Src)
}

func (c *Canvas) DrawBackground(path string) error {
	img, err := c.loadImage(path)
	if err != nil {
		return err
	}
	imagedraw.Draw(c.pixels, c.pixels.Bounds(), img, image.Point{}, imagedraw.Over)
	return nil
}

func (c *Canvas) DrawImage(path string, x, y, w, h int) error {
	return c.BlitImage(path, x, y, w, h)
}

func (c *Canvas) DrawGIF(path string, x, y, w, h, frame int) error {
	return c.BlitImage(path, x, y, w, h)
}

func (c *Canvas) loadImage(path string) (image.Image, error) {
	data, err := assets.Read(path)
	if err != nil {
		return nil, err
	}
	if len(data) >= 8 && string(data[0:8]) == "\x89PNG\r\n\x1a\n" {
		return png.Decode(bytes.NewReader(data))
	}
	if len(data) >= 3 && string(data[0:3]) == "GIF" {
		g, err := gif.DecodeAll(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		if len(g.Image) == 0 {
			return nil, fmt.Errorf("empty gif")
		}
		return g.Image[0], nil
	}
	return webp.Decode(bytes.NewReader(data))
}

func (c *Canvas) PresentToRenderer() error {
	if len(c.pixels.Pix) == 0 {
		return fmt.Errorf("empty canvas")
	}
	return c.texture.Update(nil, unsafe.Pointer(&c.pixels.Pix[0]), c.pixels.Stride)
}

func (c *Canvas) Snapshot() *image.RGBA {
	return c.pixels
}

func (c *Canvas) BlitImage(path string, x, y, w, h int) error {
	img, err := c.loadImage(path)
	if err != nil {
		return err
	}
	if w > 0 && h > 0 && (w != img.Bounds().Dx() || h != img.Bounds().Dy()) {
		scaled := image.NewRGBA(image.Rect(0, 0, w, h))
		xdraw.NearestNeighbor.Scale(scaled, scaled.Bounds(), img, img.Bounds(), imagedraw.Over, nil)
		imagedraw.Draw(c.pixels, image.Rect(x, y, x+w, y+h), scaled, image.Point{}, imagedraw.Over)
		return nil
	}
	dst := image.Rect(x, y, x+img.Bounds().Dx(), y+img.Bounds().Dy())
	imagedraw.Draw(c.pixels, dst, img, img.Bounds().Min, imagedraw.Over)
	return nil
}

func (c *Canvas) Close() {
	if c.texture != nil {
		c.texture.Destroy()
	}
}
