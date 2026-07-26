package algorithm

import "math"

// VehicleProfile mirrors valhalla.TruckProfile (kept separate on purpose - this
// package has no dependency on the valhalla client, it only knows OSM data).
type VehicleProfile struct {
	HeightM  float64
	WeightKg float64
	Hazmat   bool
}

// allowed reports whether the vehicle may use this edge at all - a hard exclusion,
// same principle as Valhalla's truck costing (see documentations/features/2026-07-21-risk-scoring-layer.md
// for why we know Valhalla enforces height/weight this way).
func allowed(e Edge, p VehicleProfile) bool {
	if e.MaxHeightM > 0 && p.HeightM > e.MaxHeightM {
		return false
	}
	if e.MaxWeightT > 0 && p.WeightKg/1000 > e.MaxWeightT {
		return false
	}
	if e.Hazmat && p.Hazmat {
		return false
	}
	return true
}

// nodeAllowed mirrors allowed() for node-level barrier restrictions (see
// Node's doc comment in graph.go and LoadOSMXML) - a vehicle passing THROUGH
// a barrier node (not just along an edge) can be excluded by it.
func nodeAllowed(n Node, p VehicleProfile) bool {
	if n.MaxHeightM > 0 && p.HeightM > n.MaxHeightM {
		return false
	}
	if n.MaxWeightT > 0 && p.WeightKg/1000 > n.MaxWeightT {
		return false
	}
	return true
}

// roadClassMultiplier expresses a preference for higher-class roads (fewer
// intersections/driveways/pedestrians per km for a large vehicle) - the same
// idea as Valhalla-side risk-scoring's highway_ratio signal (documentations/
// features/2026-07-21-risk-scoring-layer.md), expressed here as a per-meter
// cost multiplier instead of a post-hoc score term. Heuristic, uncalibrated -
// same caveat as the rest of this module's cost function. Road classes not in
// this map (including edges built directly in tests, which leave RoadClass
// empty) are left at their plain length cost.
var roadClassMultiplier = map[string]float64{
	"motorway":       0.85,
	"motorway_link":  0.9,
	"trunk":          0.9,
	"trunk_link":     0.95,
	"primary":        1.0,
	"primary_link":   1.05,
	"secondary":      1.15,
	"secondary_link": 1.2,
}

// cost is the edge weight for an allowed edge: length in meters, with a mild
// penalty for poor/unpaved surfaces and a road-class preference multiplier.
// This is deliberately simple (see SPECIFIKACIJA.md 3.3.2) - the point of this
// module is the constraint-aware search itself, not a finely tuned cost model.
func cost(e Edge) float64 {
	base := e.LengthM
	switch e.Surface {
	case "unpaved", "gravel", "dirt", "sett":
		base *= 1.3
	}
	if mult, ok := roadClassMultiplier[e.RoadClass]; ok {
		base *= mult
	}
	return base
}

// turnPenaltyMeters approximates a routing cost for changing direction at a
// node, expressed in the same "meters" unit as cost() so it composes
// additively with distance/surface/road-class cost rather than a separate
// time unit - a deliberate simplification, consistent with the rest of this
// module. angleDeg is the absolute turn angle (0 = straight through, 180 =
// full U-turn). Thresholds/penalties are heuristic, uncalibrated - same
// caveat as roadClassMultiplier and every other weight in this project.
func turnPenaltyMeters(angleDeg float64) float64 {
	switch {
	case angleDeg < 30:
		return 0
	case angleDeg < 90:
		return 15 // gentle turn - highway interchange, roundabout exit
	case angleDeg < 150:
		return 60 // sharp turn
	default:
		return 120 // near U-turn
	}
}

// turnAngle returns the absolute deviation (0-180 degrees) between the
// incoming heading (from->via) and outgoing heading (via->to). Missing nodes
// (e.g. a synthetic test graph that never set Lat/Lon) resolve to (0,0) and
// therefore a heading of 0 - harmless since search() only calls this once a
// real prev pointer exists.
func turnAngle(g *Graph, from, via, to int64) float64 {
	fromNode, viaNode, toNode := g.Nodes[from], g.Nodes[via], g.Nodes[to]
	inBearing := bearing(fromNode.Lat, fromNode.Lon, viaNode.Lat, viaNode.Lon)
	outBearing := bearing(viaNode.Lat, viaNode.Lon, toNode.Lat, toNode.Lon)
	diff := math.Abs(inBearing - outBearing)
	if diff > 180 {
		diff = 360 - diff
	}
	return diff
}
