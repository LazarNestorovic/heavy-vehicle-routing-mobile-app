// Package scoring ranks Valhalla route candidates by a heuristic risk score,
// using only the route-level signals available from the plain /route response
// (maneuver count, highway ratio, ferry/toll presence). It does NOT have access
// to per-edge bridge/surface/hazmat-proximity data - that would require
// Valhalla's /trace_attributes endpoint or the bounded custom-graph module.
// Weights below are a first heuristic pass, not calibrated against real incident
// data; tune them once evaluation data exists.
package scoring

import (
	"sort"

	"heavy-vehicle-routing/backend/internal/valhalla"
)

const (
	maneuverWeight  = 1.5 // per maneuver: more turns/transitions = more complexity for a large vehicle
	nonHighwayScale = 100 // full weight if 0% of the route is on highway (city/rural roads are riskier for trucks)
	ferryPenalty    = 50  // most heavy vehicles can't or shouldn't use ferries
	tollPenalty     = 5   // minor operational consideration, not a safety factor
)

type ScoredCandidate struct {
	valhalla.RouteCandidate
	RiskScore float64
}

// Score computes a heuristic risk score for a single candidate. Lower is better.
func Score(c valhalla.RouteCandidate) float64 {
	score := float64(c.ManeuverCount) * maneuverWeight
	score += (1 - c.HighwayRatio) * nonHighwayScale
	if c.HasFerry {
		score += ferryPenalty
	}
	if c.HasToll {
		score += tollPenalty
	}
	return score
}

// Rank scores every candidate and returns them sorted ascending by risk score (best first).
func Rank(candidates []valhalla.RouteCandidate) []ScoredCandidate {
	ranked := make([]ScoredCandidate, len(candidates))
	for i, c := range candidates {
		ranked[i] = ScoredCandidate{RouteCandidate: c, RiskScore: Score(c)}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].RiskScore < ranked[j].RiskScore
	})
	return ranked
}
