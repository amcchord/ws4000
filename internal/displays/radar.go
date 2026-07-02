package displays

import (
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"io"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/amcchord/ws4000/internal/assets"
	"github.com/amcchord/ws4000/internal/config"
	"github.com/amcchord/ws4000/internal/data/weather"
	"github.com/amcchord/ws4000/internal/engine"
	"github.com/amcchord/ws4000/internal/render"
	xdraw "golang.org/x/image/draw"
)

// Radar constants ported from radar-constants.mjs (standard mode).
const (
	radarTileW     = 680
	radarTileH     = 387
	radarFullW     = 2550 // RADAR_FULL_SIZE
	radarFullH     = 1600
	radarFinalW    = 640 // RADAR_FINAL_SIZE standard
	radarFinalH    = 367
	radarSourceW   = 240 // RADAR_SOURCE_SIZE standard
	radarSourceH   = 163
	radarOffsetX   = 240 // RADAR_OFFSET standard
	radarOffsetY   = 138
	radarHost      = "mesonet.agron.iastate.edu"
	radarMainTop   = 113 // header is 83px + 30 padding for radar
	radarFrameMax  = 6
)

var radarFileRegex = regexp.MustCompile(`n0r_(\d{12})\.png`)

type radarFrame struct {
	img  *image.RGBA // 640x367 processed frame
	time time.Time
}

type RadarDisplay struct {
	*engine.BaseDisplay
	svc    *weather.Service
	cfg    config.Config
	frames []radarFrame
	tiles  *image.RGBA // composited basemap 640x367
	over   *image.RGBA // composited overlay 640x367
}

func NewRadar(svc *weather.Service, cfg config.Config) *RadarDisplay {
	d := &RadarDisplay{
		BaseDisplay: engine.NewBaseDisplay(11, "radar", "Local Radar", true),
		svc:         svc,
		cfg:         cfg,
	}
	d.SetOkToDrawTicker(false)
	// upstream frame sequence: 350ms ticks
	d.Timing().BaseDelayMS = 350
	d.Timing().Delay = []engine.ScreenDelay{
		{Time: 4, ScreenIndex: 5}, {Time: 1, ScreenIndex: 0}, {Time: 1, ScreenIndex: 1},
		{Time: 1, ScreenIndex: 2}, {Time: 1, ScreenIndex: 3}, {Time: 1, ScreenIndex: 4},
		{Time: 4, ScreenIndex: 5}, {Time: 1, ScreenIndex: 0}, {Time: 1, ScreenIndex: 1},
		{Time: 1, ScreenIndex: 2}, {Time: 1, ScreenIndex: 3}, {Time: 1, ScreenIndex: 4},
		{Time: 4, ScreenIndex: 5}, {Time: 1, ScreenIndex: 0}, {Time: 1, ScreenIndex: 1},
		{Time: 1, ScreenIndex: 2}, {Time: 1, ScreenIndex: 3}, {Time: 1, ScreenIndex: 4},
		{Time: 12, ScreenIndex: 5},
	}
	d.Timing().CalcNavTiming()
	return d
}

// map coordinates (radar-utils.mjs getXYFromLatitudeLongitudeMap, standard shift 0)
func radarMapXY(lat, lon float64) (float64, float64) {
	y := (-145.095*lat + 7377.117) - 27 - radarTileH/2
	y = math.Max(0, math.Min(float64(radarTileH)*11-radarTileH, y))
	x := (111.407*lon + 14220.972) + 4 - radarTileW/2
	x = math.Max(0, math.Min(float64(radarTileW)*10-radarTileW, x))
	return x, y
}

// doppler crop coordinates (getXYFromLatitudeLongitudeDoppler, standard offsets)
func radarDopplerXY(lat, lon float64) (float64, float64) {
	y := (51-lat)*61.4481 - radarOffsetY
	y = math.Max(0, math.Min(6000, y))
	x := -((-129.138-lon)*42.1768) - radarOffsetX
	x = math.Max(0, math.Min(2800, x))
	return x * 2, y * 2
}

