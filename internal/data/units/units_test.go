package units_test

import (
	"testing"

	"github.com/austinmcchord/ws4000/internal/data/units"
)

func TestTempConversion(t *testing.T) {
	c := units.New("us")
	v := 0.0
	got := c.TempC(&v)
	if got != "32" {
		t.Fatalf("expected 32F, got %s", got)
	}
}

func TestWindCalm(t *testing.T) {
	c := units.New("us")
	v := 0.0
	got := c.WindMS(&v)
	if got != "Calm" {
		t.Fatalf("expected Calm, got %s", got)
	}
}

func TestDirection(t *testing.T) {
	v := 0.0
	got := units.DirectionToNSEW(&v)
	if got != "N" {
		t.Fatalf("expected N, got %s", got)
	}
}
