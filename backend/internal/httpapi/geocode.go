package httpapi

import (
	"net/http"
	"strconv"
)

type geocodeResultResponse struct {
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	DisplayName string  `json:"display_name"`
}

// handleGeocode is a thin authenticated pass-through to internal/geocode
// (Nominatim) - see documentations/features/ entry for why address search
// went through Nominatim rather than Google's paid Geocoding API. limit is
// clamped to a small range; this is a tap-to-search convenience, not a bulk
// data endpoint.
func (s *Server) handleGeocode(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "q is required")
		return
	}

	limit := 5
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 10 {
			limit = n
		}
	}

	results, err := s.Geocoder.Search(r.Context(), q, limit)
	if err != nil {
		writeError(w, http.StatusBadGateway, "geocoding failed: "+err.Error())
		return
	}

	out := make([]geocodeResultResponse, len(results))
	for i, res := range results {
		out[i] = geocodeResultResponse{Lat: res.Lat, Lon: res.Lon, DisplayName: res.DisplayName}
	}
	writeJSON(w, http.StatusOK, out)
}