func (d *RadarDisplay) Fetch(params *engine.WeatherParams) error {
	if !d.BeginFetch(params) {
		return nil
	}
	if params.State == "AK" || params.State == "HI" {
		d.SetStatus(engine.StatusNoData)
		d.Timing().TotalScreens = 0
		return nil
	}

	// 1. build the basemap and overlay composites from tiles
	srcX, srcY := radarMapXY(params.Latitude, params.Longitude)
	if err := d.buildTiles(srcX, srcY); err != nil {
		d.SetStatus(engine.StatusFailed)
		return nil
	}

	// 2. list available radar images (today, plus yesterday if needed)
	urls, err := d.listRadarURLs()
	if err != nil || len(urls) == 0 {
		d.SetStatus(engine.StatusFailed)
		return nil
	}

	// 3. fetch and process the most recent frames
	dopplerX, dopplerY := radarDopplerXY(params.Latitude, params.Longitude)
	loc := params.TZ()
	client := &http.Client{Timeout: 30 * time.Second}

	// build into a local slice; a single assignment below keeps the render
	// thread from seeing a partially-populated frame list during refreshes
	var frames []radarFrame
	for _, u := range urls {
		m := radarFileRegex.FindStringSubmatch(u)
		if m == nil {
			continue
		}
		ts, err := time.Parse("200601021504", m[1])
		if err != nil {
			continue
		}
		frame, err := fetchAndProcessRadar(client, u, dopplerX, dopplerY)
		if err != nil {
			continue
		}
		frames = append(frames, radarFrame{img: frame, time: ts.In(loc)})
	}
	if len(frames) == 0 {
		d.SetStatus(engine.StatusFailed)
		return nil
	}
	d.frames = frames
	d.Timing().CalcNavTiming()
	d.SetStatus(engine.StatusLoaded)
	return nil
}

// buildTiles composites the 640x367 viewport from map/overlay tiles
// (radar-tiles.mjs setTiles).
func (d *RadarDisplay) buildTiles(srcX, srcY float64) error {
	shiftX := int(srcX) % radarTileW
	shiftY := int(srcY) % radarTileH

	build := func(prefix string) (*image.RGBA, error) {
		out := image.NewRGBA(image.Rect(0, 0, radarFinalW, radarFinalH))
		for ty := 0; ty < 2; ty++ {
			for tx := 0; tx < 2; tx++ {
				tileX := int(srcX)/radarTileW + tx
				tileY := int(srcY)/radarTileH + ty
				if tileX < 0 || tileX > 10 || tileY < 0 || tileY > 11 {
					continue
				}
				path := fmt.Sprintf("maps/radar/%s-%d-%d.webp", prefix, tileY, tileX)
				data, err := assets.Read(path)
				if err != nil {
					continue
				}
				img, err := render.DecodeImage(data)
				if err != nil {
					continue
				}
				dstX := tx*radarTileW - shiftX
				dstY := ty*radarTileH - shiftY
				b := img.Bounds()
				imagedraw.Draw(out, image.Rect(dstX, dstY, dstX+b.Dx(), dstY+b.Dy()), img, b.Min, imagedraw.Over)
			}
		}
		return out, nil
	}

	var err error
	d.tiles, err = build("map")
	if err != nil {
		return err
	}
	d.over, err = build("overlay")
	return err
}

