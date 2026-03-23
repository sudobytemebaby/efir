package nats

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/sudobytemebaby/efir/services/message/internal/service"
)

type publisher struct {
	js jetstream.JetStream
}

func NewPublisher(js jetstream.JetStream) service.Publisher {
	return &publisher{js: js}
}

func (p *publisher) PublishMessageCreated(ctx context.Context, msg *service.Message, recipientIDs []uuid.UUID) error {
	recipientStrs := make([]string, len(recipientIDs))
	for i, id := range recipientIDs {
		recipientStrs[i] = id.String()
	}

	payload := map[string]any{
		"message_id":    msg.ID.String(),
		"room_id":       msg.RoomID.String(),
		"sender_id":     msg.SenderID.String(),
		"type":          string(msg.Type),
		"recipient_ids": recipientStrs,
		"created_at":    msg.CreatedAt.Format(time.RFC3339),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = p.js.Publish(ctx, SubjectMessageCreated, data)
	return err
}
