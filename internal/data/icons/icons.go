package icons

import (
	"path/filepath"
	"strings"
)

func LargeIcon(iconURL string) string {
	name := iconName(iconURL)
	mapping := map[string]string{
		"skc":        "Sunny.gif",
		"few":        "Mostly-Clear.gif",
		"sct":        "Partly-Cloudy.gif",
		"bkn":        "Mostly-Cloudy.gif",
		"ovc":        "Cloudy.gif",
		"wind_skc":   "Windy.gif",
		"wind_few":   "Windy.gif",
		"wind_sct":   "Windy.gif",
		"wind_bkn":   "Windy.gif",
		"wind_ovc":   "Windy.gif",
		"snow":       "Light-Snow.gif",
		"rain_snow":  "Rain-Snow.gif",
		"rain_sleet": "Rain-Sleet.gif",
		"sleet":      "Sleet.gif",
		"fzra":       "Freezing-Rain.gif",
		"rain_fzra":  "Freezing-Rain.gif",
		"rain":       "Rain.gif",
		"rain_showers": "Shower.gif",
		"tsra":       "Thunderstorm.gif",
		"tsra_sct":   "Scattered-Thunderstorms-Day.gif",
		"tsra_sct_n": "Scattered-Thunderstorms-Night.gif",
		"fog":        "Fog.gif",
		"smoke":      "Smoke.gif",
		"blizzard":   "Blowing-Snow.gif",
		"hot":        "Sunny.gif",
		"cold":       "Clear.gif",
	}
	file, ok := mapping[name]
	if !ok {
		file = "No-Data.gif"
	}
	return filepath.ToSlash(filepath.Join("icons", "current-conditions", file))
}

func iconName(iconURL string) string {
	parts := strings.Split(iconURL, "/")
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	last = strings.TrimSuffix(last, ".png")
	last = strings.TrimSuffix(last, ",day")
	last = strings.TrimSuffix(last, ",night")
	return last
}
