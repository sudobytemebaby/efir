package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler"
	"github.com/sudobytemebaby/efir/services/gateway/internal/middleware"
	authv1 "github.com/sudobytemebaby/efir/services/shared/gen/auth"
	"github.com/sudobytemebaby/efir/services/shared/pkg/errors"
)

type Handler struct {
	client authv1.AuthServiceClient
}

func NewHandler(client authv1.AuthServiceClient) *Handler {
	return &Handler{client: client}
}

func (h *Handler) RegisterPublic(r chi.Router) {
	r.Post("/auth/register", h.register)
	r.Post("/auth/login", h.login)
}

func (h *Handler) RegisterSession(r chi.Router) {
	r.Post("/auth/session/refresh", h.refresh)
	r.Post("/auth/session/logout", h.logout)
}

func (h *Handler) RegisterProtected(r chi.Router) {
	r.Get("/auth/me", h.me)
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req authv1.RegisterRequest
	if err := handler.ReadProto(r, &req); err != nil {
		handler.WriteError(w, r, err, "invalid request body")
		return
	}
	ctx := middleware.InjectRequestIDToOutgoingContext(r.Context())
	resp, err := h.client.Register(ctx, &req)
	if err != nil {
		handler.WriteError(w, r, err, "failed to register")
		return
	}

	setAuthCookies(w, resp.AccessToken, resp.RefreshToken)
	handler.WriteJSON(w, http.StatusOK, map[string]string{
		"user_id": resp.UserId,
	})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req authv1.LoginRequest
	if err := handler.ReadProto(r, &req); err != nil {
		handler.WriteError(w, r, err, "invalid request body")
		return
	}
	ctx := middleware.InjectRequestIDToOutgoingContext(r.Context())
	resp, err := h.client.Login(ctx, &req)
	if err != nil {
		handler.WriteError(w, r, err, "failed to login")
		return
	}

	setAuthCookies(w, resp.AccessToken, resp.RefreshToken)
	handler.WriteJSON(w, http.StatusOK, map[string]string{
		"user_id": resp.UserId,
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		clearAuthCookies(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ctx := middleware.InjectRequestIDToOutgoingContext(r.Context())
	if _, err := h.client.Logout(ctx, &authv1.LogoutRequest{
		RefreshToken: cookie.Value,
	}); err != nil {
		handler.WriteError(w, r, err, "failed to logout")
		return
	}

	clearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		handler.WriteCode(w, errors.CodeUnauthenticated)
		return
	}

	ctx := middleware.InjectRequestIDToOutgoingContext(r.Context())
	resp, err := h.client.RefreshToken(ctx, &authv1.RefreshTokenRequest{
		RefreshToken: cookie.Value,
	})
	if err != nil {
		handler.WriteError(w, r, err, "failed to refresh token")
		return
	}

	setAuthCookies(w, resp.AccessToken, resp.RefreshToken)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.MustGetUserID(r.Context())
	handler.WriteJSON(w, http.StatusOK, map[string]string{
		"user_id": userID,
	})
}
