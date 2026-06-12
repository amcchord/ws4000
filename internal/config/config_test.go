package config_test

import (
	"testing"

	"github.com/austinmcchord/ws4000/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.Default()
	if cfg.Units != "us" {
		t.Fatalf("expected us units, got %s", cfg.Units)
	}
	if !cfg.Displays["current-weather"] {
		t.Fatal("current-weather should be enabled by default")
	}
}

func TestValidateUnits(t *testing.T) {
	cfg := config.Default()
	cfg.Units = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestDisplayNames(t *testing.T) {
	names := config.DisplayNames()
	if len(names) < 12 {
		t.Fatalf("expected at least 12 displays, got %d", len(names))
	}
}
