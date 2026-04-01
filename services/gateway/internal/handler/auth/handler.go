package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler"
	"github.com/sudobytemebaby/efir/services/gateway/internal/middleware"
	authv1 "github.com/sudobytemebaby/efir/services/shared/gen/auth"
)

type Handler struct {
	client authv1.AuthServiceClient
}

func NewHandler(client authv1.AuthServiceClient) *Handler {
	return &Handler{client: client}
}

func (h *Handler) Register(r chi.Router) {
	r.Post("/auth/register", h.register)
	r.Post("/auth/login", h.login)
	r.Post("/auth/logout", h.logout)
	r.Post("/auth/refresh", h.refresh)
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
	handler.WriteProto(w, http.StatusOK, resp)
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
	handler.WriteProto(w, http.StatusOK, resp)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var req authv1.LogoutRequest
	if err := handler.ReadProto(r, &req); err != nil {
		handler.WriteError(w, r, err, "invalid request body")
		return
	}
	ctx := middleware.InjectRequestIDToOutgoingContext(r.Context())
	if _, err := h.client.Logout(ctx, &req); err != nil {
		handler.WriteError(w, r, err, "failed to logout")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req authv1.RefreshTokenRequest
	if err := handler.ReadProto(r, &req); err != nil {
		handler.WriteError(w, r, err, "invalid request body")
		return
	}
	ctx := middleware.InjectRequestIDToOutgoingContext(r.Context())
	resp, err := h.client.RefreshToken(ctx, &req)
	if err != nil {
		handler.WriteError(w, r, err, "failed to refresh token")
		return
	}
	handler.WriteProto(w, http.StatusOK, resp)
}
