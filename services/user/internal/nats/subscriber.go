package nats

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
	sharednats "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
	"github.com/sudobytemebaby/efir/services/user/internal/service"
)

type subscriber struct {
	js  jetstream.JetStream
	svc service.UserService
}

func NewSubscriber(js jetstream.JetStream, svc service.UserService) *subscriber {
	return &subscriber{js: js, svc: svc}
}

func (s *subscriber) Start(ctx context.Context) error {
	consumer, err := sharednats.ProvisionConsumerWithRetry(ctx, s.js, StreamAuth, UserRegisteredConsumer())
	if err != nil {
		return err
	}

	msgs, err := consumer.Consume(func(msg jetstream.Msg) {
		s.handleMessage(ctx, msg)
	})
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		msgs.Drain()
	}()

	slog.Info("user service NATS subscriber started", "consumer", ConsumerUserRegistered)
	return nil
}

func (s *subscriber) handleMessage(ctx context.Context, msg jetstream.Msg) {
	var payload struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
	}

	if err := json.Unmarshal(msg.Data(), &payload); err != nil {
		slog.Error("failed to unmarshal user registered event", "error", err)
		if err := msg.Nak(); err != nil {
			slog.Error("failed to nak message", "error", err)
		}
		return
	}

	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		slog.Error("failed to parse user_id", "user_id", payload.UserID, "error", err)
		if err := msg.Nak(); err != nil {
			slog.Error("failed to nak message", "error", err)
		}
		return
	}

	_, err = s.svc.CreateUser(ctx, userID, payload.Email)
	if err != nil {
		slog.Error("failed to create user from event", "user_id", userID, "error", err)
		if err := msg.Nak(); err != nil {
			slog.Error("failed to nak message", "error", err)
		}
		return
	}

	slog.Info("user created from NATS event", "user_id", userID, "email", payload.Email)
	if err := msg.Ack(); err != nil {
		slog.Error("failed to ack message", "error", err)
	}
}
