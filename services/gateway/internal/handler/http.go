package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sudobytemebaby/efir/services/gateway/internal/client"
	"github.com/sudobytemebaby/efir/services/shared/pkg/errors"
	"google.golang.org/grpc/status"
)

type HTTPHandler struct {
	authClient client.AuthClientInterface
}

func NewHTTPHandler(authClient client.AuthClientInterface) *HTTPHandler {
	return &HTTPHandler{
		authClient: authClient,
	}
}

func (h *HTTPHandler) Register(r chi.Router) {
	r.Post("/auth/register", h.register)
	r.Post("/auth/login", h.login)
	r.Post("/auth/logout", h.logout)
	r.Post("/auth/refresh", h.refresh)
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	UserID       string `json:"user_id,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func (h *HTTPHandler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.authClient.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to register", "error", err)
		code := errors.FromError(err)
		http.Error(w, err.Error(), code.ToHTTPCode())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(authResponse{
		UserID:       resp.UserId,
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	}); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}

func (h *HTTPHandler) login(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.authClient.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to login", "error", err)
		code := errors.FromError(err)
		http.Error(w, err.Error(), code.ToHTTPCode())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(authResponse{
		UserID:       resp.UserId,
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	}); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *HTTPHandler) logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	_, err := h.authClient.Logout(r.Context(), req.RefreshToken)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to logout", "error", err)
		code := errors.FromError(err)
		http.Error(w, err.Error(), code.ToHTTPCode())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *HTTPHandler) refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.authClient.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to refresh token", "error", err)
		st, ok := status.FromError(err)
		if ok {
			http.Error(w, st.Message(), errors.FromError(st.Err()).ToHTTPCode())
		} else {
			http.Error(w, "failed to refresh token", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(authResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	}); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode response", "error", err)
	}
}
