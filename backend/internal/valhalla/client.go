// Package valhalla wraps the Valhalla HTTP routing API, mapping our vehicle
// profile into Valhalla's truck costing options.
package valhalla

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type LatLon struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// TruckProfile mirrors Valhalla's truck costing_options, in SI units (meters, kilograms).
type TruckProfile struct {
	HeightM    float64
	WidthM     float64
	LengthM    float64
	WeightKg   float64
	AxleLoadKg float64
	Hazmat     bool
}

type RouteResult struct {
	DistanceKm float64
	DurationMin float64
	Shape       string // encoded polyline6 of the first leg
}

type routeRequest struct {
	Locations      []LatLon                  `json:"locations"`
	Costing        string                     `json:"costing"`
	CostingOptions map[string]truckCostingOpt `json:"costing_options"`
}

// truckCostingOpt uses Valhalla's units: meters for dimensions, metric tons for weight.
type truckCostingOpt struct {
	Height   float64 `json:"height"`
	Width    float64 `json:"width"`
	Length   float64 `json:"length"`
	Weight   float64 `json:"weight"`
	AxleLoad float64 `json:"axle_load"`
	Hazmat   bool    `json:"hazmat"`
}

type routeResponse struct {
	Trip *struct {
		Summary struct {
			Time   float64 `json:"time"`   // seconds
			Length float64 `json:"length"` // kilometers (default units)
		} `json:"summary"`
		Legs []struct {
			Shape string `json:"shape"`
		} `json:"legs"`
	} `json:"trip"`
	Error     string `json:"error"`
	ErrorCode int    `json:"error_code"`
}

// Route requests a truck-costed route between origin and destination for the given vehicle profile.
func (c *Client) Route(ctx context.Context, origin, destination LatLon, profile TruckProfile) (*RouteResult, error) {
	body := routeRequest{
		Locations: []LatLon{origin, destination},
		Costing:   "truck",
		CostingOptions: map[string]truckCostingOpt{
			"truck": {
				Height:   profile.HeightM,
				Width:    profile.WidthM,
				Length:   profile.LengthM,
				Weight:   profile.WeightKg / 1000,   // Valhalla expects metric tons
				AxleLoad: profile.AxleLoadKg / 1000, // Valhalla expects metric tons
				Hazmat:   profile.Hazmat,
			},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode valhalla request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/route", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build valhalla request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call valhalla: %w", err)
	}
	defer resp.Body.Close()

	var parsed routeResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode valhalla response: %w", err)
	}

	if parsed.Trip == nil {
		if parsed.Error != "" {
			return nil, fmt.Errorf("valhalla: %s", parsed.Error)
		}
		return nil, fmt.Errorf("valhalla returned no route (http %d)", resp.StatusCode)
	}

	var shape string
	if len(parsed.Trip.Legs) > 0 {
		shape = parsed.Trip.Legs[0].Shape
	}

	return &RouteResult{
		DistanceKm:  parsed.Trip.Summary.Length,
		DurationMin: parsed.Trip.Summary.Time / 60,
		Shape:       shape,
	}, nil
}
