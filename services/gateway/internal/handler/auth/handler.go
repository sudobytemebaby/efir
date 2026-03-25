package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler"
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
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	resp, err := h.client.Register(r.Context(), &req)
	if err != nil {
		handler.WriteError(w, r, err, "failed to register")
		return
	}
	handler.WriteProto(w, http.StatusOK, resp)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req authv1.LoginRequest
	if err := handler.ReadProto(r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	resp, err := h.client.Login(r.Context(), &req)
	if err != nil {
		handler.WriteError(w, r, err, "failed to login")
		return
	}
	handler.WriteProto(w, http.StatusOK, resp)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var req authv1.LogoutRequest
	if err := handler.ReadProto(r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if _, err := h.client.Logout(r.Context(), &req); err != nil {
		handler.WriteError(w, r, err, "failed to logout")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req authv1.RefreshTokenRequest
	if err := handler.ReadProto(r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	resp, err := h.client.RefreshToken(r.Context(), &req)
	if err != nil {
		handler.WriteError(w, r, err, "failed to refresh token")
		return
	}
	handler.WriteProto(w, http.StatusOK, resp)
}
