package httpapi

import (
	"math"
	"strings"
	"testing"

	"heavy-vehicle-routing/backend/internal/valhalla"
)

// encodePolyline6 is the test-only inverse of valhalla.DecodePolyline6 (same
// small helper duplicated in internal/explain's tests - this package has no
// production need to encode, only Valhalla itself ever does that).
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

func TestPlainCoords(t *testing.T) {
	stops := []preferredStop{
		{LatLon: valhalla.LatLon{Lat: 44.8, Lon: 20.4}, Name: "Kod kuće"},
		{LatLon: valhalla.LatLon{Lat: 45.0, Lon: 19.9}, Name: "НИС Петрол"},
	}
	got := plainCoords(stops)
	want := []valhalla.LatLon{{Lat: 44.8, Lon: 20.4}, {Lat: 45.0, Lon: 19.9}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}

func TestPreferredStopMessage_NamesTheMatchedStop(t *testing.T) {
	shape := encodePolyline6([]valhalla.LatLon{{Lat: 44.800, Lon: 20.400}, {Lat: 44.801, Lon: 20.401}})
	stops := []preferredStop{
		{LatLon: valhalla.LatLon{Lat: 10.0, Lon: 10.0}, Name: "Daleko"},
		{LatLon: valhalla.LatLon{Lat: 44.800, Lon: 20.400}, Name: "НИС Петрол"},
	}

	got := preferredStopMessage(shape, stops)
	if got == nil {
		t.Fatal("expected a message when a preferred stop is near the route")
	}
	if !strings.Contains(*got, "НИС Петрол") {
		t.Errorf("expected the message to name the matched stop, got %q", *got)
	}
}

func TestPreferredStopMessage_NilWhenNothingNearby(t *testing.T) {
	shape := encodePolyline6([]valhalla.LatLon{{Lat: 44.800, Lon: 20.400}, {Lat: 44.801, Lon: 20.401}})
	stops := []preferredStop{{LatLon: valhalla.LatLon{Lat: 10.0, Lon: 10.0}, Name: "Daleko"}}

	if got := preferredStopMessage(shape, stops); got != nil {
		t.Errorf("expected no message when nothing is near the route, got %q", *got)
	}
}

func TestPreferredStopMessage_NilForEmptyStops(t *testing.T) {
	shape := encodePolyline6([]valhalla.LatLon{{Lat: 44.800, Lon: 20.400}})
	if got := preferredStopMessage(shape, nil); got != nil {
		t.Errorf("expected no message with no preferred stops at all, got %q", *got)
	}
}
