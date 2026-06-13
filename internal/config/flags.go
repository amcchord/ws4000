package config

import (
	"flag"
	"fmt"
	"strings"
)

type Flags struct {
	Location     string
	Latitude     float64
	Longitude    float64
	ConfigPath   string
	Units        string
	Fullscreen   bool
	Scale        int
	Speed        float64
	Scanlines    bool
	Volume       float64
	MusicDir     string
	FixtureDir   string
	Screenshot   string
	Display      string
	ListDisplays bool
	Enable       []string
	Disable      []string
}

func ParseFlags() Flags {
	var f Flags
	flag.StringVar(&f.Location, "location", "", "Location query (geocoded)")
	flag.Float64Var(&f.Latitude, "lat", 0, "Latitude")
	flag.Float64Var(&f.Longitude, "lon", 0, "Longitude")
	flag.StringVar(&f.ConfigPath, "config", "", "Config file path")
	flag.StringVar(&f.Units, "units", "", "Units: us or metric")
	flag.BoolVar(&f.Fullscreen, "fullscreen", false, "Start fullscreen")
	flag.IntVar(&f.Scale, "scale", 0, "Integer scale factor (0=auto)")
	flag.Float64Var(&f.Speed, "speed", 0, "Playback speed multiplier")
	flag.BoolVar(&f.Scanlines, "scanlines", false, "Enable scanline overlay")
	flag.Float64Var(&f.Volume, "volume", -1, "Music volume 0.0-1.0")
	flag.StringVar(&f.MusicDir, "music-dir", "", "Directory of MP3 music files")
	flag.StringVar(&f.FixtureDir, "fixture", "", "Use API fixtures from directory")
	flag.StringVar(&f.Screenshot, "screenshot", "", "Save screenshot and exit")
	flag.StringVar(&f.Display, "display", "", "Display id to capture with --screenshot")
	flag.BoolVar(&f.ListDisplays, "list-displays", false, "List displays and exit")
	enable := flag.String("enable", "", "Enable display (repeat with comma)")
	disable := flag.String("disable", "", "Disable display (repeat with comma)")
	flag.Parse()
	if *enable != "" {
		f.Enable = strings.Split(*enable, ",")
	}
	if *disable != "" {
		f.Disable = strings.Split(*disable, ",")
	}
	return f
}

func Merge(base Config, flags Flags) Config {
	if flags.Location != "" {
		base.Location = flags.Location
	}
	if flags.Latitude != 0 {
		base.Latitude = flags.Latitude
	}
	if flags.Longitude != 0 {
		base.Longitude = flags.Longitude
	}
	if flags.Units != "" {
		base.Units = flags.Units
	}
	if flags.Fullscreen {
		base.Fullscreen = true
	}
	if flags.Scale > 0 {
		base.Scale = flags.Scale
	}
	if flags.Speed > 0 {
		base.Speed = flags.Speed
	}
	if flags.Scanlines {
		base.Scanlines = true
	}
	if flags.Volume >= 0 {
		base.Volume = flags.Volume
	}
	if flags.MusicDir != "" {
		base.MusicDir = flags.MusicDir
	}
	if flags.FixtureDir != "" {
		base.FixtureDir = flags.FixtureDir
	}
	for _, id := range flags.Enable {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if base.Displays == nil {
			base.Displays = make(map[string]bool)
		}
		base.Displays[id] = true
	}
	for _, id := range flags.Disable {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if base.Displays == nil {
			base.Displays = make(map[string]bool)
		}
		base.Displays[id] = false
	}
	if base.Units == "si" {
		base.Units = "metric"
	}
	return base
}

func PrintDisplays() {
	fmt.Println("Available displays:")
	for _, name := range DisplayNames() {
		fmt.Printf("  - %s\n", name)
	}
}
