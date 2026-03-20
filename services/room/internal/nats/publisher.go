package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/sudobytemebaby/efir/services/room/internal/service"
)

type publisher struct {
	js jetstream.JetStream
}

func NewPublisher(js jetstream.JetStream) service.Publisher {
	return &publisher{js: js}
}

func (p *publisher) PublishMembershipChanged(ctx context.Context, roomID, userID uuid.UUID, action string, recipientIDs []uuid.UUID) error {
	recipientIDStrs := make([]string, len(recipientIDs))
	for i, id := range recipientIDs {
		recipientIDStrs[i] = id.String()
	}

	payload := map[string]interface{}{
		"room_id":       roomID.String(),
		"user_id":       userID.String(),
		"action":        action,
		"recipient_ids": recipientIDStrs,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	_, err = p.js.Publish(ctx, SubjectMembershipChange, data)
	if err != nil {
		return fmt.Errorf("publish to nats: %w", err)
	}

	return nil
}

func (p *publisher) PublishRoomUpdated(ctx context.Context, roomID uuid.UUID, name string, recipientIDs []uuid.UUID) error {
	recipientIDStrs := make([]string, len(recipientIDs))
	for i, id := range recipientIDs {
		recipientIDStrs[i] = id.String()
	}

	payload := map[string]interface{}{
		"room_id":       roomID.String(),
		"name":          name,
		"recipient_ids": recipientIDStrs,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	_, err = p.js.Publish(ctx, SubjectRoomUpdated, data)
	if err != nil {
		return fmt.Errorf("publish to nats: %w", err)
	}

	return nil
}
