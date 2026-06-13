// Package icons ports upstream icons-parse.mjs / icons-large.mjs / icons-small.mjs.
package icons

import (
	"regexp"
	"strconv"
	"strings"
)

var iconURLPattern = regexp.MustCompile(`(?i)/icons/(\w+)/(day|night)/([^?]+)`)

type parsed struct {
	condition   string
	probability int
	night       bool
}

func parseIconURL(iconURL string) (parsed, bool) {
	m := iconURLPattern.FindStringSubmatch(iconURL)
	if m == nil {
		return parsed{}, false
	}
	condition := m[3]
	night := strings.EqualFold(m[2], "night")

	// dual conditions: "rain_showers,30/tsra_hi,50" - use the second when different
	prob := 100
	if strings.Contains(condition, "/") {
		parts := strings.SplitN(condition, "/", 2)
		first, firstProb := splitProb(parts[0])
		second, secondProb := splitProb(parts[1])
		if second != first {
			condition = second
			prob = secondProb
		} else {
			condition = first
			prob = firstProb
		}
	} else {
		condition, prob = splitProb(condition)
	}
	return parsed{condition: condition, probability: prob, night: night}, true
}

func splitProb(s string) (string, int) {
	parts := strings.SplitN(s, ",", 2)
	prob := 100
	if len(parts) == 2 {
		if v, err := strconv.Atoi(parts[1]); err == nil && v > 0 {
			prob = v
		}
	}
	return parts[0], prob
}

// LargeIcon maps a weather.gov icon URL to a current-conditions icon asset path.
func LargeIcon(iconURL string) string {
	p, ok := parseIconURL(iconURL)
	if !ok {
		return "icons/current-conditions/No-Data.gif"
	}
	key := p.condition
	if p.night {
		key += "-n"
	}

	pick := func(name string) string { return "icons/current-conditions/" + name }

	switch key {
	case "skc", "hot", "haze", "cold":
		return pick("Sunny.gif")
	case "skc-n", "haze-n", "cold-n":
		return pick("Clear.gif")
	case "dust", "dust-n", "smoke", "smoke-n":
		return pick("Smoke.gif")
	case "few", "sct", "bkn":
		return pick("Partly-Cloudy.gif")
	case "few-n", "sct-n", "bkn-n":
		return pick("Mostly-Clear.gif")
	case "ovc", "ovc-n":
		return pick("Cloudy.gif")
	case "fog", "fog-n":
		return pick("Fog.gif")
	case "rain_sleet", "rain_sleet-n":
		return pick("Rain-Sleet.gif")
	case "sleet", "sleet-n":
		return pick("Sleet.gif")
	case "rain_showers", "rain_showers_hi", "rain_showers_high", "rain_showers-n", "rain_showers_hi-n", "rain_showers_high-n":
		return pick("Shower.gif")
	case "rain", "rain-n":
		return pick("Rain.gif")
	case "snow", "snow-n":
		if p.probability > 50 {
			return pick("Heavy-Snow.gif")
		}
		return pick("Light-Snow.gif")
	case "rain_snow", "rain_snow-n":
		return pick("Rain-Snow.gif")
	case "snow_fzra", "snow_fzra-n", "winter_mix", "winter_mix-n":
		return pick("Freezing-Rain-Snow.gif")
	case "fzra", "fzra-n", "rain_fzra", "rain_fzra-n":
		return pick("Freezing-Rain.gif")
	case "snow_sleet", "snow_sleet-n":
		return pick("Snow-Sleet.gif")
	case "tsra_sct", "tsra":
		return pick("Scattered-Thunderstorms-Day.gif")
	case "tsra_sct-n", "tsra-n":
		return pick("Scattered-Thunderstorms-Night.gif")
	case "tsra_hi", "tsra_hi-n", "tornado", "tornado-n", "hurricane", "hurricane-n", "tropical_storm", "tropical_storm-n":
		return pick("Thunderstorm.gif")
	case "wind_skc", "wind_", "wind_-n", "wind_skc-n", "wind_few", "wind_few-n", "wind_sct", "wind_sct-n", "wind_bkn", "wind_bkn-n", "wind_ovc", "wind_ovc-n":
		return pick("Windy.gif")
	case "blizzard", "blizzard-n":
		return pick("Blowing-Snow.gif")
	default:
		return pick("No-Data.gif")
	}
}

// SmallIcon maps a weather.gov icon URL to a regional-maps icon asset path
// (ported from upstream icons-small.mjs).
func SmallIcon(iconURL string, night bool) string {
	p, ok := parseIconURL(iconURL)
	if !ok {
		return ""
	}
	if night {
		p.night = true
	}
	key := p.condition
	if p.night {
		key += "-n"
	}

	pick := func(name string) string { return "icons/regional-maps/" + name }

	switch key {
	case "skc", "hot", "cold":
		return pick("Sunny.gif")
	case "skc-n", "cold-n", "few-n":
		return pick("Clear-1992.gif")
	case "few", "sct":
		return pick("Partly-Cloudy.gif")
	case "sct-n", "bkn-n":
		return pick("Partly-Cloudy-Night.gif")
	case "bkn":
		return pick("Mostly-Cloudy-1994.gif")
	case "ovc", "ovc-n":
		return pick("Cloudy.gif")
	case "fog", "fog-n":
		return pick("Fog.gif")
	case "haze", "haze-n":
		return pick("Haze.gif")
	case "smoke", "smoke-n", "dust", "dust-n":
		return pick("Smoke.gif")
	case "sleet", "sleet-n":
		return pick("Sleet.gif")
	case "rain_sleet", "rain_sleet-n":
		return pick("Rain-Sleet.gif")
	case "rain_showers", "rain_showers_hi", "rain_showers-n", "rain_showers_hi-n":
		if p.night {
			return pick("Scattered-Showers-Night-1994.gif")
		}
		return pick("Scattered-Showers-1994.gif")
	case "rain", "rain-n":
		return pick("Rain-1992.gif")
	case "snow", "snow-n":
		if p.probability > 50 {
			return pick("Heavy-Snow-1994.gif")
		}
		return pick("Light-Snow.gif")
	case "rain_snow", "rain_snow-n":
		return pick("Rain-Snow-1992.gif")
	case "snow_fzra", "snow_fzra-n", "winter_mix", "winter_mix-n":
		return pick("Freezing-Rain-Snow-1994.gif")
	case "fzra", "fzra-n", "rain_fzra", "rain_fzra-n":
		return pick("Freezing-Rain-1992.gif")
	case "snow_sleet", "snow_sleet-n":
		return pick("Snow-Sleet.gif")
	case "tsra_sct", "tsra":
		return pick("Scattered-Tstorms-1994.gif")
	case "tsra_sct-n", "tsra-n":
		return pick("Scattered-Tstorms-Night-1994.gif")
	case "tsra_hi", "tsra_hi-n", "hurricane", "hurricane-n", "tropical_storm", "tropical_storm-n", "tornado", "tornado-n":
		return pick("Thunderstorm.gif")
	case "wind_skc", "wind_few", "wind_-n", "wind_skc-n", "wind_few-n":
		return pick("Sunny-Wind-1994.gif")
	case "wind_sct", "wind_bkn", "wind_ovc", "wind_sct-n", "wind_bkn-n", "wind_ovc-n":
		return pick("Cloudy-Wind.gif")
	case "blizzard", "blizzard-n":
		return pick("Blowing-Snow.gif")
	default:
		return ""
	}
}
