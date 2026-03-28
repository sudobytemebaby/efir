package message

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler"
	"github.com/sudobytemebaby/efir/services/gateway/internal/middleware"
	messagev1 "github.com/sudobytemebaby/efir/services/shared/gen/message"
)

type Handler struct {
	client messagev1.MessageServiceClient
}

func NewHandler(client messagev1.MessageServiceClient) *Handler {
	return &Handler{client: client}
}

func (h *Handler) Register(r chi.Router) {
	r.Post("/rooms/{id}/messages", h.sendMessage)
	r.Get("/rooms/{id}/messages", h.getMessages)
}

func (h *Handler) sendMessage(w http.ResponseWriter, r *http.Request) {
	var req messagev1.SendMessageRequest
	if err := handler.ReadProto(r, &req); err != nil {
		handler.WriteError(w, r, err, "invalid request body")
		return
	}
	req.RoomId = chi.URLParam(r, "id")
	req.SenderId = middleware.MustGetUserID(r.Context())

	resp, err := h.client.SendMessage(middleware.InjectRequestIDToOutgoingContext(r.Context()), &req)
	if err != nil {
		handler.WriteError(w, r, err, "failed to send message")
		return
	}
	handler.WriteProto(w, http.StatusCreated, resp.Message)
}

func (h *Handler) getMessages(w http.ResponseWriter, r *http.Request) {
	req := &messagev1.GetMessagesRequest{
		RoomId:      chi.URLParam(r, "id"),
		RequesterId: middleware.MustGetUserID(r.Context()),
		Limit:       50,
	}

	if c := r.URL.Query().Get("cursor"); c != "" {
		req.Cursor = &c
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.ParseInt(l, 10, 32); err == nil {
			req.Limit = int32(parsed)
		}
	}

	resp, err := h.client.GetMessages(middleware.InjectRequestIDToOutgoingContext(r.Context()), req)
	if err != nil {
		handler.WriteError(w, r, err, "failed to get messages")
		return
	}
	handler.WriteProto(w, http.StatusOK, resp)
}
