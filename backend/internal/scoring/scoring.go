// Package scoring ranks Valhalla route candidates by a heuristic, driver-preference-
// driven score, using only the route-level signals available from the plain
// /route response (maneuver count/type, highway ratio, ferry/toll presence,
// duration). It does NOT have access to per-edge bridge/surface/hazmat-proximity
// data - that would require Valhalla's /trace_attributes endpoint or the bounded
// custom-graph module.
//
// Each dimension's contribution is scaled by (priority/3), where 3 is the
// "neutral" default (see store.DriverPreferences) - a driver who never touches
// their preferences gets behavior close to a fixed, balanced formula. Base
// weights below are a first heuristic pass, not calibrated against real
// incident/consumption data; see documentations/features/2026-07-21-driver-preference-scoring.md
// for the concrete bug (a 47%-longer, 30%-slower route winning purely on
// highway ratio) that motivated adding the time term.
package scoring

import (
	"math"
	"sort"

	"heavy-vehicle-routing/backend/internal/valhalla"
)

const (
	baselinePriority = 3.0 // "neutral" 1-5 priority value

	maneuverWeight  = 1.5 // per maneuver: more turns/transitions = more complexity for a large vehicle
	nonHighwayScale = 100 // full weight if 0% of the route is on highway (city/rural roads are riskier for trucks)
	ferryPenalty    = 50  // most heavy vehicles can't or shouldn't use ferries
	tollPenalty     = 5   // minor operational consideration, not a safety factor

	timeScale       = 150 // penalty scale for being X% slower than the fastest candidate
	sharpManeuverWt = 3.0 // per sharp turn/U-turn/roundabout entry - cargo-jostling proxy

	// fuelBaseWeightKg is the reference truck weight the fuel proxy is centered
	// on (a typical loaded semi) - not a real consumption figure, just an anchor
	// so heavier vehicles score worse than lighter ones on this dimension.
	fuelBaseWeightKg  = 40000.0
	fuelWeightFactor  = 0.3
	fuelManeuverBonus = 0.05

	// preferredStopRadiusM/Bonus: a route passing within this radius of a
	// favorite/preferred-brand fuel stop gets a flat score discount - not scaled
	// by any priority dial, since there wasn't a dedicated 1-5 dimension for it
	// (see documentations/features/2026-07-21-preferred-fuel-stations.md).
	preferredStopRadiusM = 3000.0
	preferredStopBonus   = 20.0
	shapeSampleStride    = 5 // check every Nth decoded shape point, for performance
)

// Preferences mirrors store.DriverPreferences without importing the store
// package - scoring only needs the four priority values, not the DB row.
type Preferences struct {
	FuelPriority    int
	CargoPriority   int
	HighwayPriority int
	TimePriority    int
}

type ScoredCandidate struct {
	valhalla.RouteCandidate
	RiskScore float64
}

func scalar(priority int) float64 {
	return float64(priority) / baselinePriority
}

// fuelProxy is a distance/weight/maneuver-based stand-in for fuel consumption -
// there's no elevation data in this project's Valhalla setup to model road
// grade, so this is a relative signal for comparing candidates, not a real
// liters/100km estimate.
func fuelProxy(c valhalla.RouteCandidate, vehicleWeightKg float64) float64 {
	weightFactor := 1 + (vehicleWeightKg/fuelBaseWeightKg)*fuelWeightFactor
	return c.DistanceKm*weightFactor + float64(c.ManeuverCount)*fuelManeuverBonus
}

// preferredStopDiscount returns a negative (score-improving) term if the
// candidate's route passes within preferredStopRadiusM of any of the driver's
// preferred stops (favorites and/or brand matches - the caller resolves which
// stops are "preferred" before calling Rank; scoring itself doesn't know about
// brands or favorites, only coordinates).
func preferredStopDiscount(shape string, preferredStops []valhalla.LatLon) float64 {
	if len(preferredStops) == 0 {
		return 0
	}
	points := valhalla.DecodePolyline6(shape)
	for _, stop := range preferredStops {
		for i := 0; i < len(points); i += shapeSampleStride {
			if haversineMeters(points[i], stop) <= preferredStopRadiusM {
				return -preferredStopBonus
			}
		}
	}
	return 0
}

const earthRadiusM = 6371000.0

func haversineMeters(a, b valhalla.LatLon) float64 {
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }
	dLat := toRad(b.Lat - a.Lat)
	dLon := toRad(b.Lon - a.Lon)
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(a.Lat))*math.Cos(toRad(b.Lat))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusM * 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}

func score(c valhalla.RouteCandidate, prefs Preferences, vehicleWeightKg, fastestDurationMin float64, preferredStops []valhalla.LatLon) float64 {
	var timeTerm float64
	if fastestDurationMin > 0 {
		timeTerm = (c.DurationMin - fastestDurationMin) / fastestDurationMin * timeScale
	}
	highwayTerm := (1 - c.HighwayRatio) * nonHighwayScale
	fuelTerm := fuelProxy(c, vehicleWeightKg)
	cargoTerm := float64(c.SharpManeuverCount) * sharpManeuverWt

	s := scalar(prefs.TimePriority)*timeTerm +
		scalar(prefs.HighwayPriority)*highwayTerm +
		scalar(prefs.FuelPriority)*fuelTerm +
		scalar(prefs.CargoPriority)*cargoTerm +
		float64(c.ManeuverCount)*maneuverWeight

	if c.HasFerry {
		s += ferryPenalty
	}
	if c.HasToll {
		s += tollPenalty
	}
	s += preferredStopDiscount(c.Shape, preferredStops)
	return s
}

// Rank scores every candidate against the driver's preferences and returns them
// sorted ascending by risk score (best first). preferredStops is the driver's
// resolved favorite/preferred-brand station coordinates (may be nil/empty - a
// route passing near one gets a small score discount, see preferredStopDiscount).
func Rank(candidates []valhalla.RouteCandidate, prefs Preferences, vehicleWeightKg float64, preferredStops []valhalla.LatLon) []ScoredCandidate {
	fastestDurationMin := math.Inf(1)
	for _, c := range candidates {
		if c.DurationMin < fastestDurationMin {
			fastestDurationMin = c.DurationMin
		}
	}

	ranked := make([]ScoredCandidate, len(candidates))
	for i, c := range candidates {
		ranked[i] = ScoredCandidate{RouteCandidate: c, RiskScore: score(c, prefs, vehicleWeightKg, fastestDurationMin, preferredStops)}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].RiskScore < ranked[j].RiskScore
	})
	return ranked
}
