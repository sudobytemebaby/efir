package nats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type (
	StreamConfig   = jetstream.StreamConfig
	ConsumerConfig = jetstream.ConsumerConfig
)

func Connect(url, user, password string) (*nats.Conn, error) {
	nc, err := nats.Connect(
		url,
		nats.UserInfo(user, password),
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(-1),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to nats: %w", err)
	}
	return nc, nil
}

func New(nc *nats.Conn) (jetstream.JetStream, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("create jetstream context: %w", err)
	}
	return js, nil
}

func ProvisionStreams(ctx context.Context, js jetstream.JetStream, streams []StreamConfig) error {
	for _, cfg := range streams {
		if _, err := js.CreateOrUpdateStream(ctx, cfg); err != nil {
			return fmt.Errorf("provision stream %q: %w", cfg.Name, err)
		}
	}
	return nil
}

func ProvisionConsumer(ctx context.Context, js jetstream.JetStream, stream string, cfg ConsumerConfig) (jetstream.Consumer, error) {
	c, err := js.CreateOrUpdateConsumer(ctx, stream, cfg)
	if err != nil {
		return nil, fmt.Errorf("provision consumer %q on stream %q: %w", cfg.Durable, stream, err)
	}
	return c, nil
}

func ProvisionConsumerWithRetry(ctx context.Context, js jetstream.JetStream, stream string, cfg ConsumerConfig) (jetstream.Consumer, error) {
	for {
		c, err := js.CreateOrUpdateConsumer(ctx, stream, cfg)
		if err == nil {
			return c, nil
		}

		if errors.Is(err, jetstream.ErrStreamNotFound) {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("stream %q never appeared: %w", stream, ctx.Err())
			case <-time.After(2 * time.Second):
				continue
			}
		}

		return nil, fmt.Errorf("provision consumer %q on stream %q: %w", cfg.Durable, stream, err)
	}
}

func DefaultConsumerConfig(durable, filterSubject string) ConsumerConfig {
	return ConsumerConfig{
		Durable:       durable,
		FilterSubject: filterSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
		AckWait:       30 * time.Second,
	}
}
