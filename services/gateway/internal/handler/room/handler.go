package room

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sudobytemebaby/efir/services/gateway/internal/client"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler"
	"github.com/sudobytemebaby/efir/services/gateway/internal/middleware"
)

type Handler struct {
	roomClient *client.RoomClient
}

func NewHandler(roomClient *client.RoomClient) *Handler {
	return &Handler{
		roomClient: roomClient,
	}
}

func (h *Handler) Register(r chi.Router) {
	r.Post("/rooms", h.createRoom)
	r.Get("/rooms/{id}", h.getRoom)
	r.Patch("/rooms/{id}", h.updateRoom)
	r.Delete("/rooms/{id}", h.deleteRoom)
	r.Post("/rooms/{id}/members", h.addMember)
	r.Delete("/rooms/{id}/members/{userId}", h.removeMember)
}

func mapRoomTypeToString(rt client.RoomType) string {
	switch rt {
	case client.RoomTypeDirect:
		return "direct"
	case client.RoomTypeGroup:
		return "group"
	default:
		return "unspecified"
	}
}

func mapRoomToResponse(room *client.Room) roomResponse {
	return roomResponse{
		RoomID:    room.RoomId,
		Name:      room.Name,
		Type:      mapRoomTypeToString(room.Type),
		CreatedBy: room.CreatedBy,
		CreatedAt: handler.TimestampToString(room.CreatedAt),
		UpdatedAt: handler.TimestampToString(room.UpdatedAt),
	}
}

func (h *Handler) createRoom(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var roomType client.RoomType
	switch req.Type {
	case "direct":
		roomType = client.RoomTypeDirect
	case "group":
		roomType = client.RoomTypeGroup
	default:
		http.Error(w, "invalid room type", http.StatusBadRequest)
		return
	}

	resp, err := h.roomClient.CreateRoom(middleware.InjectRequestIDToOutgoingContext(r.Context()), req.Name, roomType, requesterID, req.ParticipantID)
	if err != nil {
		handler.WriteError(w, r, err, "failed to create room")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(mapRoomToResponse(resp.Room)); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}

func (h *Handler) getRoom(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing room id", http.StatusBadRequest)
		return
	}

	resp, err := h.roomClient.GetRoom(middleware.InjectRequestIDToOutgoingContext(r.Context()), id)
	if err != nil {
		handler.WriteError(w, r, err, "failed to get room")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(mapRoomToResponse(resp.Room)); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}

func (h *Handler) updateRoom(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing room id", http.StatusBadRequest)
		return
	}

	var req updateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.roomClient.UpdateRoom(middleware.InjectRequestIDToOutgoingContext(r.Context()), id, requesterID, req.Name)
	if err != nil {
		handler.WriteError(w, r, err, "failed to update room")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(mapRoomToResponse(resp.Room)); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}

func (h *Handler) deleteRoom(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing room id", http.StatusBadRequest)
		return
	}

	_, err := h.roomClient.DeleteRoom(middleware.InjectRequestIDToOutgoingContext(r.Context()), id, requesterID)
	if err != nil {
		handler.WriteError(w, r, err, "failed to delete room")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	roomID := chi.URLParam(r, "id")
	if roomID == "" {
		http.Error(w, "missing room id", http.StatusBadRequest)
		return
	}

	var req memberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	_, err := h.roomClient.AddMember(middleware.InjectRequestIDToOutgoingContext(r.Context()), roomID, req.UserID, requesterID)
	if err != nil {
		handler.WriteError(w, r, err, "failed to add member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	roomID := chi.URLParam(r, "id")
	if roomID == "" {
		http.Error(w, "missing room id", http.StatusBadRequest)
		return
	}

	userID := chi.URLParam(r, "userId")
	if userID == "" {
		http.Error(w, "missing user id", http.StatusBadRequest)
		return
	}

	_, err := h.roomClient.RemoveMember(middleware.InjectRequestIDToOutgoingContext(r.Context()), roomID, userID, requesterID)
	if err != nil {
		handler.WriteError(w, r, err, "failed to remove member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
