// Package units ports upstream utils/units.mjs. The weather.gov API returns
// SI values (celsius, km/h, meters, pascals); converters produce the same
// strings as upstream for us or metric display.
package units

import (
	"fmt"
	"math"
)

type Converter struct {
	Units string // "us" or "metric"
}

func New(units string) Converter {
	if units == "si" {
		units = "metric"
	}
	if units != "metric" {
		units = "us"
	}
	return Converter{Units: units}
}

func (c Converter) us() bool { return c.Units == "us" }

// TempC converts celsius input. us: round((c*9/5)+32); metric: round(c).
func (c Converter) TempC(v *float64) string {
	if v == nil {
		return "-"
	}
	if c.us() {
		return fmt.Sprintf("%d", int(math.Round(*v*9/5+32)))
	}
	return fmt.Sprintf("%d", int(math.Round(*v)))
}

func (c Converter) TempUnit() string {
	if c.us() {
		return "F"
	}
	return "C"
}

// WindKMH converts km/h input (upstream kphToMph). Returns "Calm" for 0.
func (c Converter) WindKMH(v *float64) string {
	if v == nil {
		return "-"
	}
	var speed int
	if c.us() {
		speed = int(math.Round(*v / 1.60934))
	} else {
		speed = int(math.Round(*v))
	}
	if speed == 0 {
		return "Calm"
	}
	return fmt.Sprintf("%d", speed)
}

func (c Converter) WindUnit() string {
	if c.us() {
		return "MPH"
	}
	return "kph"
}

// PressurePa converts pascals. us: inHg with 2 decimals (truncated); metric: mbar.
func (c Converter) PressurePa(v *float64) string {
	if v == nil {
		return "-"
	}
	if c.us() {
		inhg := math.Trunc(*v*0.0002953*100) / 100
		return fmt.Sprintf("%.2f", inhg)
	}
	return fmt.Sprintf("%d", int(math.Round(*v/100)))
}

func (c Converter) PressureUnit() string {
	if c.us() {
		return " in.hg"
	}
	return " mbar"
}

// VisibilityM converts meters: us → miles (rounded); metric → km.
// Unit strings include the upstream leading space.
func (c Converter) VisibilityM(v *float64) string {
	if v == nil {
		return "-"
	}
	if c.us() {
		miles := math.Round(math.Round(*v/1.60934) / 1000)
		return fmt.Sprintf("%d", int(miles))
	}
	return fmt.Sprintf("%d", int(math.Round(*v/1000)))
}

func (c Converter) VisibilityUnit() string {
	if c.us() {
		return " mi."
	}
	return " km."
}

// CeilingM converts meters: us → feet rounded to nearest 100; metric → m.
// Returns "Unlimited" for 0/nil like upstream.
func (c Converter) CeilingM(v *float64) string {
	if v == nil || *v == 0 {
		return "Unlimited"
	}
	if c.us() {
		feet := math.Round(math.Round(*v/0.3048)/100) * 100
		return fmt.Sprintf("%d", int(feet))
	}
	return fmt.Sprintf("%d", int(math.Round(*v)))
}

func (c Converter) CeilingUnit() string {
	if c.us() {
		return "ft."
	}
	return "m."
}

// CToF converts a celsius value to display units as float (for charts).
func (c Converter) CToF(v float64) float64 {
	if c.us() {
		return v*9/5 + 32
	}
	return v
}

func DirectionToNSEW(deg *float64) string {
	if deg == nil {
		return "-"
	}
	dirs := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	idx := int(math.Round(*deg/22.5)) % 16
	if idx < 0 {
		idx += 16
	}
	return dirs[idx]
}
