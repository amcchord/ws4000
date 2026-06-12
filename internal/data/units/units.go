package units

import (
	"fmt"
	"math"
)

type Converter struct {
	Units string
}

func New(units string) Converter {
	if units == "si" {
		units = "metric"
	}
	return Converter{Units: units}
}

func (c Converter) TempC(v *float64) string {
	if v == nil {
		return "-"
	}
	if c.Units == "metric" {
		return fmt.Sprintf("%d", int(math.Round(*v)))
	}
	return fmt.Sprintf("%d", int(math.Round(*v*9.0/5.0+32.0)))
}

func (c Converter) TempSymbol() string {
	return string(rune(176))
}

func (c Converter) WindMS(v *float64) string {
	if v == nil {
		return "-"
	}
	if *v == 0 {
		return "Calm"
	}
	if c.Units == "metric" {
		return fmt.Sprintf("%d", int(math.Round(*v*3.6)))
	}
	return fmt.Sprintf("%d", int(math.Round(*v*2.236936)))
}

func (c Converter) WindUnit() string {
	if c.Units == "metric" {
		return "KPH"
	}
	return "MPH"
}

func (c Converter) PressurePa(v *float64) string {
	if v == nil {
		return "-"
	}
	if c.Units == "metric" {
		return fmt.Sprintf("%.0f", *v/100.0)
	}
	return fmt.Sprintf("%.2f", *v*0.0002952998)
}

func (c Converter) PressureUnit() string {
	if c.Units == "metric" {
		return "MB"
	}
	return "IN"
}

func (c Converter) VisibilityM(v *float64) string {
	if v == nil {
		return "-"
	}
	if c.Units == "metric" {
		return fmt.Sprintf("%.1f", *v/1000.0)
	}
	return fmt.Sprintf("%.1f", *v*0.000621371)
}

func (c Converter) VisibilityUnit() string {
	if c.Units == "metric" {
		return "KM"
	}
	return "MI"
}

func (c Converter) CeilingM(v *float64) string {
	if v == nil || *v == 0 {
		return "Unlimited"
	}
	if c.Units == "metric" {
		return fmt.Sprintf("%d", int(math.Round(*v/100.0)))
	}
	return fmt.Sprintf("%d", int(math.Round(*v*3.28084)))
}

func (c Converter) CeilingUnit() string {
	if c.Units == "metric" {
		return "M"
	}
	return "FT"
}

func DirectionToNSEW(deg *float64) string {
	if deg == nil {
		return "-"
	}
	dirs := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	idx := int(math.Round(*deg/22.5)) % 16
	return dirs[idx]
}
