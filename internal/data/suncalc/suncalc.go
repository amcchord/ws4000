// Package suncalc ports the parts of the suncalc JS library used upstream:
// sunrise/sunset times and moon phase calculations.
package suncalc

import (
	"math"
	"time"
)

const (
	rad    = math.Pi / 180.0
	dayMS  = 1000 * 60 * 60 * 24
	j1970  = 2440588.0
	j2000  = 2451545.0
	obliq  = rad * 23.4397 // obliquity of the Earth
	j0Coef = 0.0009
)

func toJulian(t time.Time) float64 {
	return float64(t.UnixMilli())/dayMS - 0.5 + j1970
}

func fromJulian(j float64, loc *time.Location) time.Time {
	ms := (j + 0.5 - j1970) * dayMS
	return time.UnixMilli(int64(ms)).In(loc)
}

func toDays(t time.Time) float64 {
	return toJulian(t) - j2000
}

func solarMeanAnomaly(d float64) float64 {
	return rad * (357.5291 + 0.98560028*d)
}

func eclipticLongitude(m float64) float64 {
	c := rad * (1.9148*math.Sin(m) + 0.02*math.Sin(2*m) + 0.0003*math.Sin(3*m))
	p := rad * 102.9372
	return m + c + p + math.Pi
}

func declination(l, b float64) float64 {
	return math.Asin(math.Sin(b)*math.Cos(obliq) + math.Cos(b)*math.Sin(obliq)*math.Sin(l))
}

func julianCycle(d, lw float64) float64 {
	return math.Round(d - j0Coef - lw/(2*math.Pi))
}

func approxTransit(ht, lw, n float64) float64 {
	return j0Coef + (ht+lw)/(2*math.Pi) + n
}

func solarTransitJ(ds, m, l float64) float64 {
	return j2000 + ds + 0.0053*math.Sin(m) - 0.0069*math.Sin(2*l)
}

func hourAngle(h, phi, d float64) float64 {
	return math.Acos((math.Sin(h) - math.Sin(phi)*math.Sin(d)) / (math.Cos(phi) * math.Cos(d)))
}

// Times holds the sun event times for a day.
type Times struct {
	Sunrise time.Time
	Sunset  time.Time
	Valid   bool
}

// GetTimes computes sunrise/sunset for the given date and position (suncalc port).
func GetTimes(lat, lon float64, when time.Time, loc *time.Location) Times {
	lw := rad * -lon
	phi := rad * lat

	d := toDays(when)
	n := julianCycle(d, lw)
	ds := approxTransit(0, lw, n)

	m := solarMeanAnomaly(ds)
	l := eclipticLongitude(m)
	dec := declination(l, 0)

	jnoon := solarTransitJ(ds, m, l)

	// standard sunrise/sunset altitude -0.833 degrees
	h0 := -0.833 * rad
	w := hourAngle(h0, phi, dec)
	if math.IsNaN(w) {
		return Times{Valid: false}
	}
	a := approxTransit(w, lw, n)
	jset := solarTransitJ(a, m, l)
	jrise := jnoon - (jset - jnoon)

	return Times{
		Sunrise: fromJulian(jrise, loc),
		Sunset:  fromJulian(jset, loc),
		Valid:   true,
	}
}

// MoonPhase returns the moon phase 0..1 (0=new, 0.25=first quarter,
// 0.5=full, 0.75=last quarter), anchored to the known new moon of
// 2000-01-06 18:14 UTC with the mean synodic month.
func MoonPhase(t time.Time) float64 {
	const synodicMonth = 29.530588853
	anchor := time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC)
	days := t.Sub(anchor).Hours() / 24
	phase := math.Mod(days/synodicMonth, 1)
	if phase < 0 {
		phase += 1
	}
	return phase
}

// PhaseEvent is an upcoming moon phase transition.
type PhaseEvent struct {
	Name string // "First", "Full", "Last", "New"
	Icon string
	Date time.Time
}

var phaseIcons = map[string]string{
	"First": "icons/moon-phases/First-Quarter.gif",
	"Full":  "icons/moon-phases/Full-Moon.gif",
	"Last":  "icons/moon-phases/Last-Quarter.gif",
	"New":   "icons/moon-phases/New-Moon.gif",
}

// NextMoonPhases scans forward day-by-day for the next 4 quarter-phase events
// (mirrors upstream almanac.mjs brute-force scan).
func NextMoonPhases(start time.Time, count int) []PhaseEvent {
	var events []PhaseEvent
	prev := MoonPhase(start)
	t := start
	for i := 0; i < 45 && len(events) < count; i++ {
		t = t.Add(24 * time.Hour)
		cur := MoonPhase(t)
		switch {
		case prev < 0.25 && cur >= 0.25:
			events = append(events, PhaseEvent{Name: "First", Icon: phaseIcons["First"], Date: refine(t, 0.25)})
		case prev < 0.50 && cur >= 0.50:
			events = append(events, PhaseEvent{Name: "Full", Icon: phaseIcons["Full"], Date: refine(t, 0.50)})
		case prev < 0.75 && cur >= 0.75:
			events = append(events, PhaseEvent{Name: "Last", Icon: phaseIcons["Last"], Date: refine(t, 0.75)})
		case prev > cur:
			events = append(events, PhaseEvent{Name: "New", Icon: phaseIcons["New"], Date: refine(t, 0.0)})
		}
		prev = cur
	}
	return events
}

// refine narrows the crossing to within an hour.
func refine(dayEnd time.Time, threshold float64) time.Time {
	t := dayEnd.Add(-24 * time.Hour)
	prev := MoonPhase(t)
	for i := 0; i < 24; i++ {
		next := t.Add(time.Hour)
		cur := MoonPhase(next)
		crossed := false
		if threshold == 0.0 {
			crossed = prev > cur
		} else {
			crossed = prev < threshold && cur >= threshold
		}
		if crossed {
			return next
		}
		prev = cur
		t = next
	}
	return dayEnd
}

// CurrentPhaseIcon returns the icon for the current moon phase (for completeness).
func CurrentPhaseIcon(t time.Time) string {
	phase := MoonPhase(t)
	switch {
	case phase < 0.0625 || phase >= 0.9375:
		return phaseIcons["New"]
	case phase < 0.3125:
		return phaseIcons["First"]
	case phase < 0.5625:
		return phaseIcons["Full"]
	default:
		return phaseIcons["Last"]
	}
}
