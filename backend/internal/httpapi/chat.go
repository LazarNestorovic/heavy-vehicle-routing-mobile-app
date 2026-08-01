package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"heavy-vehicle-routing/backend/internal/queue"
	"heavy-vehicle-routing/backend/internal/store"
)

type driverResponse struct {
	ID       int64   `json:"id"`
	Username string  `json:"username"`
	Email    *string `json:"email,omitempty"`
}

// handleListDrivers is the "start a new chat" contact list - every other
// registered account (driver or dispatcher), since this project has no
// fleet/team concept restricting who can message whom. Optional query-string
// `q` filters by a case-insensitive substring of username or email (see
// documentations/features/ entry for the chat contact search + pinning
// feature).
func (s *Server) handleListDrivers(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	drivers, err := s.Drivers.List(r.Context(), driverID, r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list drivers: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDriverResponses(drivers))
}

type chatConversationResponse struct {
	CounterpartID       int64     `json:"counterpart_id"`
	CounterpartUsername string    `json:"counterpart_username"`
	LastMessage         string    `json:"last_message"`
	LastMessageAt       time.Time `json:"last_message_at"`
	UnreadCount         int       `json:"unread_count"`
}

func (s *Server) handleListChats(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	conversations, err := s.Chats.ListConversations(r.Context(), driverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list chats: "+err.Error())
		return
	}

	out := make([]chatConversationResponse, len(conversations))
	for i, c := range conversations {
		out[i] = chatConversationResponse{
			CounterpartID:       c.CounterpartID,
			CounterpartUsername: c.CounterpartUsername,
			LastMessage:         c.LastMessage,
			LastMessageAt:       c.LastMessageAt,
			UnreadCount:         c.UnreadCount,
		}
	}
	writeJSON(w, http.StatusOK, out)
}

type chatMessageResponse struct {
	ID           int64      `json:"id"`
	FromDriverID int64      `json:"from_driver_id"`
	ToDriverID   int64      `json:"to_driver_id"`
	Body         string     `json:"body"`
	SentAt       time.Time  `json:"sent_at"`
	ReadAt       *time.Time `json:"read_at,omitempty"`
}

func toChatMessageResponse(m store.ChatMessage) chatMessageResponse {
	return chatMessageResponse{
		ID: m.ID, FromDriverID: m.FromDriverID, ToDriverID: m.ToDriverID,
		Body: m.Body, SentAt: m.SentAt, ReadAt: m.ReadAt,
	}
}

// handleGetChatMessages returns the full thread with counterpartID and marks
// every message they sent as read (this is the driver opening the thread).
func (s *Server) handleGetChatMessages(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	counterpartID, err := strconv.ParseInt(r.PathValue("driverId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid driver id")
		return
	}

	messages, err := s.Chats.ListThread(r.Context(), driverID, counterpartID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load chat thread: "+err.Error())
		return
	}
	if err := s.Chats.MarkRead(r.Context(), driverID, counterpartID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark chat read: "+err.Error())
		return
	}

	out := make([]chatMessageResponse, len(messages))
	for i, m := range messages {
		out[i] = toChatMessageResponse(m)
	}
	writeJSON(w, http.StatusOK, out)
}

type sendChatMessageRequest struct {
	Body string `json:"body"`
}

func (s *Server) handleSendChatMessage(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	counterpartID, err := strconv.ParseInt(r.PathValue("driverId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid driver id")
		return
	}

	var req sendChatMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Body) == "" {
		writeError(w, http.StatusBadRequest, "body must not be empty")
		return
	}

	saved, err := s.Chats.Create(r.Context(), driverID, counterpartID, req.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save chat message: "+err.Error())
		return
	}

	// The message is already persisted (source of truth); a queue hiccup here
	// shouldn't fail the request - same reasoning as trip.started.
	event := queue.ChatMessageEvent{
		ID: saved.ID, FromDriverID: saved.FromDriverID, ToDriverID: saved.ToDriverID,
		Body: saved.Body, SentAt: saved.SentAt,
	}
	if body, err := json.Marshal(event); err != nil {
		log.Printf("encode chat message %d: %v", saved.ID, err)
	} else if err := s.Queue.PublishChat(r.Context(), queue.ChatRoutingKey(driverID, counterpartID), body); err != nil {
		log.Printf("publish chat message %d: %v", saved.ID, err)
	}

	writeJSON(w, http.StatusCreated, toChatMessageResponse(saved))
}

func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request) {
	driverID, _ := driverIDFromContext(r.Context())

	counterpartID, err := strconv.ParseInt(r.PathValue("counterpartId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid driver id")
		return
	}

	s.ChatWS.HandleChatStream(w, r, driverID, counterpartID)
}
