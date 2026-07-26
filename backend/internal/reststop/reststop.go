// Package reststop finds the nearest fuel/parking/rest-area OSM node to a given
// point. Data comes from a small, pre-extracted OSM XML file (~2000 nodes for all
// of Serbia - see documentations/guides/extract-osm-corridor.md for the same
// osmium-based approach), loaded once at startup and kept in memory.
package reststop

import (
	"encoding/xml"
	"math"
	"os"
)

type Stop struct {
	ID      int64
	Lat     float64
	Lon     float64
	Amenity string // "fuel", "parking", or "rest_area"
	Name    string // may be empty - not every node is tagged with a name
	Brand   string // may be empty - only fuel stations tend to carry this (e.g. "НИС Петрол")
}

type osmXML struct {
	Nodes []osmNode `xml:"node"`
}

type osmNode struct {
	ID   int64    `xml:"id,attr"`
	Lat  float64  `xml:"lat,attr"`
	Lon  float64  `xml:"lon,attr"`
	Tags []osmTag `xml:"tag"`
}

type osmTag struct {
	K string `xml:"k,attr"`
	V string `xml:"v,attr"`
}

func (n osmNode) tag(key string) (string, bool) {
	for _, t := range n.Tags {
		if t.K == key {
			return t.V, true
		}
	}
	return "", false
}

// Load parses the rest-stop OSM XML extract into a flat list of Stops.
func Load(path string) ([]Stop, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var doc osmXML
	if err := xml.NewDecoder(f).Decode(&doc); err != nil {
		return nil, err
	}

	stops := make([]Stop, 0, len(doc.Nodes))
	for _, n := range doc.Nodes {
		amenity, hasAmenity := n.tag("amenity")
		_, isRestArea := n.tag("highway") // filtered extract only has highway=rest_area nodes
		switch {
		case hasAmenity && (amenity == "fuel" || amenity == "parking"):
		case isRestArea:
			amenity = "rest_area"
		default:
			continue
		}
		name, _ := n.tag("name")
		brand, _ := n.tag("brand")
		stops = append(stops, Stop{ID: n.ID, Lat: n.Lat, Lon: n.Lon, Amenity: amenity, Name: name, Brand: brand})
	}
	return stops, nil
}

// Finder answers nearest-stop queries over a fixed, in-memory list of Stops.
type Finder struct {
	stops []Stop
}

func NewFinder(stops []Stop) *Finder {
	return &Finder{stops: stops}
}

// Nearest does a linear scan - fine for ~2000 stops; would need a spatial index
// at a much larger scale (e.g. all of Europe).
func (f *Finder) Nearest(lat, lon float64) (Stop, float64, bool) {
	return nearestIn(f.stops, lat, lon)
}

// ByBrand returns every stop tagged with the given brand (e.g. "НИС Петрол") -
// used by route scoring to check whether a candidate passes near any of them,
// not just the single nearest one.
func (f *Finder) ByBrand(brand string) []Stop {
	if brand == "" {
		return nil
	}
	var matches []Stop
	for _, s := range f.stops {
		if s.Brand == brand {
			matches = append(matches, s)
		}
	}
	return matches
}

// Point is a bare lat/lon pair - kept separate from valhalla.LatLon so this
// package stays free of a dependency on the valhalla client (same principle
// as the rest of this package's design), even though callers will usually
// build these from a decoded Valhalla route shape (valhalla.DecodePolyline6).
type Point struct {
	Lat float64
	Lon float64
}

// DefaultRouteCorridorRadiusM is how far off the route's sampled shape a stop
// may be before NearestOnRoute rejects it as "not really on the way" - same
// order of magnitude as scoring.go's preferredStopRadiusM.
const DefaultRouteCorridorRadiusM = 3000.0

// onRoute reports whether (lat,lon) is within radiusM of at least one point
// in routePoints. An empty routePoints means "no route given" - treated as
// "don't filter" so existing callers that never pass a route keep working.
func onRoute(lat, lon float64, routePoints []Point, radiusM float64) bool {
	if len(routePoints) == 0 {
		return true
	}
	for _, p := range routePoints {
		if haversineMeters(lat, lon, p.Lat, p.Lon) <= radiusM {
			return true
		}
	}
	return false
}

func filterStops(stops []Stop, keep func(Stop) bool) []Stop {
	out := make([]Stop, 0, len(stops))
	for _, s := range stops {
		if keep(s) {
			out = append(out, s)
		}
	}
	return out
}

// hazmatFuelPreferenceToleranceM: for a hazmat vehicle, a fuel station within
// this much extra distance of the plain-nearest stop is preferred over a
// closer parking/rest-area node. Heuristic proxy, not a real safety
// guarantee - OSM doesn't reliably tag whether a hazmat vehicle may actually
// stop at a given node; a staffed fuel station is simply a more plausible bet
// than an unstaffed parking lot (see documentations/features/ entry).
const hazmatFuelPreferenceToleranceM = 5000.0

