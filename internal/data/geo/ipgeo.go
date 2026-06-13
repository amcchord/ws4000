package geo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// IPResult is a location derived from the caller's public IP address.
type IPResult struct {
	Latitude  float64
	Longitude float64
	City      string
	Region    string
}

// LocateByIP determines an approximate location from the machine's public IP.
// It tries ipapi.co first, then ip-api.com as a fallback. Both are free,
// keyless services; accuracy is city-level which is plenty for a forecast.
func LocateByIP() (*IPResult, error) {
	if r, err := locateIPAPICo(); err == nil {
		return r, nil
	}
	return locateIPAPICom()
}

func locateIPAPICo() (*IPResult, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "https://ipapi.co/json/", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ws4000/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ipapi.co status %d", resp.StatusCode)
	}
	var data struct {
		Latitude   float64 `json:"latitude"`
		Longitude  float64 `json:"longitude"`
		City       string  `json:"city"`
		RegionCode string  `json:"region_code"`
		Country    string  `json:"country_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Latitude == 0 && data.Longitude == 0 {
		return nil, fmt.Errorf("ipapi.co returned no coordinates")
	}
	return &IPResult{
		Latitude:  data.Latitude,
		Longitude: data.Longitude,
		City:      data.City,
		Region:    data.RegionCode,
	}, nil
}

func locateIPAPICom() (*IPResult, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("http://ip-api.com/json/?fields=status,lat,lon,city,region")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data struct {
		Status string  `json:"status"`
		Lat    float64 `json:"lat"`
		Lon    float64 `json:"lon"`
		City   string  `json:"city"`
		Region string  `json:"region"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Status != "success" || (data.Lat == 0 && data.Lon == 0) {
		return nil, fmt.Errorf("ip-api.com lookup failed")
	}
	return &IPResult{
		Latitude:  data.Lat,
		Longitude: data.Lon,
		City:      data.City,
		Region:    data.Region,
	}, nil
}
