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

type UserHandler struct {
	userClient *client.UserClient
}

func NewUserHandler(userClient *client.UserClient) *UserHandler {
	return &UserHandler{
		userClient: userClient,
	}
}

func (h *UserHandler) Register(r chi.Router) {
	r.Get("/users/me", h.getMe)
	r.Get("/users/{id}", h.getByID)
	r.Patch("/users/me", h.updateMe)
}

type userResponse struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Bio         string `json:"bio,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func userToResponse(user *client.User) userResponse {
	resp := userResponse{
		UserID:      user.UserId,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		CreatedAt:   timestampToString(user.CreatedAt),
		UpdatedAt:   timestampToString(user.UpdatedAt),
	}
	if user.AvatarUrl != nil {
		resp.AvatarURL = *user.AvatarUrl
	}
	if user.Bio != nil {
		resp.Bio = *user.Bio
	}
	return resp
}

func (h *UserHandler) getMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := h.userClient.GetUser(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get user", "error", err)
		code := errors.FromError(err)
		http.Error(w, err.Error(), code.ToHTTPCode())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(userToResponse(resp.User)); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}

func (h *UserHandler) getByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing user id", http.StatusBadRequest)
		return
	}

	resp, err := h.userClient.GetUser(r.Context(), id)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get user", "error", err)
		code := errors.FromError(err)
		http.Error(w, err.Error(), code.ToHTTPCode())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(userToResponse(resp.User)); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}

type updateUserRequest struct {
	DisplayName *string `json:"display_name,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	Bio         *string `json:"bio,omitempty"`
}

func (h *UserHandler) updateMe(w http.ResponseWriter, r *http.Request) {
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

	resp, err := h.userClient.UpdateUser(r.Context(), userID, req.DisplayName, req.AvatarURL, req.Bio)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to update user", "error", err)
		code := errors.FromError(err)
		http.Error(w, err.Error(), code.ToHTTPCode())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(userToResponse(resp.User)); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}
