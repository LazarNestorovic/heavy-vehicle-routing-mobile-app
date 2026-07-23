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

// RouteCandidate is one possible route (the primary route or one of Valhalla's alternates),
// with the route-level signals available from the plain /route response (no per-edge
// bridge/surface/hazmat-proximity data - that would require /trace_attributes, which is
// out of scope for this layer; see the bounded custom-graph module for that).
type RouteCandidate struct {
	DistanceKm    float64
	DurationMin   float64
	Shape         string // encoded polyline6 of the first leg
	ManeuverCount int
	HighwayRatio  float64 // share of route length (0..1) that runs on Valhalla-flagged "highway" edges
	HasFerry      bool
	HasToll       bool
	StreetNames   []string // one entry per maneuver: its primary street name, or "" if unnamed
}

type routeRequest struct {
	Locations      []LatLon                   `json:"locations"`
	Costing        string                     `json:"costing"`
	CostingOptions map[string]truckCostingOpt `json:"costing_options"`
	Alternates     int                        `json:"alternates,omitempty"`
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

type tripData struct {
	Summary struct {
		Time     float64 `json:"time"`   // seconds
		Length   float64 `json:"length"` // kilometers (default units)
		HasFerry bool    `json:"has_ferry"`
		HasToll  bool    `json:"has_toll"`
	} `json:"summary"`
	Legs []struct {
		Shape     string `json:"shape"`
		Maneuvers []struct {
			Length      float64  `json:"length"` // kilometers
			Highway     bool     `json:"highway"`
			StreetNames []string `json:"street_names"`
		} `json:"maneuvers"`
	} `json:"legs"`
}

type routeResponse struct {
	Trip       *tripData `json:"trip"`
	Alternates []struct {
		Trip tripData `json:"trip"`
	} `json:"alternates"`
	Error     string `json:"error"`
	ErrorCode int    `json:"error_code"`
}

func toCandidate(t tripData) RouteCandidate {
	c := RouteCandidate{
		DistanceKm:  t.Summary.Length,
		DurationMin: t.Summary.Time / 60,
		HasFerry:    t.Summary.HasFerry,
		HasToll:     t.Summary.HasToll,
	}
	if len(t.Legs) > 0 {
		c.Shape = t.Legs[0].Shape
	}

	var highwayKm float64
	for _, leg := range t.Legs {
		for _, m := range leg.Maneuvers {
			c.ManeuverCount++
			if m.Highway {
				highwayKm += m.Length
			}
			name := ""
			if len(m.StreetNames) > 0 {
				name = m.StreetNames[0]
			}
			c.StreetNames = append(c.StreetNames, name)
		}
	}
	if c.DistanceKm > 0 {
		c.HighwayRatio = highwayKm / c.DistanceKm
	}
	return c
}

// RouteAlternates requests a truck-costed route plus up to numAlternates alternative routes
// between origin and destination for the given vehicle profile. The primary route is always
// candidates[0]; order beyond that is whatever Valhalla returned.
func (c *Client) RouteAlternates(ctx context.Context, origin, destination LatLon, profile TruckProfile, numAlternates int) ([]RouteCandidate, error) {
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
		Alternates: numAlternates,
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

	candidates := make([]RouteCandidate, 0, 1+len(parsed.Alternates))
	candidates = append(candidates, toCandidate(*parsed.Trip))
	for _, alt := range parsed.Alternates {
		candidates = append(candidates, toCandidate(alt.Trip))
	}

	return candidates, nil
}
