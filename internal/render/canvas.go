package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"image/gif"
	"image/png"
	"sync"
	"unsafe"

	"github.com/amcchord/ws4000/internal/assets"
	"github.com/veandco/go-sdl2/sdl"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

// Style describes a text style matching upstream CSS.
type Style struct {
	Family string
	Size   int
	Color  color.RGBA
	Shadow bool // CSS text-shadow: 3px 3px offset plus ~1.5px outline
}

// Canvas is a 640x480 CPU-side framebuffer mirroring the upstream logical canvas.
type Canvas struct {
	renderer *sdl.Renderer
	fonts    *FontManager
	texture  *sdl.Texture
	pixels   *image.RGBA

	imgMu    sync.Mutex
	imgCache map[string]image.Image
	// cache of rendered text masks keyed by family|size|text
	textMu    sync.Mutex
	textCache map[string]*image.Alpha
}

func NewCanvas(renderer *sdl.Renderer, fonts *FontManager) (*Canvas, error) {
	var texture *sdl.Texture
	if renderer != nil {
		// ABGR8888 matches image.RGBA's R,G,B,A byte order on little-endian.
		t, err := renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, LogicalWidth, LogicalHeight)
		if err != nil {
			return nil, err
		}
		texture = t
	}
	c := &Canvas{
		renderer:  renderer,
		fonts:     fonts,
		texture:   texture,
		pixels:    image.NewRGBA(image.Rect(0, 0, LogicalWidth, LogicalHeight)),
		imgCache:  make(map[string]image.Image),
		textCache: make(map[string]*image.Alpha),
	}
	c.Clear()
	return c, nil
}

func (c *Canvas) Size() (int, int) { return LogicalWidth, LogicalHeight }

func (c *Canvas) Snapshot() *image.RGBA { return c.pixels }

func (c *Canvas) Clear() {
	imagedraw.Draw(c.pixels, c.pixels.Bounds(), image.NewUniform(color.RGBA{A: 255}), image.Point{}, imagedraw.Src)
}

// --- images ---

func (c *Canvas) LoadImage(path string) (image.Image, error) {
	c.imgMu.Lock()
	if img, ok := c.imgCache[path]; ok {
		c.imgMu.Unlock()
		return img, nil
	}
	c.imgMu.Unlock()

	data, err := assets.Read(path)
	if err != nil {
		return nil, err
	}
	img, err := DecodeImage(data)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}

	c.imgMu.Lock()
	c.imgCache[path] = img
	c.imgMu.Unlock()
	return img, nil
}

// DecodeImage decodes PNG, GIF (first frame), or WebP bytes.
func DecodeImage(data []byte) (image.Image, error) {
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
		// composite first frame onto its bounds (frames may have offsets)
		out := image.NewRGBA(image.Rect(0, 0, g.Config.Width, g.Config.Height))
		imagedraw.Draw(out, g.Image[0].Bounds(), g.Image[0], g.Image[0].Bounds().Min, imagedraw.Over)
		return out, nil
	}
	return webp.Decode(bytes.NewReader(data))
}

func (c *Canvas) DrawBackground(path string) error {
	img, err := c.LoadImage(path)
	if err != nil {
		return err
	}
	imagedraw.Draw(c.pixels, c.pixels.Bounds(), img, img.Bounds().Min, imagedraw.Src)
	return nil
}

// Image draws an asset at natural size with alpha blending.
func (c *Canvas) Image(path string, x, y int) error {
	img, err := c.LoadImage(path)
	if err != nil {
		return err
	}
	b := img.Bounds()
	imagedraw.Draw(c.pixels, image.Rect(x, y, x+b.Dx(), y+b.Dy()), img, b.Min, imagedraw.Over)
	return nil
}

// ImageScaled draws an asset scaled to w x h (nearest neighbor, like upstream pixelated rendering).
func (c *Canvas) ImageScaled(path string, x, y, w, h int) error {
	img, err := c.LoadImage(path)
	if err != nil {
		return err
	}
	c.DrawGoImageScaled(img, img.Bounds(), image.Rect(x, y, x+w, y+h))
	return nil
}

