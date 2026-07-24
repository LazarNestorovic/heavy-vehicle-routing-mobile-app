package httpapi

import (
	"encoding/json"
	"net/http"

	"heavy-vehicle-routing/backend/internal/auth"
	"heavy-vehicle-routing/backend/internal/explain"
	"heavy-vehicle-routing/backend/internal/queue"
	"heavy-vehicle-routing/backend/internal/reststop"
	"heavy-vehicle-routing/backend/internal/store"
	"heavy-vehicle-routing/backend/internal/valhalla"
	"heavy-vehicle-routing/backend/internal/ws"
)

// numAlternates is how many extra Valhalla route alternatives we ask for and score
// alongside the primary route, in addition to the primary route itself.
const numAlternates = 2

type Server struct {
	Valhalla      *valhalla.Client
	Vehicles      *store.VehicleStore
	Trips         *store.TripStore
	Drivers       *store.DriverStore
	Preferences   *store.PreferencesStore
	FavoriteStops *store.FavoriteStopStore
	RestStops     *reststop.Finder
	Queue         *queue.Client
	Explain       *explain.Explainer
	WS            *ws.Gateway
	Auth          *auth.Manager
}

func NewServer(v *valhalla.Client, vehicles *store.VehicleStore, trips *store.TripStore, drivers *store.DriverStore, preferences *store.PreferencesStore, favoriteStops *store.FavoriteStopStore, restStops *reststop.Finder, q *queue.Client, ex *explain.Explainer, wsGateway *ws.Gateway, authManager *auth.Manager) *Server {
	return &Server{
		Valhalla: v, Vehicles: vehicles, Trips: trips, Drivers: drivers,
		Preferences: preferences, FavoriteStops: favoriteStops, RestStops: restStops,
		Queue: q, Explain: ex, WS: wsGateway, Auth: authManager,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)

	mux.HandleFunc("POST /api/v1/routes", s.RequireAuth(s.handleCreateRoute))
	mux.HandleFunc("POST /api/v1/vehicles", s.RequireAuth(s.handleCreateVehicle))
	mux.HandleFunc("GET /api/v1/vehicles", s.RequireAuth(s.handleListVehicles))
	mux.HandleFunc("GET /api/v1/vehicles/{id}", s.RequireAuth(s.handleGetVehicle))
	mux.HandleFunc("POST /api/v1/trips", s.RequireAuth(s.handleCreateTrip))
	mux.HandleFunc("GET /api/v1/trips/{id}", s.RequireAuth(s.handleGetTrip))
	mux.HandleFunc("GET /ws/trips/{id}", s.RequireAuthQuery(s.handleTripStream))
	mux.HandleFunc("GET /api/v1/preferences", s.RequireAuth(s.handleGetPreferences))
	mux.HandleFunc("PUT /api/v1/preferences", s.RequireAuth(s.handleUpdatePreferences))
	mux.HandleFunc("POST /api/v1/favorite-stops", s.RequireAuth(s.handleCreateFavoriteStop))
	mux.HandleFunc("GET /api/v1/favorite-stops", s.RequireAuth(s.handleListFavoriteStops))
	mux.HandleFunc("DELETE /api/v1/favorite-stops/{id}", s.RequireAuth(s.handleDeleteFavoriteStop))
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