// listRadarURLs returns the most recent n0r composite URLs (oldest first).
func (d *RadarDisplay) listRadarURLs() ([]string, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	now := time.Now().UTC()

	list := func(day time.Time) ([]string, error) {
		dir := fmt.Sprintf("https://%s/archive/data/%s/GIS/uscomp/", radarHost, day.Format("2006/01/02"))
		req, err := http.NewRequest(http.MethodGet, dir+"?F=0&P=n0r*.png", nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var urls []string
		for _, m := range radarFileRegex.FindAllString(string(body), -1) {
			urls = append(urls, dir+m)
		}
		return urls, nil
	}

	todayURLs, err := list(now)
	if err != nil {
		todayURLs = nil
	}
	urls := todayURLs

	// fetch yesterday's list if today doesn't have enough frames
	if len(dedupe(urls)) < radarFrameMax {
		yesterdayURLs, err := list(now.Add(-24 * time.Hour))
		if err == nil {
			urls = append(yesterdayURLs, urls...)
		}
	}
	urls = dedupe(urls)

	sort.Slice(urls, func(i, j int) bool {
		return radarTimestamp(urls[i]) < radarTimestamp(urls[j])
	})
	if len(urls) > radarFrameMax {
		urls = urls[len(urls)-radarFrameMax:]
	}
	return urls, nil
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func radarTimestamp(url string) string {
	m := radarFileRegex.FindStringSubmatch(url)
	if m == nil {
		return ""
	}
	return m[1]
}

// fetchAndProcessRadar ports radar-processor.mjs: scale the national composite
// to 2550x1600, crop 240x163 at the doppler offset, remap the palette, and
// stretch to 640x367.
func fetchAndProcessRadar(client *http.Client, url string, dopplerX, dopplerY float64) (*image.RGBA, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("radar fetch %s: %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	src, err := render.DecodeImage(data)
	if err != nil {
		return nil, err
	}

	// crop coordinates are relative to the 2550x1600 scaled composite
	cropX := int(math.Round(dopplerX / 2))
	cropY := int(math.Round(dopplerY / 2))

	// map the crop region back to source image coordinates and scale directly
	sb := src.Bounds()
	srcX0 := cropX * sb.Dx() / radarFullW
	srcY0 := cropY * sb.Dy() / radarFullH
	srcX1 := (cropX + radarSourceW) * sb.Dx() / radarFullW
	srcY1 := (cropY + radarSourceH) * sb.Dy() / radarFullH

	cropped := image.NewRGBA(image.Rect(0, 0, radarSourceW, radarSourceH))
	xdraw.NearestNeighbor.Scale(cropped, cropped.Bounds(), src,
		image.Rect(sb.Min.X+srcX0, sb.Min.Y+srcY0, sb.Min.X+srcX1, sb.Min.Y+srcY1), xdraw.Src, nil)

	remapRadarColors(cropped)

	out := image.NewRGBA(image.Rect(0, 0, radarFinalW, radarFinalH))
	xdraw.NearestNeighbor.Scale(out, out.Bounds(), cropped, cropped.Bounds(), xdraw.Src, nil)
	return out, nil
}

// remapRadarColors ports removeDopplerRadarImageNoise exactly.
func remapRadarColors(img *image.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := rgbaAt(img, x, y)
			var nr, ng, nb, na uint8
			switch {
			case (r == 0 && g == 0 && bl == 0) ||
				(r == 0 && g == 236 && bl == 236) ||
				(r == 1 && g == 160 && bl == 246) ||
				(r == 0 && g == 0 && bl == 246):
				nr, ng, nb, na = 0, 0, 0, 0 // transparent (noise)
			case r == 0 && g == 255 && bl == 0:
				nr, ng, nb, na = 49, 210, 22, 255 // light green 1
			case r == 0 && g == 200 && bl == 0:
				nr, ng, nb, na = 0, 142, 0, 255 // light green 2
			case r == 0 && g == 144 && bl == 0:
				nr, ng, nb, na = 20, 90, 15, 255 // dark green 1
			case r == 255 && g == 255 && bl == 0:
				nr, ng, nb, na = 10, 40, 10, 255 // dark green 2
			case r == 231 && g == 192 && bl == 0:
				nr, ng, nb, na = 196, 179, 70, 255 // yellow
			case r == 255 && g == 144 && bl == 0:
				nr, ng, nb, na = 190, 72, 19, 255 // orange
			case (r == 214 && g == 0 && bl == 0) || (r == 255 && g == 0 && bl == 0):
				nr, ng, nb, na = 171, 14, 14, 255 // red
			case (r == 192 && g == 0 && bl == 0) || (r == 255 && g == 0 && bl == 255):
				nr, ng, nb, na = 115, 31, 4, 255 // brown
			default:
				continue
			}
			i := img.PixOffset(x, y)
			img.Pix[i] = nr
			img.Pix[i+1] = ng
			img.Pix[i+2] = nb
			img.Pix[i+3] = na
		}
	}
}

func rgbaAt(img *image.RGBA, x, y int) (uint8, uint8, uint8, uint8) {
	i := img.PixOffset(x, y)
	return img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]
}

