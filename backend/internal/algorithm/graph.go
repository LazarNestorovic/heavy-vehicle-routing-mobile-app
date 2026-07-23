// Package algorithm is a standalone, from-scratch A*/Dijkstra implementation over a
// bounded road subgraph (see SPECIFIKACIJA.md 3.3.2). It reads OSM data directly
// (not through Valhalla) and applies a custom, vehicle-aware cost function. It is
// NOT on the production routing path (Valhalla is, for national-scale coverage) -
// this exists to demonstrate and evaluate the algorithm itself for the thesis.
package algorithm

import (
	"errors"
	"math"
)

var (
	errNoNodes = errors.New("algorithm: graph has no nodes")
	errNoPath  = errors.New("algorithm: no path found for the given vehicle profile")
)

type Node struct {
	ID  int64
	Lat float64
	Lon float64
}

// Edge is one directed road segment between two consecutive nodes of an OSM way.
// MaxHeightM/MaxWeightT are 0 when the way carries no such restriction.
type Edge struct {
	To         int64
	LengthM    float64
	MaxHeightM float64
	MaxWeightT float64
	Hazmat     bool // true if this edge forbids hazardous-materials transport (hazmat=no)
	Surface    string
}

type Graph struct {
	Nodes   map[int64]Node
	AdjList map[int64][]Edge
}

func NewGraph() *Graph {
	return &Graph{
		Nodes:   make(map[int64]Node),
		AdjList: make(map[int64][]Edge),
	}
}

func (g *Graph) addEdge(from int64, e Edge) {
	g.AdjList[from] = append(g.AdjList[from], e)
}

// NearestNode does a linear scan for the closest node to (lat, lon) that has at
// least one outgoing edge - plenty of nodes in an OSM extract are dead ends (the
// last node of a oneway street, a short cut-off link road at the extract's
// boundary, etc.); picking one of those as a search start would immediately fail
// with "no path found" for reasons that have nothing to do with the vehicle
// profile. Fine for a bounded graph with tens of thousands of nodes; would need
// a spatial index (e.g. a grid or k-d tree) at national scale.
func (g *Graph) NearestNode(lat, lon float64) (int64, error) {
	var (
		bestID   int64
		bestDist = math.Inf(1)
		found    bool
	)
	for id, n := range g.Nodes {
		if len(g.AdjList[id]) == 0 {
			continue
		}
		d := haversineMeters(lat, lon, n.Lat, n.Lon)
		if d < bestDist {
			bestDist = d
			bestID = id
			found = true
		}
	}
	if !found {
		return 0, errNoNodes
	}
	return bestID, nil
}

const earthRadiusM = 6371000.0

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusM * c
}
