package units_test

import (
	"testing"

	"github.com/amcchord/ws4000/internal/data/units"
)

func f(v float64) *float64 { return &v }

func TestTempConversion(t *testing.T) {
	c := units.New("us")
	if got := c.TempC(f(0)); got != "32" {
		t.Fatalf("expected 32, got %s", got)
	}
	if got := c.TempC(f(26.1)); got != "79" {
		t.Fatalf("expected 79, got %s", got)
	}
}

func TestWindKMH(t *testing.T) {
	c := units.New("us")
	if got := c.WindKMH(f(0)); got != "Calm" {
		t.Fatalf("expected Calm, got %s", got)
	}
	// 16 km/h ~ 10 mph
	if got := c.WindKMH(f(16.1)); got != "10" {
		t.Fatalf("expected 10, got %s", got)
	}
}

func TestVisibility(t *testing.T) {
	c := units.New("us")
	// 16093 m = 10 miles
	if got := c.VisibilityM(f(16093.4)); got != "10" {
		t.Fatalf("expected 10, got %s", got)
	}
}

func TestCeiling(t *testing.T) {
	c := units.New("us")
	if got := c.CeilingM(f(0)); got != "Unlimited" {
		t.Fatalf("expected Unlimited, got %s", got)
	}
	// 1402m = 4600.7ft -> 4600
	if got := c.CeilingM(f(1402)); got != "4600" {
		t.Fatalf("expected 4600, got %s", got)
	}
}

func TestPressure(t *testing.T) {
	c := units.New("us")
	// 101660 Pa ~ 30.02 inHg
	if got := c.PressurePa(f(101660)); got != "30.02" {
		t.Fatalf("expected 30.02, got %s", got)
	}
}

func TestDirection(t *testing.T) {
	if got := units.DirectionToNSEW(f(0)); got != "N" {
		t.Fatalf("expected N, got %s", got)
	}
	if got := units.DirectionToNSEW(f(225)); got != "SW" {
		t.Fatalf("expected SW, got %s", got)
	}
}
