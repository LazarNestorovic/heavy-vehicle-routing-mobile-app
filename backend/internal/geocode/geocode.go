// Package geocode is a thin proxy to Nominatim's search endpoint (OSM-based
// geocoding, consistent with this project's OSM-first approach - see
// documentations/features/ entry for why this wasn't Google's paid Geocoding
// API, the same reasoning that kept route computation on Valhalla instead of
// Google Directions).
//
// Nominatim's usage policy (https://operations.osmfoundation.org/policies/nominatim/)
// requires a real User-Agent identifying the application and caps the public
// instance at roughly 1 request/second - both enforced here so every caller
// of this package benefits automatically, rather than each having to
// remember the rules. A real (non-thesis-demo) deployment with more than a
// handful of concurrent users should run its own Nominatim instance instead
// of the public one.
package geocode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// minRequestInterval is a bit over Nominatim's ~1 request/second cap.
const minRequestInterval = 1100 * time.Millisecond

type Client struct {
	baseURL    string
	userAgent  string
	httpClient *http.Client

	mu            sync.Mutex
	lastRequestAt time.Time
}

func New(baseURL, userAgent string) *Client {
	return &Client{
		baseURL:    baseURL,
		userAgent:  userAgent,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type Result struct {
	Lat         float64
	Lon         float64
	DisplayName string
}

type nominatimResult struct {
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
}

// Search queries Nominatim for query, returning up to limit results ordered
// by Nominatim's own relevance ranking.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	c.throttle()

	u := c.baseURL + "/search?" + url.Values{
		"q":      {query},
		"format": {"jsonv2"},
		"limit":  {strconv.Itoa(limit)},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build nominatim request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call nominatim: %w", err)
	}
	defer resp.Body.Close()

	var parsed []nominatimResult
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode nominatim response: %w", err)
	}

	out := make([]Result, 0, len(parsed))
	for _, r := range parsed {
		lat, errLat := strconv.ParseFloat(r.Lat, 64)
		lon, errLon := strconv.ParseFloat(r.Lon, 64)
		if errLat != nil || errLon != nil {
			continue // malformed coordinate from upstream - skip rather than fail the whole search
		}
		out = append(out, Result{Lat: lat, Lon: lon, DisplayName: r.DisplayName})
	}
	return out, nil
}

// throttle blocks until at least minRequestInterval has passed since the
// last request, so concurrent callers can't collectively exceed Nominatim's
// public usage policy.
func (c *Client) throttle() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if wait := minRequestInterval - time.Since(c.lastRequestAt); wait > 0 {
		time.Sleep(wait)
	}
	c.lastRequestAt = time.Now()
}
