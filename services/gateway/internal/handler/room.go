package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sudobytemebaby/efir/services/gateway/internal/client"
	"github.com/sudobytemebaby/efir/services/gateway/internal/middleware"
	"github.com/sudobytemebaby/efir/services/shared/pkg/errors"
)

type RoomHandler struct {
	roomClient *client.RoomClient
}

func NewRoomHandler(roomClient *client.RoomClient) *RoomHandler {
	return &RoomHandler{
		roomClient: roomClient,
	}
}

func (h *RoomHandler) Register(r chi.Router) {
	r.Post("/rooms", h.createRoom)
	r.Get("/rooms/{id}", h.getRoom)
	r.Patch("/rooms/{id}", h.updateRoom)
	r.Delete("/rooms/{id}", h.deleteRoom)
	r.Post("/rooms/{id}/members", h.addMember)
	r.Delete("/rooms/{id}/members/{userId}", h.removeMember)
}

type createRoomRequest struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	ParticipantID string `json:"participant_id,omitempty"`
}

type roomResponse struct {
	RoomID    string `json:"room_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func roomTypeToString(rt client.RoomType) string {
	switch rt {
	case client.RoomTypeDirect:
		return "direct"
	case client.RoomTypeGroup:
		return "group"
	default:
		return "unspecified"
	}
}

func roomToResponse(room *client.Room) roomResponse {
	return roomResponse{
		RoomID:    room.RoomId,
		Name:      room.Name,
		Type:      roomTypeToString(room.Type),
		CreatedBy: room.CreatedBy,
		CreatedAt: timestampToString(room.CreatedAt),
		UpdatedAt: timestampToString(room.UpdatedAt),
	}
}

func (h *RoomHandler) createRoom(w http.ResponseWriter, r *http.Request) {
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

	resp, err := h.roomClient.CreateRoom(r.Context(), req.Name, roomType, requesterID, req.ParticipantID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create room", "error", err)
		code := errors.FromError(err)
		http.Error(w, err.Error(), code.ToHTTPCode())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(roomToResponse(resp.Room)); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}

func (h *RoomHandler) getRoom(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing room id", http.StatusBadRequest)
		return
	}

	resp, err := h.roomClient.GetRoom(r.Context(), id)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get room", "error", err)
		code := errors.FromError(err)
		http.Error(w, err.Error(), code.ToHTTPCode())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(roomToResponse(resp.Room)); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}

type updateRoomRequest struct {
	Name *string `json:"name,omitempty"`
}

func (h *RoomHandler) updateRoom(w http.ResponseWriter, r *http.Request) {
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

	resp, err := h.roomClient.UpdateRoom(r.Context(), id, requesterID, req.Name)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to update room", "error", err)
		code := errors.FromError(err)
		http.Error(w, err.Error(), code.ToHTTPCode())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(roomToResponse(resp.Room)); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}

func (h *RoomHandler) deleteRoom(w http.ResponseWriter, r *http.Request) {
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

	_, err := h.roomClient.DeleteRoom(r.Context(), id, requesterID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to delete room", "error", err)
		code := errors.FromError(err)
		http.Error(w, err.Error(), code.ToHTTPCode())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type memberRequest struct {
	UserID string `json:"user_id"`
}

func (h *RoomHandler) addMember(w http.ResponseWriter, r *http.Request) {
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

	_, err := h.roomClient.AddMember(r.Context(), roomID, req.UserID, requesterID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to add member", "error", err)
		code := errors.FromError(err)
		http.Error(w, err.Error(), code.ToHTTPCode())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *RoomHandler) removeMember(w http.ResponseWriter, r *http.Request) {
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

	_, err := h.roomClient.RemoveMember(r.Context(), roomID, userID, requesterID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to remove member", "error", err)
		code := errors.FromError(err)
		http.Error(w, err.Error(), code.ToHTTPCode())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
