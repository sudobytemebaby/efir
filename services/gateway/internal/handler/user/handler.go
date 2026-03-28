package user

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler"
	"github.com/sudobytemebaby/efir/services/gateway/internal/middleware"
	userv1 "github.com/sudobytemebaby/efir/services/shared/gen/user"
)

type Handler struct {
	client userv1.UserServiceClient
}

func NewHandler(client userv1.UserServiceClient) *Handler {
	return &Handler{client: client}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/users/me", h.getMe)
	r.Get("/users/{id}", h.getByID)
	r.Patch("/users/me", h.updateMe)
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	resp, err := h.client.GetUser(
		middleware.InjectRequestIDToOutgoingContext(r.Context()),
		&userv1.GetUserRequest{UserId: middleware.MustGetUserID(r.Context())},
	)
	if err != nil {
		handler.WriteError(w, r, err, "failed to get user")
		return
	}
	handler.WriteProto(w, http.StatusOK, resp.User)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	resp, err := h.client.GetUser(
		middleware.InjectRequestIDToOutgoingContext(r.Context()),
		&userv1.GetUserRequest{UserId: chi.URLParam(r, "id")},
	)
	if err != nil {
		handler.WriteError(w, r, err, "failed to get user")
		return
	}
	handler.WriteProto(w, http.StatusOK, resp.User)
}

func (h *Handler) updateMe(w http.ResponseWriter, r *http.Request) {
	var req userv1.UpdateUserRequest
	if err := handler.ReadProto(r, &req); err != nil {
		handler.WriteError(w, r, err, "invalid request body")
		return
	}
	req.UserId = middleware.MustGetUserID(r.Context())

	resp, err := h.client.UpdateUser(middleware.InjectRequestIDToOutgoingContext(r.Context()), &req)
	if err != nil {
		handler.WriteError(w, r, err, "failed to update user")
		return
	}
	handler.WriteProto(w, http.StatusOK, resp.User)
}
