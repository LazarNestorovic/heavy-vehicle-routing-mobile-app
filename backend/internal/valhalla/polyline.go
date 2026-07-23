package valhalla

import "math"

// DecodePolyline6 decodes Valhalla's default shape encoding (Google's polyline
// algorithm at 1e-6 precision, i.e. "polyline6").
func DecodePolyline6(encoded string) []LatLon {
	var coords []LatLon
	index, lat, lon := 0, 0, 0
	for index < len(encoded) {
		lat += decodeValue(encoded, &index)
		lon += decodeValue(encoded, &index)
		coords = append(coords, LatLon{Lat: float64(lat) / 1e6, Lon: float64(lon) / 1e6})
	}
	return coords
}

func decodeValue(encoded string, index *int) int {
	shift, result := 0, 0
	for {
		b := int(encoded[*index]) - 63
		*index++
		result |= (b & 0x1f) << shift
		shift += 5
		if b < 0x20 {
			break
		}
	}
	if result&1 != 0 {
		return ^(result >> 1)
	}
	return result >> 1
}

// PointAtFraction walks the decoded shape and returns the point at the given
// fraction (0..1) of its total length - used to approximate "where will the
// vehicle be after X% of the trip", assuming roughly constant speed. That's a
// simplification (real speed varies by road segment); good enough for a rest-stop
// suggestion, see documentations/features/2026-07-21-rest-stop-locations.md.
func PointAtFraction(points []LatLon, fraction float64) LatLon {
	if len(points) == 0 {
		return LatLon{}
	}
	if fraction <= 0 {
		return points[0]
	}
	if fraction >= 1 {
		return points[len(points)-1]
	}

	segLengths := make([]float64, len(points)-1)
	total := 0.0
	for i := 0; i < len(points)-1; i++ {
		segLengths[i] = haversineMeters(points[i], points[i+1])
		total += segLengths[i]
	}
	if total == 0 {
		return points[0]
	}

	target := total * fraction
	cum := 0.0
	for i, segLen := range segLengths {
		if cum+segLen >= target && segLen > 0 {
			segFraction := (target - cum) / segLen
			return LatLon{
				Lat: points[i].Lat + (points[i+1].Lat-points[i].Lat)*segFraction,
				Lon: points[i].Lon + (points[i+1].Lon-points[i].Lon)*segFraction,
			}
		}
		cum += segLen
	}
	return points[len(points)-1]
}

const earthRadiusM = 6371000.0

func haversineMeters(a, b LatLon) float64 {
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }
	dLat := toRad(b.Lat - a.Lat)
	dLon := toRad(b.Lon - a.Lon)
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(a.Lat))*math.Cos(toRad(b.Lat))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusM * 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}
