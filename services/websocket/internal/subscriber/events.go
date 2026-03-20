package subscriber

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/sudobytemebaby/efir/services/websocket/internal/hub"
)

type MessageCreatedEvent struct {
	MessageID string `json:"message_id"`
	RoomID    string `json:"room_id"`
	UserID    string `json:"user_id"`
	Content   any    `json:"content"`
}

type RoomMembershipChangedEvent struct {
	RoomID string `json:"room_id"`
	UserID string `json:"user_id"`
	Joined bool   `json:"joined"`
}

type RoomUpdatedEvent struct {
	RoomID    string `json:"room_id"`
	UpdatedBy string `json:"updated_by"`
}

type Subscriber struct {
	hub *hub.Hub
	nc  *nats.Conn
}

func NewSubscriber(hub *hub.Hub, nc *nats.Conn) *Subscriber {
	return &Subscriber{
		hub: hub,
		nc:  nc,
	}
}

func (s *Subscriber) Start(ctx context.Context) error {
	if _, err := s.nc.Subscribe("message.created", s.handleMessageCreated); err != nil {
		return err
	}

	if _, err := s.nc.Subscribe("room.membership.changed", s.handleRoomMembershipChanged); err != nil {
		return err
	}

	if _, err := s.nc.Subscribe("room.updated", s.handleRoomUpdated); err != nil {
		return err
	}

	return nil
}

func (s *Subscriber) handleMessageCreated(msg *nats.Msg) {
	var event MessageCreatedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		slog.Error("failed to unmarshal message.created event", "error", err)
		return
	}

	envelope := hub.Envelope{
		Type:    hub.TypeMessageCreated,
		Payload: json.RawMessage{},
	}

	payload, err := json.Marshal(hub.MessageCreatedPayload{
		MessageID: event.MessageID,
		RoomID:    event.RoomID,
		UserID:    event.UserID,
		Content:   event.Content,
	})
	if err != nil {
		slog.Error("failed to marshal message.created payload", "error", err)
		return
	}
	envelope.Payload = payload

	s.hub.BroadcastToRoom(event.RoomID, envelope)
}

func (s *Subscriber) handleRoomMembershipChanged(msg *nats.Msg) {
	var event RoomMembershipChangedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		slog.Error("failed to unmarshal room.membership.changed event", "error", err)
		return
	}

	envelope := hub.Envelope{
		Type:    hub.TypeRoomMembershipChange,
		Payload: json.RawMessage{},
	}

	payload, err := json.Marshal(hub.RoomMembershipChangedPayload{
		RoomID: event.RoomID,
		UserID: event.UserID,
		Joined: event.Joined,
	})
	if err != nil {
		slog.Error("failed to marshal room.membership.changed payload", "error", err)
		return
	}
	envelope.Payload = payload

	s.hub.BroadcastToRoom(event.RoomID, envelope)
}

func (s *Subscriber) handleRoomUpdated(msg *nats.Msg) {
	var event RoomUpdatedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		slog.Error("failed to unmarshal room.updated event", "error", err)
		return
	}

	envelope := hub.Envelope{
		Type:    hub.TypeRoomUpdated,
		Payload: json.RawMessage{},
	}

	payload, err := json.Marshal(hub.RoomUpdatedPayload{
		RoomID:    event.RoomID,
		UpdatedBy: event.UpdatedBy,
	})
	if err != nil {
		slog.Error("failed to marshal room.updated payload", "error", err)
		return
	}
	envelope.Payload = payload

	s.hub.BroadcastToRoom(event.RoomID, envelope)
}
