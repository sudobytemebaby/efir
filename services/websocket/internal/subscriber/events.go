package subscriber

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"
	sharednats "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
	"github.com/sudobytemebaby/efir/services/websocket/internal/hub"
	wsnats "github.com/sudobytemebaby/efir/services/websocket/internal/nats"
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
	hub  *hub.Hub
	js   jetstream.JetStream
	subs []jetstream.ConsumeContext
}

func NewSubscriber(hub *hub.Hub, js jetstream.JetStream) *Subscriber {
	return &Subscriber{
		hub:  hub,
		js:   js,
		subs: make([]jetstream.ConsumeContext, 0, 3),
	}
}

func (s *Subscriber) Start(ctx context.Context) error {
	msgConsumer, err := sharednats.ProvisionConsumerWithRetry(ctx, s.js, wsnats.StreamMessage, wsnats.MessageCreatedConsumer())
	if err != nil {
		return err
	}

	membershipConsumer, err := sharednats.ProvisionConsumerWithRetry(ctx, s.js, wsnats.StreamRoom, wsnats.RoomMembershipChangedConsumer())
	if err != nil {
		return err
	}

	roomUpdatedConsumer, err := sharednats.ProvisionConsumerWithRetry(ctx, s.js, wsnats.StreamRoom, wsnats.RoomUpdatedConsumer())
	if err != nil {
		return err
	}

	msgSub, err := msgConsumer.Consume(s.handleMessageCreated)
	if err != nil {
		return err
	}
	s.subs = append(s.subs, msgSub)

	membershipSub, err := membershipConsumer.Consume(s.handleRoomMembershipChanged)
	if err != nil {
		return err
	}
	s.subs = append(s.subs, membershipSub)

	roomUpdatedSub, err := roomUpdatedConsumer.Consume(s.handleRoomUpdated)
	if err != nil {
		return err
	}
	s.subs = append(s.subs, roomUpdatedSub)

	go func() {
		<-ctx.Done()
		for _, sub := range s.subs {
			sub.Drain()
		}
	}()

	return nil
}

func (s *Subscriber) handleMessageCreated(msg jetstream.Msg) {
	var event MessageCreatedEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		slog.Error("failed to unmarshal message.created event", "error", err)
		_ = msg.Nak()
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
		_ = msg.Nak()
		return
	}
	envelope.Payload = payload

	s.hub.BroadcastToRoom(event.RoomID, envelope)
	_ = msg.Ack()
}

func (s *Subscriber) handleRoomMembershipChanged(msg jetstream.Msg) {
	var event RoomMembershipChangedEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		slog.Error("failed to unmarshal room.membership.changed event", "error", err)
		_ = msg.Nak()
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
		_ = msg.Nak()
		return
	}
	envelope.Payload = payload

	s.hub.BroadcastToRoom(event.RoomID, envelope)
	_ = msg.Ack()
}

func (s *Subscriber) handleRoomUpdated(msg jetstream.Msg) {
	var event RoomUpdatedEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		slog.Error("failed to unmarshal room.updated event", "error", err)
		_ = msg.Nak()
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
		_ = msg.Nak()
		return
	}
	envelope.Payload = payload

	s.hub.BroadcastToRoom(event.RoomID, envelope)
	_ = msg.Ack()
}
