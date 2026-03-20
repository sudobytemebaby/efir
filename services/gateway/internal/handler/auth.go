package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/shared/pkg/valkey"
	vk "github.com/valkey-io/valkey-go"
)

type WSAuthHandler struct {
	client    vk.Client
	ticketTTL time.Duration
}

func NewWSAuthHandler(client vk.Client, ticketTTL time.Duration) *WSAuthHandler {
	return &WSAuthHandler{
		client:    client,
		ticketTTL: ticketTTL,
	}
}

func (h *WSAuthHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/ws-ticket", h.CreateTicket)
	mux.HandleFunc("GET /auth/validate", h.ValidateTicket)
}

type CreateTicketResponse struct {
	Ticket string `json:"ticket"`
}

func (h *WSAuthHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		http.Error(w, "missing X-User-Id header", http.StatusUnauthorized)
		return
	}

	if _, err := uuid.Parse(userID); err != nil {
		http.Error(w, "invalid X-User-Id header", http.StatusBadRequest)
		return
	}

	ticket := uuid.New().String()
	key := valkey.GatewayWSTicketKey(ticket)

	err := h.client.Do(ctx, h.client.B().Set().Key(key).Value(userID).Ex(h.ticketTTL).Build()).Error()
	if err != nil {
		slog.ErrorContext(ctx, "failed to store ws ticket", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(CreateTicketResponse{Ticket: ticket}); err != nil {
		slog.ErrorContext(ctx, "failed to encode response", "error", err)
	}
}

func (h *WSAuthHandler) ValidateTicket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ticket := r.Header.Get("X-Ws-Ticket")
	if ticket == "" {
		http.Error(w, "missing X-Ws-Ticket header", http.StatusUnauthorized)
		return
	}

	key := valkey.GatewayWSTicketKey(ticket)

	getResp := h.client.Do(ctx, h.client.B().Getdel().Key(key).Build())
	userID, err := getResp.ToString()
	if err != nil {
		if vk.IsValkeyNil(err) {
			http.Error(w, "invalid or expired ticket", http.StatusUnauthorized)
			return
		}
		slog.ErrorContext(ctx, "failed to get/del ws ticket", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-User-Id", userID)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"user_id": userID}); err != nil {
		slog.ErrorContext(ctx, "failed to encode response", "error", err)
	}
}
