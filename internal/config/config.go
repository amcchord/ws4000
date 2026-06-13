package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const DefaultConfigPath = ".config/ws4000/config.toml"

type Config struct {
	Location     string             `toml:"location"`
	Latitude     float64            `toml:"latitude"`
	Longitude    float64            `toml:"longitude"`
	Units        string             `toml:"units"`
	Fullscreen   bool               `toml:"fullscreen"`
	Scale        int                `toml:"scale"`
	Speed        float64            `toml:"speed"`
	Scanlines    bool               `toml:"scanlines"`
	Volume       float64            `toml:"volume"`
	MusicDir     string             `toml:"music_dir"`
	CustomScroll string             `toml:"custom_scroll"`
	FixtureDir   string             `toml:"fixture_dir"`
	Displays     map[string]bool    `toml:"displays"`
	RefreshMS    int                `toml:"refresh_ms"`
}

func Default() Config {
	return Config{
		Location:  "auto",
		Units:     "us",
		Scale:     2,
		Speed:     1.0,
		Volume:    0.0,
		RefreshMS: 600000,
		Displays: map[string]bool{
			"hazards":             true,
			"current-weather":     true,
			"latest-observations": true,
			"hourly":              false,
			"hourly-graph":        true,
			"travel":              false,
			"regional-forecast":   true,
			"local-forecast":      true,
			"extended-forecast":   true,
			"almanac":             false,
			"spc-outlook":         true,
			"radar":               true,
		},
	}
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DefaultConfigPath), nil
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return cfg, err
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Units != "" && c.Units != "us" && c.Units != "metric" && c.Units != "si" {
		return fmt.Errorf("units must be us or metric")
	}
	if c.Latitude != 0 || c.Longitude != 0 {
		if c.Latitude < -90 || c.Latitude > 90 {
			return fmt.Errorf("latitude out of range")
		}
		if c.Longitude < -180 || c.Longitude > 180 {
			return fmt.Errorf("longitude out of range")
		}
	}
	return nil
}

func DisplayNames() []string {
	return []string{
		"hazards",
		"current-weather",
		"latest-observations",
		"hourly",
		"hourly-graph",
		"travel",
		"regional-forecast",
		"local-forecast",
		"extended-forecast",
		"almanac",
		"spc-outlook",
		"radar",
	}
}
