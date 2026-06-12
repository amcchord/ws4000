package nws

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	BaseURL    = "https://api.weather.gov"
	UserAgent  = "ws4000 (github.com/austinmcchord/ws4000)"
)

type Client struct {
	http    *http.Client
	cache   string
	fixture string
}

func NewClient(cacheDir, fixtureDir string) *Client {
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "ws4000-cache")
	}
	_ = os.MkdirAll(cacheDir, 0o755)
	return &Client{
		http: &http.Client{Timeout: 30 * time.Second},
		cache: cacheDir,
		fixture: fixtureDir,
	}
}

func (c *Client) GetJSON(path string, dest interface{}) error {
	body, err := c.get(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dest)
}

func (c *Client) get(path string) ([]byte, error) {
	if c.fixture != "" {
		fixPath := filepath.Join(c.fixture, strings.ReplaceAll(strings.TrimPrefix(path, "/"), "/", "_"))
		if data, err := os.ReadFile(fixPath); err == nil {
			return data, nil
		}
	}

	cacheKey := strings.ReplaceAll(path, "/", "_")
	cachePath := filepath.Join(c.cache, cacheKey)
	if data, err := os.ReadFile(cachePath); err == nil {
		if time.Since(fileModTime(cachePath)) < 5*time.Minute {
			return data, nil
		}
	}

	fullURL := path
	if !strings.HasPrefix(path, "http") {
		fullURL = BaseURL + path
	}

	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/geo+json, application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		if data, readErr := os.ReadFile(cachePath); readErr == nil {
			return data, nil
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("nws request failed: %s (%d)", path, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	_ = os.WriteFile(cachePath, body, 0o644)
	return body, nil
}

func fileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

type PointResponse struct {
	Properties PointProperties `json:"properties"`
}

type PointProperties struct {
	Forecast             string `json:"forecast"`
	ForecastGridData     string `json:"forecastGridData"`
	ForecastZone         string `json:"forecastZone"`
	RadarStation         string `json:"radarStation"`
	ObservationStations  string `json:"observationStations"`
	CWA                  string `json:"cwa"`
	TimeZone             string `json:"timeZone"`
	RelativeLocation     RelativeLocation `json:"relativeLocation"`
}

type RelativeLocation struct {
	Properties RelativeProps `json:"properties"`
}

type RelativeProps struct {
	City  string `json:"city"`
	State string `json:"state"`
}

type StationsResponse struct {
	Features []StationFeature `json:"features"`
}

type StationFeature struct {
	ID         string           `json:"id"`
	Properties StationProperties `json:"properties"`
}

type StationProperties struct {
	StationIdentifier string `json:"stationIdentifier"`
	Name              string `json:"name"`
}

type ObservationResponse struct {
	Features []ObservationFeature `json:"features"`
}

type ObservationFeature struct {
	Properties ObservationProperties `json:"properties"`
}

type ObservationProperties struct {
	Timestamp          string      `json:"timestamp"`
	TextDescription    string      `json:"textDescription"`
	Temperature        ValueUnit   `json:"temperature"`
	Dewpoint           ValueUnit   `json:"dewpoint"`
	WindSpeed          ValueUnit   `json:"windSpeed"`
	WindDirection      ValueUnit   `json:"windDirection"`
	WindGust           ValueUnit   `json:"windGust"`
	BarometricPressure ValueUnit   `json:"barometricPressure"`
	RelativeHumidity   ValueUnit   `json:"relativeHumidity"`
	Visibility         ValueUnit   `json:"visibility"`
	HeatIndex          ValueUnit   `json:"heatIndex"`
	WindChill          ValueUnit   `json:"windChill"`
	Icon               string      `json:"icon"`
	CloudLayers        []CloudLayer `json:"cloudLayers"`
}

type ValueUnit struct {
	Value *float64 `json:"value"`
	UnitCode string `json:"unitCode"`
}

type CloudLayer struct {
	Base ValueUnit `json:"base"`
}

type ForecastResponse struct {
	Properties ForecastProperties `json:"properties"`
}

type ForecastProperties struct {
	Periods []ForecastPeriod `json:"periods"`
}

type ForecastPeriod struct {
	Name              string `json:"name"`
	Temperature       int    `json:"temperature"`
	TemperatureUnit   string `json:"temperatureUnit"`
	ShortForecast     string `json:"shortForecast"`
	DetailedForecast  string `json:"detailedForecast"`
	StartTime         string `json:"startTime"`
	EndTime           string `json:"endTime"`
	IsDaytime         bool   `json:"isDaytime"`
}

type HourlyResponse struct {
	Properties HourlyProperties `json:"properties"`
}

type HourlyProperties struct {
	Periods []HourlyPeriod `json:"periods"`
}

type HourlyPeriod struct {
	StartTime           string    `json:"startTime"`
	Temperature         int       `json:"temperature"`
	TemperatureUnit     string    `json:"temperatureUnit"`
	WindSpeed           string    `json:"windSpeed"`
	ShortForecast       string    `json:"shortForecast"`
	ProbabilityOfPrecipitation ValueUnit `json:"probabilityOfPrecipitation"`
}

type AlertsResponse struct {
	Features []AlertFeature `json:"features"`
}

type AlertFeature struct {
	Properties AlertProperties `json:"properties"`
}

type AlertProperties struct {
	Event       string `json:"event"`
	Headline    string `json:"headline"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

func (c *Client) GetPoint(lat, lon float64) (*PointResponse, error) {
	path := fmt.Sprintf("/points/%.4f,%.4f", lat, lon)
	var resp PointResponse
	if err := c.GetJSON(path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetStations(urlStr string) (*StationsResponse, error) {
	var resp StationsResponse
	if err := c.GetJSON(urlStr, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetObservations(stationURL string, limit int) (*ObservationResponse, error) {
	u, err := url.Parse(stationURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("limit", fmt.Sprintf("%d", limit))
	u.RawQuery = q.Encode()
	var resp ObservationResponse
	if err := c.GetJSON(u.String(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetForecast(forecastURL, units string) (*ForecastResponse, error) {
	u, err := url.Parse(forecastURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if units == "metric" {
		q.Set("units", "si")
	} else {
		q.Set("units", "us")
	}
	u.RawQuery = q.Encode()
	var resp ForecastResponse
	if err := c.GetJSON(u.String(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetHourly(forecastURL, units string) (*HourlyResponse, error) {
	hourlyURL := strings.Replace(forecastURL, "/forecast", "/forecast/hourly", 1)
	u, err := url.Parse(hourlyURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if units == "metric" {
		q.Set("units", "si")
	} else {
		q.Set("units", "us")
	}
	u.RawQuery = q.Encode()
	var resp HourlyResponse
	if err := c.GetJSON(u.String(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetAlerts(zoneID string) (*AlertsResponse, error) {
	path := fmt.Sprintf("/alerts/active/zone/%s", zoneID)
	var resp AlertsResponse
	if err := c.GetJSON(path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func ZoneFromURL(zoneURL string) string {
	parts := strings.Split(zoneURL, "/")
	if len(parts) == 0 {
		return zoneURL
	}
	return parts[len(parts)-1]
}

func RadarFromURL(radarURL string) string {
	parts := strings.Split(radarURL, "/")
	if len(parts) == 0 {
		return radarURL
	}
	return parts[len(parts)-1]
}

func FilterStations(stations []StationFeature) []StationFeature {
	var out []StationFeature
	for _, s := range stations {
		id := s.Properties.StationIdentifier
		if len(id) == 4 && id[0] >= 'A' && id[0] <= 'Z' {
			out = append(out, s)
		}
	}
	return out
}
