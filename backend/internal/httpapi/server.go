package httpapi

import (
	"encoding/json"
	"net/http"

	"heavy-vehicle-routing/backend/internal/explain"
	"heavy-vehicle-routing/backend/internal/queue"
	"heavy-vehicle-routing/backend/internal/store"
	"heavy-vehicle-routing/backend/internal/valhalla"
	"heavy-vehicle-routing/backend/internal/ws"
)

// numAlternates is how many extra Valhalla route alternatives we ask for and score
// alongside the primary route, in addition to the primary route itself.
const numAlternates = 2

type Server struct {
	Valhalla *valhalla.Client
	Vehicles *store.VehicleStore
	Trips    *store.TripStore
	Queue    *queue.Client
	Explain  *explain.Explainer
	WS       *ws.Gateway
}

func NewServer(v *valhalla.Client, vehicles *store.VehicleStore, trips *store.TripStore, q *queue.Client, ex *explain.Explainer, wsGateway *ws.Gateway) *Server {
	return &Server{Valhalla: v, Vehicles: vehicles, Trips: trips, Queue: q, Explain: ex, WS: wsGateway}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /api/v1/routes", s.handleCreateRoute)
	mux.HandleFunc("POST /api/v1/vehicles", s.handleCreateVehicle)
	mux.HandleFunc("GET /api/v1/vehicles/{id}", s.handleGetVehicle)
	mux.HandleFunc("POST /api/v1/trips", s.handleCreateTrip)
	mux.HandleFunc("GET /api/v1/trips/{id}", s.handleGetTrip)
	mux.HandleFunc("GET /ws/trips/{id}", s.WS.HandleTripStream)
	return mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
