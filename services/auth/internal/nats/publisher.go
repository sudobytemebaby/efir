package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/sudobytemebaby/efir/services/auth/internal/service"
)

type publisher struct {
	js jetstream.JetStream
}

func NewPublisher(js jetstream.JetStream) service.Publisher {
	return &publisher{js: js}
}

func (p *publisher) PublishUserRegistered(ctx context.Context, userID uuid.UUID, email string) error {
	payload := map[string]string{
		"user_id": userID.String(),
		"email":   email,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	_, err = p.js.Publish(ctx, "auth.user.registered", data)
	if err != nil {
		return fmt.Errorf("publish to nats: %w", err)
	}

	return nil
}
