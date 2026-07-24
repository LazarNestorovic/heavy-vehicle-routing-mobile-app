// Package explain implements SPECIFIKACIJA.md section 3.10: when the chosen route
// detours from the shortest possible path, tell the driver *why*, not just *what*.
//
// Mechanism: request a "reference" route with every vehicle dimension relaxed to
// an unrealistically permissive value (so no real-world restriction can exclude
// any edge), then compare its distance to the chosen route's. If they're
// basically the same, there's no meaningful detour to explain. If they differ,
// relax the requested profile's dimensions one at a time (height, weight,
// axle_load, width, hazmat) until the resulting route matches the reference
// distance again - whichever dimension's relaxation did that is the binding
// constraint. This is the same principle as the manual binary search used to
// confirm the Novi Banovci height restriction (documentations/features/2026-07-21-risk-scoring-layer.md),
// automated and generalized to all vehicle dimensions.
package explain

import (
	"context"
	"fmt"
	"math"

	"heavy-vehicle-routing/backend/internal/scoring"
	"heavy-vehicle-routing/backend/internal/valhalla"
)

// distanceToleranceKm treats two routes as "the same route" if their distances
// are within this margin - avoids false positives from minor path differences
// that have nothing to do with vehicle constraints.
const distanceToleranceKm = 1.0

// numAlternates matches httpapi's own choice (see server.go) - kept as a
// separate constant on purpose: this package probes Valhalla independently of
// the request path and shouldn't depend on an unexported httpapi constant.
const numAlternates = 2

type Explainer struct {
	Valhalla *valhalla.Client
}

func New(v *valhalla.Client) *Explainer {
	return &Explainer{Valhalla: v}
}

func referenceProfile() valhalla.TruckProfile {
	return valhalla.TruckProfile{
		HeightM: 100, WidthM: 100, LengthM: 100,
		WeightKg: 900000, AxleLoadKg: 900000,
		Hazmat: false,
	}
}

type dimension struct {
	name  string
	relax func(valhalla.TruckProfile) valhalla.TruckProfile
	value func(valhalla.TruckProfile) string
}

func dimensions() []dimension {
	ref := referenceProfile()
	return []dimension{
		{"height", func(p valhalla.TruckProfile) valhalla.TruckProfile { p.HeightM = ref.HeightM; return p },
			func(p valhalla.TruckProfile) string { return fmt.Sprintf("visina vozila (%.1fm)", p.HeightM) }},
		{"weight", func(p valhalla.TruckProfile) valhalla.TruckProfile { p.WeightKg = ref.WeightKg; return p },
			func(p valhalla.TruckProfile) string { return fmt.Sprintf("težina vozila (%.0fkg)", p.WeightKg) }},
		{"axle_load", func(p valhalla.TruckProfile) valhalla.TruckProfile { p.AxleLoadKg = ref.AxleLoadKg; return p },
			func(p valhalla.TruckProfile) string {
				return fmt.Sprintf("osovinsko opterećenje vozila (%.0fkg)", p.AxleLoadKg)
			}},
		{"width", func(p valhalla.TruckProfile) valhalla.TruckProfile { p.WidthM = ref.WidthM; return p },
			func(p valhalla.TruckProfile) string { return fmt.Sprintf("širina vozila (%.2fm)", p.WidthM) }},
		{"hazmat", func(p valhalla.TruckProfile) valhalla.TruckProfile { p.Hazmat = false; return p },
			func(p valhalla.TruckProfile) string { return "prevoz opasnog tereta" }},
	}
}

// rankedBest requests alternatives and applies the exact same scoring.Rank used
// by the production route selection (httpapi.bestRoute), so any distance
// difference found by Explain reflects an actual vehicle constraint - not just
// our own risk-scoring formula preferring a different route than Valhalla's raw
// top pick would. That mismatch was a real bug caught while testing this
// feature: comparing a scored/ranked "chosen" route against a raw, unscored
// reference made Explain blame vehicle dimensions for distance differences that
// were actually caused by our own scoring formula's highway-ratio preference
// (see documentations/features/2026-07-21-route-explainability.md).
func (e *Explainer) rankedBest(ctx context.Context, origin, destination valhalla.LatLon, profile valhalla.TruckProfile, prefs scoring.Preferences, vehicleWeightKg float64, preferredStops []valhalla.LatLon) (valhalla.RouteCandidate, error) {
	candidates, err := e.Valhalla.RouteAlternates(ctx, origin, destination, profile, numAlternates)
	if err != nil {
		return valhalla.RouteCandidate{}, err
	}
	return scoring.Rank(candidates, prefs, vehicleWeightKg, preferredStops)[0].RouteCandidate, nil
}

// Explain returns a driver-facing explanation string, or nil if the chosen route
// doesn't meaningfully detour from what an unconstrained vehicle would take.
// A Valhalla error while probing is treated as "no explanation available", not
// a fatal error for the caller - this is a nice-to-have, not a hard requirement
// for the route response. prefs/vehicleWeightKg/preferredStops must be the same
// values used to pick `chosen`, so the reference/probe routes are ranked by the
// identical formula - see the rankedBest doc comment for why that matters.
func (e *Explainer) Explain(ctx context.Context, origin, destination valhalla.LatLon, profile valhalla.TruckProfile, chosen valhalla.RouteCandidate, prefs scoring.Preferences, vehicleWeightKg float64, preferredStops []valhalla.LatLon) *string {
	ref, err := e.rankedBest(ctx, origin, destination, referenceProfile(), prefs, vehicleWeightKg, preferredStops)
	if err != nil {
		return nil
	}

	if math.Abs(chosen.DistanceKm-ref.DistanceKm) < distanceToleranceKm {
		return nil
	}

	location := firstDivergentStreetName(chosen.StreetNames, ref.StreetNames)

	for _, d := range dimensions() {
		relaxed := d.relax(profile)
		result, err := e.rankedBest(ctx, origin, destination, relaxed, prefs, vehicleWeightKg, preferredStops)
		if err != nil {
			continue
		}
		if math.Abs(result.DistanceKm-ref.DistanceKm) < distanceToleranceKm {
			msg := fmt.Sprintf("Ruta skreće kod %s jer %s ne zadovoljava ograničenje na toj deonici.", location, d.value(profile))
			return &msg
		}
	}

	msg := fmt.Sprintf("Ruta odstupa od najkraćeg puta kod %s zbog ograničenja koje vaše vozilo ne zadovoljava.", location)
	return &msg
}

// firstDivergentStreetName finds where the chosen route's maneuver street names
// stop matching the reference route's, maneuver-by-maneuver - an approximation of
// "where the detour starts" using only data already available from /route
// responses (see documentations/features/2026-07-21-risk-scoring-layer.md for why
// per-edge attribution like bridge/tunnel isn't available here).
func firstDivergentStreetName(chosen, reference []string) string {
	n := len(chosen)
	if len(reference) < n {
		n = len(reference)
	}
	for i := 0; i < n; i++ {
		if chosen[i] != reference[i] {
			if chosen[i] != "" {
				return chosen[i]
			}
			return "jednoj deonici puta"
		}
	}
	return "jednoj deonici puta"
}
