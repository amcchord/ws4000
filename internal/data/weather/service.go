package weather

import (
	"fmt"
	"os"
	"strings"

	"github.com/amcchord/ws4000/internal/data/geo"
	"github.com/amcchord/ws4000/internal/data/nws"
	"github.com/amcchord/ws4000/internal/engine"
)

type Service struct {
	Client *nws.Client
}

func NewService(cacheDir, fixtureDir string) *Service {
	return &Service{Client: nws.NewClient(cacheDir, fixtureDir)}
}

// ResolveLocation determines the forecast location. Priority:
//  1. explicit latitude/longitude
//  2. a location query string (geocoded), unless set to "auto"
//  3. geo-IP lookup of the machine's public IP
//  4. a built-in fallback so the app always starts
func (s *Service) ResolveLocation(location string, lat, lon float64) (float64, float64, string, error) {
	if lat != 0 || lon != 0 {
		return lat, lon, "", nil
	}
	if location != "" && !strings.EqualFold(location, "auto") {
		result, err := geo.Geocode(location)
		if err != nil {
			return 0, 0, "", fmt.Errorf("could not geocode %q: %w", location, err)
		}
		return result.Latitude, result.Longitude, result.Name, nil
	}

	if ip, err := geo.LocateByIP(); err == nil {
		name := strings.TrimSpace(ip.City + ", " + ip.Region)
		fmt.Fprintf(os.Stderr, "ws4000: using geo-IP location %s (%.4f, %.4f)\n", name, ip.Latitude, ip.Longitude)
		return ip.Latitude, ip.Longitude, name, nil
	}

	// last resort so the display still comes up
	fmt.Fprintln(os.Stderr, "ws4000: geo-IP lookup failed, defaulting to Orlando FL (set location in config)")
	return 28.431, -81.3076, "Orlando, FL", nil
}

func (s *Service) BuildParams(lat, lon float64, units string) (*engine.WeatherParams, error) {
	point, err := s.Client.GetPoint(lat, lon)
	if err != nil {
		return nil, err
	}

	stations, err := s.Client.GetStations(point.Properties.ObservationStations)
	if err != nil {
		return nil, err
	}
	filtered := nws.FilterStations(stations.Features)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no usable observation stations found for %.4f,%.4f", lat, lon)
	}

	zoneID := nws.ZoneFromURL(point.Properties.ForecastZone)
	radarID := nws.RadarFromURL(point.Properties.RadarStation)

	var stationURLs []string
	for _, st := range filtered {
		stationURLs = append(stationURLs, st.ID)
	}

	return &engine.WeatherParams{
		Latitude:        lat,
		Longitude:       lon,
		City:            point.Properties.RelativeLocation.Properties.City,
		State:           point.Properties.RelativeLocation.Properties.State,
		ZoneID:          zoneID,
		RadarID:         radarID,
		StationID:       filtered[0].Properties.StationIdentifier,
		WeatherOffice:   point.Properties.CWA,
		TimeZone:        point.Properties.TimeZone,
		ForecastURL:     point.Properties.Forecast,
		ForecastGridURL: point.Properties.ForecastGridData,
		ObservationURL:  filtered[0].ID,
		Units:           units,
		StationURLs:     stationURLs,
	}, nil
}

func CleanLocation(name string) string {
	name = strings.TrimSpace(name)
	if idx := strings.Index(name, "/"); idx >= 0 {
		name = name[:idx]
	}
	return strings.TrimSpace(name)
}

func ShortCondition(condition string) string {
	replacements := []struct{ old, new string }{
		{"Light", "L"}, {"Heavy", "H"}, {"Partly", "P"}, {"Mostly", "M"},
		{"Few", "F"}, {"Thunderstorm", "T'storm"}, {" in ", ""}, {"Vicinity", ""},
		{" and ", " "}, {"Freezing Rain", "Frz Rn"}, {"Freezing", "Frz"},
		{"Unknown Precip", ""}, {"L Snow Fog", "L Snw/Fog"}, {" with ", "/"},
	}
	for _, r := range replacements {
		condition = strings.ReplaceAll(condition, r.old, r.new)
	}
	return condition
}
