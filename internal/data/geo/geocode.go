package geo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Result struct {
	Latitude  float64
	Longitude float64
	Name      string
}

type arcGISResponse struct {
	Candidates []struct {
		Location struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"location"`
		Address string `json:"address"`
	} `json:"candidates"`
}

func Geocode(query string) (*Result, error) {
	endpoint := "https://geocode.arcgis.com/arcgis/rest/services/World/GeocodeServer/findAddressCandidates"
	params := url.Values{}
	params.Set("f", "json")
	params.Set("outFields", "Match_addr,Addr_type")
	params.Set("maxLocations", "1")
	params.Set("singleLine", query)
	params.Set("countryCode", "USA")

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data arcGISResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if len(data.Candidates) == 0 {
		return nil, fmt.Errorf("no geocode results for %q", query)
	}
	c := data.Candidates[0]
	return &Result{
		Latitude:  c.Location.Y,
		Longitude: c.Location.X,
		Name:      c.Address,
	}, nil
}
