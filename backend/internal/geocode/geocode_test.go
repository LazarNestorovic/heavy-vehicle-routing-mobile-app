package geocode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func fakeNominatimServer(t *testing.T, body string) (*httptest.Server, *string) {
	t.Helper()
	var gotUserAgent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &gotUserAgent
}

func TestSearch_ParsesResults(t *testing.T) {
	srv, _ := fakeNominatimServer(t, `[
		{"lat":"44.8125","lon":"20.4612","display_name":"Beograd, Srbija"},
		{"lat":"45.2551","lon":"19.8452","display_name":"Novi Sad, Srbija"}
	]`)
	c := New(srv.URL, "test-agent/1.0")

	results, err := c.Search(context.Background(), "Beograd", 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Lat != 44.8125 || results[0].Lon != 20.4612 || results[0].DisplayName != "Beograd, Srbija" {
		t.Errorf("unexpected first result: %+v", results[0])
	}
}

func TestSearch_SendsUserAgent(t *testing.T) {
	srv, gotUserAgent := fakeNominatimServer(t, `[]`)
	c := New(srv.URL, "heavy-vehicle-routing-thesis/1.0")

	if _, err := c.Search(context.Background(), "test", 1); err != nil {
		t.Fatalf("search: %v", err)
	}
	if *gotUserAgent != "heavy-vehicle-routing-thesis/1.0" {
		t.Errorf("expected the configured User-Agent to be sent, got %q", *gotUserAgent)
	}
}

func TestSearch_SkipsMalformedCoordinates(t *testing.T) {
	srv, _ := fakeNominatimServer(t, `[
		{"lat":"not-a-number","lon":"20.4612","display_name":"Bad"},
		{"lat":"44.8125","lon":"20.4612","display_name":"Good"}
	]`)
	c := New(srv.URL, "test-agent/1.0")

	results, err := c.Search(context.Background(), "test", 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 || results[0].DisplayName != "Good" {
		t.Errorf("expected only the well-formed result to survive, got %+v", results)
	}
}

func TestThrottle_EnforcesMinimumInterval(t *testing.T) {
	srv, _ := fakeNominatimServer(t, `[]`)
	c := New(srv.URL, "test-agent/1.0")

	start := time.Now()
	if _, err := c.Search(context.Background(), "first", 1); err != nil {
		t.Fatalf("first search: %v", err)
	}
	if _, err := c.Search(context.Background(), "second", 1); err != nil {
		t.Fatalf("second search: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed < minRequestInterval {
		t.Errorf("expected the second call to be throttled to at least %v, took %v", minRequestInterval, elapsed)
	}
}
