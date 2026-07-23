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
		stops = append(stops, Stop{ID: n.ID, Lat: n.Lat, Lon: n.Lon, Amenity: amenity, Name: name})
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
	var (
		best     Stop
		bestDist = math.Inf(1)
		found    bool
	)
	for _, s := range f.stops {
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
