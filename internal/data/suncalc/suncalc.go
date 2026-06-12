package suncalc

import (
	"math"
	"time"
)

const rad = math.Pi / 180.0

type Times struct {
	Sunrise time.Time
	Sunset  time.Time
}

type Moon struct {
	Phase      float64
	PhaseName  string
	Icon       string
}

func GetTimes(lat, lon float64, when time.Time, loc *time.Location) Times {
	jd := toJulian(when)
	n := jd - 2451545.0 + 0.0008
	meanAnomaly := (357.5291 + 0.98560028*n) * rad
	center := (280.459 + 0.98564736*n) * rad
	ecliptic := center + (1.915*math.Sin(meanAnomaly)+0.020*math.Sin(2*meanAnomaly))*rad
	declination := math.Asin(math.Sin(ecliptic) * math.Sin(23.439*rad))

	hourAngle := math.Acos((math.Sin(-0.833*rad) - math.Sin(lat*rad)*math.Sin(declination)) / (math.Cos(lat*rad) * math.Cos(declination)))
	solarNoon := 2451545.0 + n + (lon/360.0)
	sunriseJD := solarNoon - hourAngle/(2*math.Pi)
	sunsetJD := solarNoon + hourAngle/(2*math.Pi)

	return Times{
		Sunrise: fromJulian(sunriseJD, loc),
		Sunset:  fromJulian(sunsetJD, loc),
	}
}

func GetMoon(when time.Time) Moon {
	jd := toJulian(when)
	days := jd - 2451545.0
	phase := math.Mod(days/29.530588853, 1.0)
	if phase < 0 {
		phase += 1
	}
	name, icon := moonPhaseName(phase)
	return Moon{Phase: phase, PhaseName: name, Icon: icon}
}

func moonPhaseName(phase float64) (string, string) {
	switch {
	case phase < 0.03 || phase > 0.97:
		return "New Moon", "icons/moon-phases/New-Moon.gif"
	case phase < 0.22:
		return "First Quarter", "icons/moon-phases/First-Quarter.gif"
	case phase < 0.28:
		return "First Quarter", "icons/moon-phases/First-Quarter.gif"
	case phase < 0.47:
		return "Full Moon", "icons/moon-phases/Full-Moon.gif"
	case phase < 0.53:
		return "Full Moon", "icons/moon-phases/Full-Moon-Degraded.gif"
	case phase < 0.72:
		return "Last Quarter", "icons/moon-phases/Last-Quarter.gif"
	default:
		return "Last Quarter", "icons/moon-phases/Last-Quarter.gif"
	}
}

func toJulian(t time.Time) float64 {
	y, m, d := t.Date()
	if m <= 2 {
		y--
		m += 12
	}
	a := y / 100
	b := 2 - a + a/4
	return float64(int(365.25*(float64(y)+4716))) + float64(int(30.6001*float64(m+1))) + float64(d) + float64(t.Hour())/24.0 + float64(t.Minute())/1440.0 + float64(t.Second())/86400.0 + float64(b) - 1524.5
}

func fromJulian(jd float64, loc *time.Location) time.Time {
	z := int(jd + 0.5)
	f := jd + 0.5 - float64(z)
	if z < 2299161 {
		a := z
		_ = a
	} else {
		alpha := int((float64(z) - 1867216.25) / 36524.25)
		z = z + 1 + alpha - alpha/4
	}
	b := z + 1524
	c := int((float64(b) - 122.1) / 365.25)
	d := int(365.25 * float64(c))
	e := int(float64(b-d) / 30.6001)
	day := b - d - int(30.6001*float64(e))
	month := e - 1
	if month > 12 {
		month -= 12
	}
	year := c - 4715
	if month > 2 {
		year--
	}
	hours := f * 24.0
	hour := int(hours)
	minutes := int((hours - float64(hour)) * 60.0)
	seconds := int(((hours-float64(hour))*60.0 - float64(minutes)) * 60.0)
	return time.Date(year, time.Month(month), day, hour, minutes, seconds, 0, loc)
}
