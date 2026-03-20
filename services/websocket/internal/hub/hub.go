package hub

import (
	"encoding/json"
	"log/slog"
	"sync"
)

const (
	TypeMessageCreated       = "message.created"
	TypeRoomMembershipChange = "room.membership.changed"
	TypeRoomUpdated          = "room.updated"
	TypePing                 = "ping"
	TypePong                 = "pong"
	TypeSubscribe            = "subscribe"
	TypeUnsubscribe          = "unsubscribe"
	TypeError                = "error"
)

type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type MessageCreatedPayload struct {
	MessageID string `json:"message_id"`
	RoomID    string `json:"room_id"`
	UserID    string `json:"user_id"`
	Content   any    `json:"content"`
}

type RoomMembershipChangedPayload struct {
	RoomID string `json:"room_id"`
	UserID string `json:"user_id"`
	Joined bool   `json:"joined"`
}

type RoomUpdatedPayload struct {
	RoomID    string `json:"room_id"`
	UpdatedBy string `json:"updated_by"`
}

type SubscribePayload struct {
	RoomID string `json:"room_id"`
}

type UnsubscribePayload struct {
	RoomID string `json:"room_id"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type StatusCode int

const StatusAbnormalClosure StatusCode = 1006

type Conn interface {
	WriteJSON(v any) error
	Close(code StatusCode, reason string) error
}

type Hub struct {
	mu      sync.RWMutex
	rooms   map[string]map[string][]Conn
	userIDs map[Conn]string

	register   chan *ConnRegistration
	unregister chan *ConnUnregistration
	broadcast  chan *BroadcastMessage
}

type ConnRegistration struct {
	Conn   Conn
	UserID string
	RoomID string
}

type ConnUnregistration struct {
	Conn   Conn
	RoomID string
}

type BroadcastMessage struct {
	RoomID   string
	Envelope Envelope
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[string][]Conn),
		userIDs:    make(map[Conn]string),
		register:   make(chan *ConnRegistration, 256),
		unregister: make(chan *ConnUnregistration, 256),
		broadcast:  make(chan *BroadcastMessage, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case reg := <-h.register:
			h.addConn(reg.Conn, reg.UserID, reg.RoomID)

		case unreg := <-h.unregister:
			h.removeConn(unreg.Conn, unreg.RoomID)

		case msg := <-h.broadcast:
			h.sendToRoom(msg.RoomID, msg.Envelope)
		}
	}
}

func (h *Hub) Register(conn Conn, userID, roomID string) {
	h.register <- &ConnRegistration{Conn: conn, UserID: userID, RoomID: roomID}
}

func (h *Hub) Unregister(conn Conn, roomID string) {
	h.unregister <- &ConnUnregistration{Conn: conn, RoomID: roomID}
}

func (h *Hub) BroadcastToRoom(roomID string, envelope Envelope) {
	h.broadcast <- &BroadcastMessage{RoomID: roomID, Envelope: envelope}
}

func (h *Hub) addConn(conn Conn, userID, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.userIDs[conn] = userID

	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[string][]Conn)
	}

	userConns := h.rooms[roomID][userID]
	h.rooms[roomID][userID] = append(userConns, conn)
}

func (h *Hub) removeConn(conn Conn, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	userID := h.userIDs[conn]
	if userID == "" {
		return
	}

	delete(h.userIDs, conn)

	conns := h.rooms[roomID][userID]
	for i, c := range conns {
		if c == conn {
			h.rooms[roomID][userID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}

	if len(h.rooms[roomID][userID]) == 0 {
		delete(h.rooms[roomID], userID)
	}

	if len(h.rooms[roomID]) == 0 {
		delete(h.rooms, roomID)
	}
}

func (h *Hub) sendToRoom(roomID string, envelope Envelope) {
	h.mu.RLock()
	room := h.rooms[roomID]
	h.mu.RUnlock()

	if len(room) == 0 {
		return
	}

	for _, conns := range room {
		for _, conn := range conns {
			if err := conn.WriteJSON(envelope); err != nil {
				slog.Error("failed to write to conn", "error", err)
				if closeErr := conn.Close(StatusAbnormalClosure, "write error"); closeErr != nil {
					slog.Error("failed to close conn", "error", closeErr)
				}
			}
		}
	}
}

func (h *Hub) GetRoomUserCount(roomID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if room := h.rooms[roomID]; room != nil {
		return len(room)
	}
	return 0
}
