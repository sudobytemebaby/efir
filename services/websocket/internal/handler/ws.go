package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/shared/pkg/valkey"
	"github.com/sudobytemebaby/efir/services/websocket/internal/config"
	"github.com/sudobytemebaby/efir/services/websocket/internal/hub"
	vk "github.com/valkey-io/valkey-go"
	"nhooyr.io/websocket"
)

type WebSocketHandler struct {
	hub        *hub.Hub
	gatewayURL string
	client     vk.Client
	cfg        *config.Config
}

func NewWebSocketHandler(hub *hub.Hub, gatewayURL string, client vk.Client, cfg *config.Config) *WebSocketHandler {
	return &WebSocketHandler{
		hub:        hub,
		gatewayURL: gatewayURL,
		client:     client,
		cfg:        cfg,
	}
}

func (h *WebSocketHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		http.Error(w, "missing ticket", http.StatusUnauthorized)
		return
	}

	userID, err := h.validateTicket(r.Context(), ticket)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to validate ticket", "error", err)
		http.Error(w, "invalid or expired ticket", http.StatusUnauthorized)
		return
	}

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to accept websocket", "error", err)
		return
	}

	wsConn := &wsConnWrapper{ws: conn}
	initialRoomID := r.URL.Query().Get("room_id")

	h.hub.Register(wsConn, userID, initialRoomID)

	go h.readPump(wsConn, userID)
}

func (h *WebSocketHandler) validateTicket(ctx context.Context, ticket string) (string, error) {
	key := valkey.GatewayWSTicketKey(ticket)
	resp := h.client.Do(ctx, h.client.B().Getdel().Key(key).Build())
	userID, err := resp.ToString()
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (h *WebSocketHandler) readPump(conn *wsConnWrapper, userID string) {
	defer func() {
		h.hub.Disconnect(conn)
		if err := conn.Close(hub.StatusCode(websocket.StatusNormalClosure), "closing"); err != nil {
			slog.ErrorContext(context.Background(), "failed to close websocket", "error", err)
		}
	}()

	for {
		_, msg, err := conn.Read(context.Background())
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return
			}
			slog.ErrorContext(context.Background(), "failed to read message", "error", err)
			return
		}

		if len(msg) > int(h.cfg.ReadLimitBytes()) {
			h.sendError(conn, "message_too_large", "message exceeds size limit")
			continue
		}

		var env hub.Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			h.sendError(conn, "invalid_json", "failed to parse message")
			continue
		}

		h.handleMessage(conn, userID, env)
	}
}

func (h *WebSocketHandler) handleMessage(conn *wsConnWrapper, userID string, env hub.Envelope) {
	switch env.Type {
	case hub.TypeSubscribe:
		var payload hub.SubscribePayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			h.sendError(conn, "invalid_payload", "failed to parse subscribe payload")
			return
		}
		if _, err := uuid.Parse(payload.RoomID); err != nil {
			h.sendError(conn, "invalid_room_id", "invalid room ID format")
			return
		}
		h.hub.Register(conn, userID, payload.RoomID)

	case hub.TypeUnsubscribe:
		var payload hub.UnsubscribePayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			h.sendError(conn, "invalid_payload", "failed to parse unsubscribe payload")
			return
		}
		h.hub.Unregister(conn, payload.RoomID)

	case hub.TypePing:
		pong := hub.Envelope{Type: hub.TypePong}
		if err := conn.WriteJSON(pong); err != nil {
			slog.Error("failed to send pong", "error", err)
		}

	default:
		h.sendError(conn, "unknown_type", "unknown message type")
	}
}

func (h *WebSocketHandler) sendError(conn *wsConnWrapper, code, message string) {
	errResp := hub.ErrorPayload{Code: code, Message: message}
	env := hub.Envelope{
		Type:    hub.TypeError,
		Payload: json.RawMessage{},
	}
	payload, err := json.Marshal(errResp)
	if err != nil {
		slog.Error("failed to marshal error payload", "error", err)
		return
	}
	env.Payload = payload

	if err := conn.WriteJSON(env); err != nil {
		slog.Error("failed to send error", "error", err)
	}
}

type wsConnWrapper struct {
	ws *websocket.Conn
}

func (c *wsConnWrapper) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	return c.ws.Read(ctx)
}

func (c *wsConnWrapper) WriteJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.ws.Write(context.Background(), websocket.MessageText, data)
}

func (c *wsConnWrapper) Close(code hub.StatusCode, reason string) error {
	return c.ws.Close(websocket.StatusCode(code), reason)
}
