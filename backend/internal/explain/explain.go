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
	"sync"
	"time"

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

// referenceCacheTTL is how long a computed reference route (the unconstrained
// "what would the shortest path be with no vehicle restrictions" probe) is
// reused for repeated Explain calls on the same origin/destination - avoids
// an extra Valhalla round-trip when a driver previews the same route more
// than once in a short window (see documentations/features/ entry).
//
// Known simplification: the cache key is origin/destination only, not driver
// preferences - the reference route is technically also ranked by
// scoring.Rank (see rankedBest), so two different drivers/dispatchers
// requesting the same coordinates within the TTL could get a reference route
// ranked by the FIRST caller's preferences. Accepted the same way as this
// project's other documented approximations: the common case (one driver
// re-previewing the same route) benefits, the rare cross-driver coincidence
// costs nothing but a slightly-off "reference" that's still the same
// distance ballpark.
const referenceCacheTTL = 5 * time.Minute

type referenceCacheKey struct {
	originLat, originLon, destLat, destLon float64
}

type referenceCacheEntry struct {
	candidate valhalla.RouteCandidate
	expiresAt time.Time
}

type Explainer struct {
	Valhalla *valhalla.Client

	cacheMu sync.Mutex
	cache   map[referenceCacheKey]referenceCacheEntry
}

func New(v *valhalla.Client) *Explainer {
	return &Explainer{Valhalla: v, cache: make(map[referenceCacheKey]referenceCacheEntry)}
}

// roundCoord rounds to 4 decimal places (~11m) so origin/destination pairs
// that differ only by floating-point noise still hit the same cache entry.
func roundCoord(v float64) float64 {
	return math.Round(v*10000) / 10000
}

// cachedReference returns the reference-profile route for (origin, destination),
// reusing a cached one from within referenceCacheTTL if available.
func (e *Explainer) cachedReference(ctx context.Context, origin, destination valhalla.LatLon, prefs scoring.Preferences, vehicleWeightKg float64, preferredStops []valhalla.LatLon) (valhalla.RouteCandidate, error) {
	key := referenceCacheKey{roundCoord(origin.Lat), roundCoord(origin.Lon), roundCoord(destination.Lat), roundCoord(destination.Lon)}

	e.cacheMu.Lock()
	if entry, ok := e.cache[key]; ok && time.Now().Before(entry.expiresAt) {
		e.cacheMu.Unlock()
		return entry.candidate, nil
	}
	e.cacheMu.Unlock()

	ref, err := e.rankedBest(ctx, origin, destination, referenceProfile(), prefs, vehicleWeightKg, preferredStops)
	if err != nil {
		return valhalla.RouteCandidate{}, err
	}

	e.cacheMu.Lock()
	e.cache[key] = referenceCacheEntry{candidate: ref, expiresAt: time.Now().Add(referenceCacheTTL)}
	e.cacheMu.Unlock()

	return ref, nil
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
	ref, err := e.cachedReference(ctx, origin, destination, prefs, vehicleWeightKg, preferredStops)
	if err != nil {
		return nil
	}

	if math.Abs(chosen.DistanceKm-ref.DistanceKm) < distanceToleranceKm {
		return nil
	}

	location := divergenceStreetName(chosen, ref)

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

// divergenceThresholdM: a point on the chosen route farther than this from
// EVERY point on the reference route is considered "off the reference path" -
// the geometric analogue of "the routes have diverged here".
const divergenceThresholdM = 200.0

// shapeSampleStride bounds the cost of the geometric divergence walk (same
// idea, and same value, as scoring.go's constant of the same name).
const shapeSampleStride = 5

// divergenceStreetName finds where the chosen route's actual GEOMETRY (its
// decoded shape) stops overlapping the reference route's, then returns the
// street name of the chosen route's maneuver whose starting point is closest
// to that divergence point.
//
// This replaced an earlier version (firstDivergentStreetName) that compared
// StreetNames maneuver-by-maneuver BY LIST POSITION (chosen[i] vs
// reference[i]). That worked when the two routes shared a common prefix and
// diverged locally, but for routes that are globally different from the very
// first maneuver (a heavily constrained vehicle can make Valhalla pick a
// completely different overall strategy), position-based comparison reported
// a location near the START of the route, not near the actual obstacle - see
// documentations/features/2026-07-21-route-explainability.md's "Poznato
// ograničenje" section. Real geometry doesn't have that failure mode: it
// finds where the paths physically stop overlapping, independent of how
// their maneuver lists happen to be indexed.
func divergenceStreetName(chosen, reference valhalla.RouteCandidate) string {
	const fallback = "jednoj deonici puta"

	chosenPoints := valhalla.DecodePolyline6(chosen.Shape)
	refPoints := valhalla.DecodePolyline6(reference.Shape)
	if len(chosenPoints) == 0 || len(refPoints) == 0 {
		return fallback
	}

	divergedAt, found := divergencePoint(chosenPoints, refPoints)
	if !found {
		return fallback
	}

	if name := nearestManeuverStreetName(divergedAt, chosen); name != "" {
		return name
	}
	return fallback
}

// divergencePoint walks `from`'s shape (sampled every shapeSampleStride
// points, for performance) looking for the first point whose nearest point on
// `to`'s shape exceeds divergenceThresholdM - an approximation of "where
// these two paths stop being the same road", cheap enough to run on full-
// resolution route shapes without a spatial index (same performance
// trade-off already accepted elsewhere in this project, e.g. scoring.go's
// preferredStopDiscount).
func divergencePoint(from, to []valhalla.LatLon) (valhalla.LatLon, bool) {
	for i := 0; i < len(from); i += shapeSampleStride {
		if nearestDistanceM(from[i], to) > divergenceThresholdM {
			return from[i], true
		}
	}
	return valhalla.LatLon{}, false
}

func nearestDistanceM(p valhalla.LatLon, points []valhalla.LatLon) float64 {
	best := math.Inf(1)
	for i := 0; i < len(points); i += shapeSampleStride {
		if d := haversineMeters(p, points[i]); d < best {
			best = d
		}
	}
	return best
}

// nearestManeuverStreetName returns the street name of route's maneuver whose
// ManeuverPoints entry is closest to target - "" if route has no maneuvers
// with a resolved point, or if the nearest one is itself unnamed.
func nearestManeuverStreetName(target valhalla.LatLon, route valhalla.RouteCandidate) string {
	bestDist := math.Inf(1)
	bestName := ""
	for i, p := range route.ManeuverPoints {
		if p == (valhalla.LatLon{}) {
			continue // begin_shape_index couldn't be resolved for this maneuver
		}
		if d := haversineMeters(target, p); d < bestDist {
			bestDist = d
			if i < len(route.StreetNames) {
				bestName = route.StreetNames[i]
			}
		}
	}
	return bestName
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
