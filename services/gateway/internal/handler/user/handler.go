package user

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
	userClient *client.UserClient
}

func NewHandler(userClient *client.UserClient) *Handler {
	return &Handler{
		userClient: userClient,
	}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/users/me", h.getMe)
	r.Get("/users/{id}", h.getByID)
	r.Patch("/users/me", h.updateMe)
}

func mapUserToResponse(user *client.User) userResponse {
	resp := userResponse{
		UserID:      user.UserId,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		CreatedAt:   handler.TimestampToString(user.CreatedAt),
		UpdatedAt:   handler.TimestampToString(user.UpdatedAt),
	}
	if user.AvatarUrl != nil {
		resp.AvatarURL = *user.AvatarUrl
	}
	if user.Bio != nil {
		resp.Bio = *user.Bio
	}
	return resp
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := h.userClient.GetUser(middleware.InjectRequestIDToOutgoingContext(r.Context()), userID)
	if err != nil {
		handler.WriteError(w, r, err, "failed to get user")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(mapUserToResponse(resp.User)); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing user id", http.StatusBadRequest)
		return
	}

	resp, err := h.userClient.GetUser(middleware.InjectRequestIDToOutgoingContext(r.Context()), id)
	if err != nil {
		handler.WriteError(w, r, err, "failed to get user")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(mapUserToResponse(resp.User)); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}

func (h *Handler) updateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.userClient.UpdateUser(middleware.InjectRequestIDToOutgoingContext(r.Context()), userID, req.DisplayName, req.AvatarURL, req.Bio)
	if err != nil {
		handler.WriteError(w, r, err, "failed to update user")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(mapUserToResponse(resp.User)); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}
