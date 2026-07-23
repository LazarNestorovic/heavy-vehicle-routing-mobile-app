package algorithm

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

// cost is the edge weight for an allowed edge: length in meters, with a mild
// penalty for poor/unpaved surfaces. This is deliberately simple (see
// SPECIFIKACIJA.md 3.3.2) - the point of this module is the constraint-aware
// search itself, not a finely tuned cost model.
func cost(e Edge) float64 {
	switch e.Surface {
	case "unpaved", "gravel", "dirt", "sett":
		return e.LengthM * 1.3
	default:
		return e.LengthM
	}
}
