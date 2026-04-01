package hub

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
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
	Close(code StatusCode, reason string) error
	Send(data []byte) bool
}

type Hub struct {
	rooms     map[string]map[string][]Conn
	userIDs   map[Conn]string
	connRooms map[Conn]map[string]struct{}

	register   chan *ConnRegistration
	unregister chan *ConnUnregistration
	disconnect chan Conn
	broadcast  chan *BroadcastMessage
	roomCount  chan *RoomCountRequest
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

type RoomCountRequest struct {
	RoomID  string
	CountCh chan int
}

func NewHub(bufferSize int) *Hub {
	return &Hub{
		rooms:      make(map[string]map[string][]Conn),
		userIDs:    make(map[Conn]string),
		connRooms:  make(map[Conn]map[string]struct{}),
		register:   make(chan *ConnRegistration, bufferSize),
		unregister: make(chan *ConnUnregistration, bufferSize),
		disconnect: make(chan Conn, bufferSize),
		broadcast:  make(chan *BroadcastMessage, bufferSize),
		roomCount:  make(chan *RoomCountRequest, bufferSize),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case reg := <-h.register:
			h.addConn(reg.Conn, reg.UserID, reg.RoomID)

		case unreg := <-h.unregister:
			h.removeConn(unreg.Conn, unreg.RoomID)

		case conn := <-h.disconnect:
			h.disconnectAll(conn)

		case msg := <-h.broadcast:
			h.sendToRoom(msg.RoomID, msg.Envelope)

		case req := <-h.roomCount:
			count := h.getRoomUserCount(req.RoomID)
			req.CountCh <- count
		}
	}
}

func (h *Hub) Register(conn Conn, userID, roomID string) {
	h.register <- &ConnRegistration{Conn: conn, UserID: userID, RoomID: roomID}
}

func (h *Hub) Unregister(conn Conn, roomID string) {
	h.unregister <- &ConnUnregistration{Conn: conn, RoomID: roomID}
}

func (h *Hub) Disconnect(conn Conn) {
	h.disconnect <- conn
}

func (h *Hub) BroadcastToRoom(roomID string, envelope Envelope) {
	h.broadcast <- &BroadcastMessage{RoomID: roomID, Envelope: envelope}
}

func (h *Hub) GetRoomUserCount(roomID string) int {
	countCh := make(chan int, 1)
	h.roomCount <- &RoomCountRequest{RoomID: roomID, CountCh: countCh}
	return <-countCh
}

func (h *Hub) addConn(conn Conn, userID, roomID string) {
	h.userIDs[conn] = userID

	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[string][]Conn)
	}

	userConns := h.rooms[roomID][userID]
	if slices.Contains(userConns, conn) {
		return
	}
	h.rooms[roomID][userID] = append(userConns, conn)

	if h.connRooms[conn] == nil {
		h.connRooms[conn] = make(map[string]struct{})
	}
	h.connRooms[conn][roomID] = struct{}{}
}

func (h *Hub) removeConn(conn Conn, roomID string) {
	userID := h.userIDs[conn]
	if userID == "" {
		return
	}

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

	delete(h.connRooms[conn], roomID)
	if len(h.connRooms[conn]) == 0 {
		delete(h.connRooms, conn)
		delete(h.userIDs, conn)
	}
}

func (h *Hub) disconnectAll(conn Conn) {
	userID := h.userIDs[conn]
	if userID == "" {
		return
	}

	for roomID := range h.connRooms[conn] {
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

	delete(h.connRooms, conn)
	delete(h.userIDs, conn)
}

func (h *Hub) sendToRoom(roomID string, envelope Envelope) {
	room := h.rooms[roomID]
	if len(room) == 0 {
		return
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		slog.Error("failed to marshal envelope", "error", err)
		return
	}

	for _, conns := range room {
		for _, conn := range conns {
			if !conn.Send(data) {
				go func(c Conn) {
					_ = c.Close(StatusAbnormalClosure, "slow write")
				}(conn)
			}
		}
	}
}

func (h *Hub) getRoomUserCount(roomID string) int {
	if room := h.rooms[roomID]; room != nil {
		return len(room)
	}
	return 0
}
