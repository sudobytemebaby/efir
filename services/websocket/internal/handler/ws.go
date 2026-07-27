package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/shared/pkg/errors"
	"github.com/sudobytemebaby/efir/services/websocket/internal/config"
	"github.com/sudobytemebaby/efir/services/websocket/internal/hub"
	"nhooyr.io/websocket"
)

type WebSocketHandler struct {
	hub *hub.Hub
	cfg *config.Config
}

func NewWebSocketHandler(hub *hub.Hub, cfg *config.Config) *WebSocketHandler {
	return &WebSocketHandler{
		hub: hub,
		cfg: cfg,
	}
}

// HandleWS upgrades the HTTP connection to WebSocket. Each connection is managed
// by three goroutines (readPump, writePump, pingPump) that share a cancellable
// context — when any one exits it cancels the context, triggering the others to stop.
func (h *WebSocketHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		writeError(w, r, errors.CodeUnauthenticated, "missing user id")
		return
	}

	initialRoomID := r.URL.Query().Get("room_id")
	if initialRoomID != "" {
		if _, err := uuid.Parse(initialRoomID); err != nil {
			writeError(w, r, errors.CodeInvalidArgument, "invalid room_id format")
			return
		}
	}

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to accept websocket", "error", err)
		return
	}

	conn.SetReadLimit(h.cfg.WebSocket.ReadLimit)
	wsConn := newWSConnWrapper(conn, h.cfg.WebSocket.WriteDeadline, h.cfg.WebSocket.ReadDeadline, h.cfg.WebSocket.WriteBuffer)

	if initialRoomID != "" {
		h.hub.Register(wsConn, userID, initialRoomID)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go h.writePump(ctx, cancel, wsConn, userID)
	go h.readPump(ctx, cancel, wsConn, userID)
	go h.pingPump(ctx, cancel, wsConn, userID)
}

func (h *WebSocketHandler) readPump(ctx context.Context, cancel context.CancelFunc, conn *wsConnWrapper, userID string) {
	defer func() {
		cancel()
		h.hub.Disconnect(conn)
		conn.closeOutbound()
		if err := conn.Close(hub.StatusCode(websocket.StatusNormalClosure), "closing"); err != nil {
			slog.ErrorContext(context.Background(), "failed to close websocket", "user_id", userID, "error", err)
		}
	}()

	for {
		readCtx, readCancel := context.WithTimeout(ctx, conn.readDeadline)
		_, msg, err := conn.Read(readCtx)
		readCancel()
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return
			}
			slog.ErrorContext(context.Background(), "failed to read message", "user_id", userID, "error", err)
			return
		}

		var env hub.Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			h.sendError(conn, "invalid_json", "failed to parse message")
			continue
		}

		h.handleMessage(conn, userID, env)
	}
}

func (h *WebSocketHandler) writePump(ctx context.Context, cancel context.CancelFunc, conn *wsConnWrapper, userID string) {
	defer cancel()
	for data := range conn.outbound {
		writeCtx, writeCancel := context.WithTimeout(ctx, conn.writeTimeout)
		err := conn.ws.Write(writeCtx, websocket.MessageText, data)
		writeCancel()
		if err != nil {
			slog.ErrorContext(context.Background(), "failed to write message", "user_id", userID, "error", err)
			h.hub.Disconnect(conn)
			return
		}
	}
}

func (h *WebSocketHandler) pingPump(ctx context.Context, cancel context.CancelFunc, conn *wsConnWrapper, userID string) {
	defer cancel()
	ticker := time.NewTicker(h.cfg.WebSocket.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := conn.Ping(); err != nil {
				slog.ErrorContext(context.Background(), "failed to send ping", "user_id", userID, "error", err)
				h.hub.Disconnect(conn)
				conn.closeOutbound()
				return
			}
		}
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
		data, err := json.Marshal(pong)
		if err != nil {
			slog.Error("failed to marshal pong", "error", err)
			return
		}
		conn.Send(data)

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

	data, err := json.Marshal(env)
	if err != nil {
		slog.Error("failed to marshal error envelope", "error", err)
		return
	}
	conn.Send(data)
}

func writeError(w http.ResponseWriter, r *http.Request, code errors.Code, msg string) {
	slog.ErrorContext(r.Context(), msg, "code", code)

	body := map[string]string{"error": msg, "code": string(code)}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code.ToHTTPCode())
	_ = json.NewEncoder(w).Encode(body)
}

type wsConnWrapper struct {
	ws           *websocket.Conn
	writeTimeout time.Duration
	readDeadline time.Duration
	outbound     chan []byte
	closeOnce    sync.Once
}

func newWSConnWrapper(ws *websocket.Conn, writeTimeout, readDeadline time.Duration, bufSize int) *wsConnWrapper {
	return &wsConnWrapper{
		ws:           ws,
		writeTimeout: writeTimeout,
		readDeadline: readDeadline,
		outbound:     make(chan []byte, bufSize),
	}
}

func (c *wsConnWrapper) closeOutbound() {
	c.closeOnce.Do(func() {
		close(c.outbound)
	})
}

// Send enqueues data for the writePump without blocking. Returns false if the buffer
// is full (slow client). The recover guards against a send on a closed channel during
// shutdown, which would otherwise panic.
func (c *wsConnWrapper) Send(data []byte) bool {
	defer func() {
		_ = recover()
	}()
	select {
	case c.outbound <- data:
		return true
	default:
		return false
	}
}

func (c *wsConnWrapper) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.writeTimeout)
	defer cancel()
	return c.ws.Ping(ctx)
}

func (c *wsConnWrapper) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	return c.ws.Read(ctx)
}

func (c *wsConnWrapper) Close(code hub.StatusCode, reason string) error {
	return c.ws.Close(websocket.StatusCode(code), reason)
}
