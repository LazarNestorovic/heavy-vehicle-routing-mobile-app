package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"heavy-vehicle-routing/backend/internal/store"
)

type favoriteStopRequest struct {
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Name string  `json:"name"`
}

type favoriteStopResponse struct {
	ID   int64   `json:"id"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Name string  `json:"name"`
}

func toFavoriteStopResponse(f store.FavoriteStop) favoriteStopResponse {
	return favoriteStopResponse{ID: f.ID, Lat: f.Lat, Lon: f.Lon, Name: f.Name}
}

func (s *Server) handleCreateFavoriteStop(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	var req favoriteStopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	saved, err := s.FavoriteStops.Create(r.Context(), store.FavoriteStop{
		DriverID: driverID, Lat: req.Lat, Lon: req.Lon, Name: req.Name,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save favorite stop: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toFavoriteStopResponse(saved))
}

func (s *Server) handleListFavoriteStops(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	stops, err := s.FavoriteStops.List(r.Context(), driverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list favorite stops: "+err.Error())
		return
	}

	out := make([]favoriteStopResponse, len(stops))
	for i, f := range stops {
		out[i] = toFavoriteStopResponse(f)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleDeleteFavoriteStop(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid favorite stop id")
		return
	}

	if err := s.FavoriteStops.Delete(r.Context(), id, driverID); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "favorite stop not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete favorite stop: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
