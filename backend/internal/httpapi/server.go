package httpapi

import (
	"encoding/json"
	"net/http"

	"heavy-vehicle-routing/backend/internal/auth"
	"heavy-vehicle-routing/backend/internal/explain"
	"heavy-vehicle-routing/backend/internal/geocode"
	"heavy-vehicle-routing/backend/internal/mailer"
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
	Valhalla           *valhalla.Client
	Vehicles           *store.VehicleStore
	Trips              *store.TripStore
	TripEvents         *store.TripEventStore
	Drivers            *store.DriverStore
	DispatcherRequests *store.DispatcherRequestStore
	Preferences        *store.PreferencesStore
	FavoriteStops      *store.FavoriteStopStore
	Chats              *store.ChatMessageStore
	EmailVerifications *store.EmailVerificationTokenStore
	PasswordResets     *store.PasswordResetTokenStore
	Geocoder           *geocode.Client
	RestStops          *reststop.Finder
	Queue              *queue.Client
	Explain            *explain.Explainer
	WS                 *ws.Gateway
	ChatWS             *ws.ChatGateway
	Auth               *auth.Manager
	GoogleAuth         *auth.GoogleVerifier
	Mailer             *mailer.Client
	PublicBackendURL   string
}

func NewServer(v *valhalla.Client, vehicles *store.VehicleStore, trips *store.TripStore, tripEvents *store.TripEventStore, drivers *store.DriverStore, dispatcherRequests *store.DispatcherRequestStore, preferences *store.PreferencesStore, favoriteStops *store.FavoriteStopStore, chats *store.ChatMessageStore, emailVerifications *store.EmailVerificationTokenStore, passwordResets *store.PasswordResetTokenStore, restStops *reststop.Finder, q *queue.Client, ex *explain.Explainer, wsGateway *ws.Gateway, chatWS *ws.ChatGateway, authManager *auth.Manager, googleAuth *auth.GoogleVerifier, mailerClient *mailer.Client, geocoder *geocode.Client, publicBackendURL string) *Server {
	return &Server{
		Valhalla: v, Vehicles: vehicles, Trips: trips, TripEvents: tripEvents, Drivers: drivers,
		DispatcherRequests: dispatcherRequests,
		Preferences:        preferences, FavoriteStops: favoriteStops, Chats: chats, EmailVerifications: emailVerifications, PasswordResets: passwordResets, RestStops: restStops,
		Queue: q, Explain: ex, WS: wsGateway, ChatWS: chatWS, Auth: authManager,
		GoogleAuth: googleAuth, Mailer: mailerClient, Geocoder: geocoder, PublicBackendURL: publicBackendURL,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/google", s.handleGoogleAuth)
	mux.HandleFunc("GET /api/v1/auth/verify-email", s.handleVerifyEmail)
	mux.HandleFunc("POST /api/v1/auth/resend-verification", s.RequireAuth(s.handleResendVerification))
	mux.HandleFunc("GET /api/v1/auth/me", s.RequireAuth(s.handleMe))
	mux.HandleFunc("POST /api/v1/auth/forgot-password", s.handleForgotPassword)
	mux.HandleFunc("GET /api/v1/auth/reset-password", s.handleShowResetPasswordForm)
	mux.HandleFunc("POST /api/v1/auth/reset-password", s.handleSubmitResetPassword)
	mux.HandleFunc("POST /api/v1/auth/logout-all", s.RequireAuth(s.handleLogoutAll))

	mux.HandleFunc("POST /api/v1/routes", s.RequireAuth(s.handleCreateRoute))
	mux.HandleFunc("GET /api/v1/geocode", s.RequireAuth(s.handleGeocode))
	mux.HandleFunc("GET /api/v1/geocode/reverse", s.RequireAuth(s.handleReverseGeocode))
	mux.HandleFunc("POST /api/v1/vehicles", s.RequireAuth(s.handleCreateVehicle))
	mux.HandleFunc("GET /api/v1/vehicles", s.RequireAuth(s.handleListVehicles))
	mux.HandleFunc("GET /api/v1/vehicles/{id}", s.RequireAuth(s.handleGetVehicle))
	mux.HandleFunc("PUT /api/v1/vehicles/{id}", s.RequireAuth(s.handleUpdateVehicle))
	mux.HandleFunc("DELETE /api/v1/vehicles/{id}", s.RequireAuth(s.handleDeleteVehicle))
	mux.HandleFunc("PATCH /api/v1/vehicles/{id}/status", s.RequireAuth(s.handleUpdateVehicleStatus))
	mux.HandleFunc("GET /api/v1/vehicles/{id}/hours", s.RequireAuth(s.handleGetVehicleHours))
	mux.HandleFunc("POST /api/v1/trips", s.RequireAuth(s.handleCreateTrip))
	mux.HandleFunc("GET /api/v1/trips", s.RequireAuth(s.handleListTrips))
	mux.HandleFunc("GET /api/v1/trips/{id}", s.RequireAuth(s.handleGetTrip))
	mux.HandleFunc("POST /api/v1/trips/{id}/accept", s.RequireAuth(s.handleAcceptTrip))
	mux.HandleFunc("POST /api/v1/trips/{id}/reject", s.RequireAuth(s.handleRejectTrip))
	mux.HandleFunc("POST /api/v1/trips/{id}/start", s.RequireAuth(s.handleStartTrip))
	mux.HandleFunc("POST /api/v1/trips/{id}/position", s.RequireAuth(s.handleReportPosition))
	mux.HandleFunc("POST /api/v1/trips/{id}/complete", s.RequireAuth(s.handleCompleteTrip))
	mux.HandleFunc("GET /api/v1/trips/{id}/events", s.RequireAuth(s.handleListTripEvents))
	mux.HandleFunc("GET /ws/trips/{id}", s.RequireAuthQuery(s.handleTripStream))
	mux.HandleFunc("GET /api/v1/dispatcher/drivers", s.RequireAuth(s.handleListManagedDrivers))
	mux.HandleFunc("GET /api/v1/dispatcher/drivers/{id}/vehicles", s.RequireAuth(s.handleListDriverVehiclesForDispatcher))
	mux.HandleFunc("GET /api/v1/dispatcher/available-drivers", s.RequireAuth(s.handleListAvailableDrivers))
	mux.HandleFunc("POST /api/v1/dispatcher/requests", s.RequireAuth(s.handleCreateDispatcherRequest))
	mux.HandleFunc("GET /api/v1/dispatcher/requests", s.RequireAuth(s.handleListDispatcherRequests))
	mux.HandleFunc("GET /api/v1/driver/requests", s.RequireAuth(s.handleListDriverRequests))
	mux.HandleFunc("POST /api/v1/driver/requests/{id}/respond", s.RequireAuth(s.handleRespondDispatcherRequest))
	mux.HandleFunc("POST /api/v1/driver/leave-dispatcher", s.RequireAuth(s.handleLeaveDispatcher))
	mux.HandleFunc("GET /api/v1/preferences", s.RequireAuth(s.handleGetPreferences))
	mux.HandleFunc("PUT /api/v1/preferences", s.RequireAuth(s.handleUpdatePreferences))
	mux.HandleFunc("POST /api/v1/favorite-stops", s.RequireAuth(s.handleCreateFavoriteStop))
	mux.HandleFunc("GET /api/v1/favorite-stops", s.RequireAuth(s.handleListFavoriteStops))
	mux.HandleFunc("DELETE /api/v1/favorite-stops/{id}", s.RequireAuth(s.handleDeleteFavoriteStop))
	mux.HandleFunc("GET /api/v1/drivers", s.RequireAuth(s.handleListDrivers))
	mux.HandleFunc("GET /api/v1/chats", s.RequireAuth(s.handleListChats))
	mux.HandleFunc("GET /api/v1/chats/{driverId}/messages", s.RequireAuth(s.handleGetChatMessages))
	mux.HandleFunc("POST /api/v1/chats/{driverId}/messages", s.RequireAuth(s.handleSendChatMessage))
	mux.HandleFunc("GET /ws/chats/{counterpartId}", s.RequireAuthQuery(s.handleChatStream))
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
