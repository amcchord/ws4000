package weather

import (
	"strings"

	"github.com/austinmcchord/ws4000/internal/data/geo"
	"github.com/austinmcchord/ws4000/internal/data/nws"
	"github.com/austinmcchord/ws4000/internal/engine"
)

type Service struct {
	Client *nws.Client
}

func NewService(cacheDir, fixtureDir string) *Service {
	return &Service{Client: nws.NewClient(cacheDir, fixtureDir)}
}

func (s *Service) ResolveLocation(location string, lat, lon float64) (float64, float64, string, error) {
	if lat != 0 || lon != 0 {
		return lat, lon, "", nil
	}
	if location == "" {
		location = "Orlando International Airport, Orlando, FL, USA"
	}
	result, err := geo.Geocode(location)
	if err != nil {
		return 0, 0, "", err
	}
	return result.Latitude, result.Longitude, result.Name, nil
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
		return nil, err
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