// radar precip scale colors (from _radar.scss)
var radarScale = []color.RGBA{
	{R: 49, G: 210, B: 22, A: 255},
	{R: 28, G: 138, B: 18, A: 255},
	{R: 20, G: 90, B: 15, A: 255},
	{R: 10, G: 40, B: 10, A: 255},
	{R: 196, G: 179, B: 70, A: 255},
	{R: 190, G: 72, B: 19, A: 255},
	{R: 171, G: 14, B: 14, A: 255},
	{R: 115, G: 31, B: 4, A: 255},
}

func (d *RadarDisplay) Draw(c *render.Canvas, screenIndex int) error {
	_ = c.DrawBackground("backgrounds/4.png")

	// radar title: white Arial bold 38 at x=155 (per _radar.scss)
	titleStyle := render.Style{Family: render.FontArialBold, Size: 38, Color: colWhite, Shadow: true}
	c.Text(titleStyle, "Local", 155, 26)
	c.Text(titleStyle, "Radar", 155, 61)

	// precip scale in header right block (x=280..640 centered):
	// PRECIP  Light [8 boxes] Heavy
	scaleStyle := render.Style{Family: render.FontStar4000, Size: 24, Color: colWhite, Shadow: true}
	scaleW := c.MeasureText(scaleStyle, "PRECIP") + 10 + c.MeasureText(scaleStyle, "Light") + 6 + 8*17 + 6 + c.MeasureText(scaleStyle, "Heavy")
	sx := 280 + (360-scaleW)/2
	c.Text(scaleStyle, "PRECIP", sx, 32)
	sx += c.MeasureText(scaleStyle, "PRECIP") + 10
	c.Text(scaleStyle, "Light", sx, 32)
	sx += c.MeasureText(scaleStyle, "Light") + 6
	for _, col := range radarScale {
		c.FillRect(sx, 32, 17, 24, col)
		// black border
		c.FillRect(sx, 32, 17, 2, color.RGBA{A: 255})
		c.FillRect(sx, 54, 17, 2, color.RGBA{A: 255})
		c.FillRect(sx, 32, 2, 24, color.RGBA{A: 255})
		c.FillRect(sx+15, 32, 2, 24, color.RGBA{A: 255})
		sx += 17
	}
	sx += 6
	c.Text(scaleStyle, "Heavy", sx, 32)

	// frame time (Star4000 Small 32, centered in right block)
	if screenIndex >= 0 && screenIndex < len(d.frames) {
		timeStr := strings.ToUpper(d.frames[screenIndex].time.Format("3:04 PM"))
		timeStyle := render.Style{Family: render.FontStar4000Small, Size: 32, Color: colWhite, Shadow: true}
		c.TextCenter(timeStyle, timeStr, 460, 72)
	}

	// map stack: basemap, radar frame, overlay
	if d.tiles != nil {
		c.DrawGoImage(d.tiles, 0, radarMainTop)
	}
	if screenIndex >= 0 && screenIndex < len(d.frames) && d.frames[screenIndex].img != nil {
		c.DrawGoImage(d.frames[screenIndex].img, 0, radarMainTop)
	}
	if d.over != nil {
		c.DrawGoImage(d.over, 0, radarMainTop)
	}
	return nil
}
