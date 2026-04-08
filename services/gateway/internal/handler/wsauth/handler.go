package wsauth

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler"
	"github.com/sudobytemebaby/efir/services/gateway/internal/middleware"
	"github.com/sudobytemebaby/efir/services/shared/pkg/errors"
	"github.com/sudobytemebaby/efir/services/shared/pkg/valkey"
	vk "github.com/valkey-io/valkey-go"
)

type Handler struct {
	client    vk.Client
	ticketTTL time.Duration
}

func NewHandler(client vk.Client, ticketTTL time.Duration) *Handler {
	return &Handler{
		client:    client,
		ticketTTL: ticketTTL,
	}
}

// Register wires only ValidateTicket. CreateTicket requires JWT+rate-limit middleware
// and must be registered separately by the caller (as main.go does).
func (h *Handler) Register(r chi.Router) {
	r.Get("/auth/validate", h.ValidateTicket)
}

type createTicketResponse struct {
	Ticket string `json:"ticket"`
}

func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := middleware.MustGetUserID(r.Context())
	if userID == "" {
		handler.WriteCode(w, errors.CodeUnauthenticated)
		return
	}

	if _, err := uuid.Parse(userID); err != nil {
		handler.WriteCode(w, errors.CodeInvalidArgument)
		return
	}

	ticket := uuid.New().String()
	key := valkey.GatewayWSTicketKey(ticket)

	err := h.client.Do(ctx, h.client.B().Set().Key(key).Value(userID).Ex(h.ticketTTL).Build()).Error()
	if err != nil {
		slog.ErrorContext(ctx, "failed to store ws ticket", "error", err)
		handler.WriteCode(w, errors.CodeInternal)
		return
	}

	handler.WriteJSON(w, http.StatusCreated, createTicketResponse{Ticket: ticket})
}

func (h *Handler) ValidateTicket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ticket := r.Header.Get("X-Ws-Ticket")
	if ticket == "" {
		handler.WriteCode(w, errors.CodeUnauthenticated)
		return
	}

	key := valkey.GatewayWSTicketKey(ticket)

	getResp := h.client.Do(ctx, h.client.B().Getdel().Key(key).Build())
	userID, err := getResp.ToString()
	if err != nil {
		if vk.IsValkeyNil(err) {
			handler.WriteCode(w, errors.CodeUnauthenticated)
			return
		}
		slog.ErrorContext(ctx, "failed to get/del ws ticket", "error", err)
		handler.WriteCode(w, errors.CodeInternal)
		return
	}

	w.Header().Set("X-User-Id", userID)
	handler.WriteJSON(w, http.StatusOK, map[string]string{"user_id": userID})
}