// nearestPreferringAmenity finds the plain nearest stop, but if a stop tagged
// preferredAmenity exists within toleranceM of that plain-nearest distance,
// returns that one instead.
func nearestPreferringAmenity(stops []Stop, lat, lon float64, preferredAmenity string, toleranceM float64) (Stop, float64, bool) {
	best, bestDist, found := nearestIn(stops, lat, lon)
	if !found || best.Amenity == preferredAmenity {
		return best, bestDist, found
	}
	var preferred []Stop
	for _, s := range stops {
		if s.Amenity == preferredAmenity {
			preferred = append(preferred, s)
		}
	}
	if pStop, pDist, pFound := nearestIn(preferred, lat, lon); pFound && pDist <= bestDist+toleranceM {
		return pStop, pDist, true
	}
	return best, bestDist, found
}

// NearestOnRoute is NearestPreferred with one added constraint: a candidate
// must be within routeCorridorRadiusM of at least one of routePoints (a
// sampling of the DECODED ROUTE SHAPE, not just the single "you'll be here
// after N minutes" point NearestPreferred is normally called with). Without
// this, the previously "nearest" stop could be the closest point-to-point
// match to that one interpolated point while still being a real detour off
// the corridor the vehicle is actually driving - e.g. a fuel station down a
// side road into a village. Falls back to the plain (route-unfiltered)
// NearestPreferred if nothing satisfies the corridor constraint at all - a
// long trip through a sparse area shouldn't lose its rest-stop suggestion
// entirely just because nothing happens to sit directly on the sampled shape.
//
// hazmat, if true, applies nearestPreferringAmenity at the final (no
// favorite/brand match) tier - a hazmat load prefers a staffed fuel station
// over an unstaffed parking/rest-area node when both are reasonably close.
func (f *Finder) NearestOnRoute(lat, lon float64, brand string, favorites []Stop, maxRadiusM float64, routePoints []Point, routeCorridorRadiusM float64, hazmat bool) (Stop, float64, bool) {
	onCorridor := func(s Stop) bool { return onRoute(s.Lat, s.Lon, routePoints, routeCorridorRadiusM) }

	if stop, dist, found := nearestIn(filterStops(favorites, onCorridor), lat, lon); found && dist <= maxRadiusM {
		return stop, dist, true
	}

	if brand != "" {
		var brandMatches []Stop
		for _, s := range f.stops {
			if s.Brand == brand && onCorridor(s) {
				brandMatches = append(brandMatches, s)
			}
		}
		if stop, dist, found := nearestIn(brandMatches, lat, lon); found && dist <= maxRadiusM {
			return stop, dist, true
		}
	}

	corridorStops := filterStops(f.stops, onCorridor)
	if hazmat {
		if stop, dist, found := nearestPreferringAmenity(corridorStops, lat, lon, "fuel", hazmatFuelPreferenceToleranceM); found {
			return stop, dist, true
		}
	}
	if stop, dist, found := nearestIn(corridorStops, lat, lon); found {
		return stop, dist, true
	}

	return f.NearestPreferred(lat, lon, brand, favorites, maxRadiusM)
}

// DefaultPreferredRadiusM is how far a driver's preferred brand/favorite stop
// may be from the reference point before it's not worth the detour and we fall
// back to the plain nearest stop of any kind.
const DefaultPreferredRadiusM = 15000

// NearestPreferred prefers, in order: a saved favorite stop, then a stop
// matching the driver's preferred brand, within maxRadiusM of (lat, lon) -
// falling back to the plain nearest stop of any kind/brand if neither is close
// enough (or set at all).
func (f *Finder) NearestPreferred(lat, lon float64, brand string, favorites []Stop, maxRadiusM float64) (Stop, float64, bool) {
	if stop, dist, found := nearestIn(favorites, lat, lon); found && dist <= maxRadiusM {
		return stop, dist, true
	}

	if brand != "" {
		var brandMatches []Stop
		for _, s := range f.stops {
			if s.Brand == brand {
				brandMatches = append(brandMatches, s)
			}
		}
		if stop, dist, found := nearestIn(brandMatches, lat, lon); found && dist <= maxRadiusM {
			return stop, dist, true
		}
	}

	return f.Nearest(lat, lon)
}

func nearestIn(stops []Stop, lat, lon float64) (Stop, float64, bool) {
	var (
		best     Stop
		bestDist = math.Inf(1)
		found    bool
	)
	for _, s := range stops {
		d := haversineMeters(lat, lon, s.Lat, s.Lon)
		if d < bestDist {
			bestDist = d
			best = s
			found = true
		}
	}
	return best, bestDist, found
}

const earthRadiusM = 6371000.0

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusM * 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}
