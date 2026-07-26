package explain

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"heavy-vehicle-routing/backend/internal/scoring"
	"heavy-vehicle-routing/backend/internal/valhalla"
)

// fakeValhallaServer returns an httptest.Server that answers every /route call
// with a minimal valid response (a flat, maneuver-less trip) and counts how
// many requests it received - used to verify cachedReference actually skips
// the network round-trip on a cache hit.
func fakeValhallaServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"trip":{"summary":{"time":60,"length":1.0,"has_ferry":false,"has_toll":false},"legs":[{"shape":"","maneuvers":[]}]}}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// encodePolyline6 is the test-only inverse of valhalla.DecodePolyline6 - lets
// these tests build real RouteCandidate.Shape strings from plain coordinates,
// instead of only exercising divergencePoint/nearestManeuverStreetName in
// isolation. Standard Google polyline algorithm at 1e-6 precision.
func encodePolyline6(points []valhalla.LatLon) string {
	var buf strings.Builder
	prevLat, prevLon := 0, 0
	encodeValue := func(v int) {
		shifted := v << 1
		if v < 0 {
			shifted = ^shifted
		}
		for shifted >= 0x20 {
			buf.WriteByte(byte((shifted&0x1f)|0x20) + 63)
			shifted >>= 5
		}
		buf.WriteByte(byte(shifted) + 63)
	}
	for _, p := range points {
		lat := int(math.Round(p.Lat * 1e6))
		lon := int(math.Round(p.Lon * 1e6))
		encodeValue(lat - prevLat)
		encodeValue(lon - prevLon)
		prevLat, prevLon = lat, lon
	}
	return buf.String()
}

func TestDivergencePoint_FindsWhereShapesStopOverlapping(t *testing.T) {
	// Indices 0-4 shared; chosen[5] jumps far away while reference[5]
	// continues the straight line - shapeSampleStride=5 means only indices 0
	// and 5 are actually sampled here, which this test relies on.
	shared := []valhalla.LatLon{
		{Lat: 44.800, Lon: 20.400},
		{Lat: 44.801, Lon: 20.401},
		{Lat: 44.802, Lon: 20.402},
		{Lat: 44.803, Lon: 20.403},
		{Lat: 44.804, Lon: 20.404},
	}
	chosen := append(append([]valhalla.LatLon{}, shared...), valhalla.LatLon{Lat: 44.900, Lon: 20.700})
	reference := append(append([]valhalla.LatLon{}, shared...), valhalla.LatLon{Lat: 44.805, Lon: 20.405})

	got, found := divergencePoint(chosen, reference)
	if !found {
		t.Fatal("expected a divergence point to be found")
	}
	if got != chosen[5] {
		t.Errorf("expected divergence at chosen[5]=%+v, got %+v", chosen[5], got)
	}
}

func TestDivergencePoint_NoDivergenceWhenShapesMatch(t *testing.T) {
	points := []valhalla.LatLon{
		{Lat: 44.800, Lon: 20.400},
		{Lat: 44.801, Lon: 20.401},
		{Lat: 44.802, Lon: 20.402},
	}
	_, found := divergencePoint(points, points)
	if found {
		t.Error("expected no divergence between identical shapes")
	}
}

func TestNearestManeuverStreetName_PicksClosestManeuver(t *testing.T) {
	route := valhalla.RouteCandidate{
		StreetNames: []string{"A1", "Ulica Skretanja"},
		ManeuverPoints: []valhalla.LatLon{
			{Lat: 44.800, Lon: 20.400},
			{Lat: 44.900, Lon: 20.700},
		},
	}

	got := nearestManeuverStreetName(valhalla.LatLon{Lat: 44.900, Lon: 20.700}, route)
	if got != "Ulica Skretanja" {
		t.Errorf("expected the maneuver at the exact target point to win, got %q", got)
	}
}

func TestNearestManeuverStreetName_SkipsUnresolvedPoints(t *testing.T) {
	route := valhalla.RouteCandidate{
		StreetNames: []string{"Nerazrešen manevar", "A1"},
		ManeuverPoints: []valhalla.LatLon{
			{}, // begin_shape_index couldn't be resolved for this one
			{Lat: 44.800, Lon: 20.400},
		},
	}

	got := nearestManeuverStreetName(valhalla.LatLon{Lat: 44.800, Lon: 20.400}, route)
	if got != "A1" {
		t.Errorf("expected the unresolved (zero-value) maneuver point to be skipped, got %q", got)
	}
}

func TestDivergenceStreetName_EndToEnd(t *testing.T) {
	shared := []valhalla.LatLon{
		{Lat: 44.800, Lon: 20.400},
		{Lat: 44.801, Lon: 20.401},
		{Lat: 44.802, Lon: 20.402},
		{Lat: 44.803, Lon: 20.403},
		{Lat: 44.804, Lon: 20.404},
	}
	chosenPoints := append(append([]valhalla.LatLon{}, shared...), valhalla.LatLon{Lat: 44.900, Lon: 20.700})
	refPoints := append(append([]valhalla.LatLon{}, shared...), valhalla.LatLon{Lat: 44.805, Lon: 20.405})

	chosen := valhalla.RouteCandidate{
		Shape:          encodePolyline6(chosenPoints),
		StreetNames:    []string{"A1", "Ulica Skretanja"},
		ManeuverPoints: []valhalla.LatLon{chosenPoints[0], chosenPoints[5]},
	}
	reference := valhalla.RouteCandidate{Shape: encodePolyline6(refPoints)}

	got := divergenceStreetName(chosen, reference)
	if got != "Ulica Skretanja" {
		t.Errorf("expected the maneuver nearest the geometric divergence point, got %q", got)
	}
}

func TestDivergenceStreetName_FallsBackWhenShapesEmpty(t *testing.T) {
	got := divergenceStreetName(valhalla.RouteCandidate{}, valhalla.RouteCandidate{})
	if got != "jednoj deonici puta" {
		t.Errorf("expected the generic fallback for empty shapes, got %q", got)
	}
}

func TestExplainer_CachedReference_ReusesWithinTTL(t *testing.T) {
	srv, hits := fakeValhallaServer(t)
	e := New(valhalla.New(srv.URL))

	origin := valhalla.LatLon{Lat: 44.8, Lon: 20.4}
	destination := valhalla.LatLon{Lat: 45.25, Lon: 19.85}

	if _, err := e.cachedReference(context.Background(), origin, destination, scoring.Preferences{}, 40000, nil); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := e.cachedReference(context.Background(), origin, destination, scoring.Preferences{}, 40000, nil); err != nil {
		t.Fatalf("second call: %v", err)
	}

	if got := hits.Load(); got != 1 {
		t.Errorf("expected exactly 1 Valhalla request (second call should hit the cache), got %d", got)
	}
}

func TestExplainer_CachedReference_MissesForDifferentCoordinates(t *testing.T) {
	srv, hits := fakeValhallaServer(t)
	e := New(valhalla.New(srv.URL))

	if _, err := e.cachedReference(context.Background(), valhalla.LatLon{Lat: 44.8, Lon: 20.4}, valhalla.LatLon{Lat: 45.25, Lon: 19.85}, scoring.Preferences{}, 40000, nil); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := e.cachedReference(context.Background(), valhalla.LatLon{Lat: 43.0, Lon: 21.0}, valhalla.LatLon{Lat: 44.0, Lon: 22.0}, scoring.Preferences{}, 40000, nil); err != nil {
		t.Fatalf("second call: %v", err)
	}

	if got := hits.Load(); got != 2 {
		t.Errorf("expected 2 Valhalla requests for two different coordinate pairs, got %d", got)
	}
}