// ImageCenteredIn draws an asset horizontally centered in [x, x+boxW), top-aligned
// at y, scaled down proportionally if taller than maxH (CSS max-height behavior).
func (c *Canvas) ImageCenteredIn(path string, x, y, boxW, maxH int) error {
	img, err := c.LoadImage(path)
	if err != nil {
		return err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if maxH > 0 && h > maxH {
		w = w * maxH / h
		h = maxH
	}
	dx := x + (boxW-w)/2
	c.DrawGoImageScaled(img, b, image.Rect(dx, y, dx+w, y+h))
	return nil
}

func (c *Canvas) DrawGoImage(img image.Image, x, y int) {
	b := img.Bounds()
	imagedraw.Draw(c.pixels, image.Rect(x, y, x+b.Dx(), y+b.Dy()), img, b.Min, imagedraw.Over)
}

func (c *Canvas) DrawGoImageScaled(img image.Image, src image.Rectangle, dst image.Rectangle) {
	if src.Dx() == dst.Dx() && src.Dy() == dst.Dy() {
		imagedraw.Draw(c.pixels, dst, img, src.Min, imagedraw.Over)
		return
	}
	xdraw.NearestNeighbor.Scale(c.pixels, dst, img, src, xdraw.Over, nil)
}

func (c *Canvas) FillRect(x, y, w, h int, col color.RGBA) {
	imagedraw.Draw(c.pixels, image.Rect(x, y, x+w, y+h), image.NewUniform(col), image.Point{}, imagedraw.Over)
}

// --- text ---

// textMask renders a glyph coverage mask for the string (cached).
func (c *Canvas) textMask(family string, size int, text string) (*image.Alpha, error) {
	key := fmt.Sprintf("%s|%d|%s", family, size, text)
	c.textMu.Lock()
	if m, ok := c.textCache[key]; ok {
		c.textMu.Unlock()
		return m, nil
	}
	c.textMu.Unlock()

	font, err := c.fonts.Get(family, size)
	if err != nil {
		return nil, err
	}
	surf, err := font.RenderUTF8Blended(text, sdl.Color{R: 255, G: 255, B: 255, A: 255})
	if err != nil {
		return nil, err
	}
	defer surf.Free()
	conv, err := surf.ConvertFormat(sdl.PIXELFORMAT_ABGR8888, 0)
	if err != nil {
		return nil, err
	}
	defer conv.Free()

	w, h := int(conv.W), int(conv.H)
	mask := image.NewAlpha(image.Rect(0, 0, w, h))
	src := conv.Pixels()
	pitch := int(conv.Pitch)
	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			a := src[row*pitch+col*4+3]
			mask.SetAlpha(col, row, color.Alpha{A: a})
		}
	}

	c.textMu.Lock()
	if len(c.textCache) > 768 {
		c.textCache = make(map[string]*image.Alpha)
	}
	c.textCache[key] = mask
	c.textMu.Unlock()
	return mask, nil
}

func (c *Canvas) stampMask(mask *image.Alpha, x, y int, col color.RGBA) {
	b := mask.Bounds()
	dst := image.Rect(x, y, x+b.Dx(), y+b.Dy())
	imagedraw.DrawMask(c.pixels, dst, image.NewUniform(col), image.Point{}, mask, b.Min, imagedraw.Over)
}

// Text draws left-aligned text. y is the top of the glyph box (CSS-like).
func (c *Canvas) Text(s Style, text string, x, y int) {
	if text == "" {
		return
	}
	mask, err := c.textMask(s.Family, s.Size, text)
	if err != nil {
		return
	}
	if s.Shadow {
		black := color.RGBA{A: 255}
		// 1px outline in 8 directions approximating the CSS 1.5px outline
		for _, off := range [][2]int{{-1, -1}, {0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}} {
			c.stampMask(mask, x+off[0], y+off[1], black)
		}
		// 3px drop shadow
		c.stampMask(mask, x+3, y+3, black)
	}
	c.stampMask(mask, x, y, s.Color)
}

func (c *Canvas) MeasureText(s Style, text string) int {
	font, err := c.fonts.Get(s.Family, s.Size)
	if err != nil {
		return 0
	}
	w, _, err := font.SizeUTF8(text)
	if err != nil {
		return 0
	}
	return w
}

// TextRight draws text with its right edge at rightX.
func (c *Canvas) TextRight(s Style, text string, rightX, y int) {
	c.Text(s, text, rightX-c.MeasureText(s, text), y)
}

// TextCenter draws text centered on centerX.
func (c *Canvas) TextCenter(s Style, text string, centerX, y int) {
	c.Text(s, text, centerX-c.MeasureText(s, text)/2, y)
}

// TextCenterIn draws text centered within [x, x+w).
func (c *Canvas) TextCenterIn(s Style, text string, x, w, y int) {
	c.TextCenter(s, text, x+w/2, y)
}

// --- presentation ---

func (c *Canvas) ApplyScanlines() {
	for y := 1; y < LogicalHeight; y += 2 {
		row := c.pixels.Pix[y*c.pixels.Stride : y*c.pixels.Stride+LogicalWidth*4]
		for i := 0; i < len(row); i += 4 {
			row[i] = row[i] * 3 / 4
			row[i+1] = row[i+1] * 3 / 4
			row[i+2] = row[i+2] * 3 / 4
		}
	}
}

func (c *Canvas) PresentToRenderer() error {
	if c.texture == nil {
		return nil
	}
	return c.texture.Update(nil, unsafe.Pointer(&c.pixels.Pix[0]), c.pixels.Stride)
}

func (c *Canvas) SavePNG(path string) error {
	return savePNG(path, c.pixels)
}

func (c *Canvas) Close() {
	if c.texture != nil {
		c.texture.Destroy()
	}
}
