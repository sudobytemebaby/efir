package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/sudobytemebaby/efir/services/shared/pkg/valkey"
	vk "github.com/valkey-io/valkey-go"
)

const (
	ActionLogin    = "login"
	ActionRegister = "register"
)

// ErrRateLimitExceeded is returned when the rate limit is exceeded.
type ErrRateLimitExceeded struct {
	Action string
	Email  string
}

func (e *ErrRateLimitExceeded) Error() string {
	return fmt.Sprintf("rate limit exceeded for %s on %s", e.Action, e.Email)
}

// Limiter checks and increments rate limit counters.
//
//go:generate mockery --name Limiter
type Limiter interface {
	// Allow returns nil if the request is within the limit,
	// or *ErrRateLimitExceeded if the limit has been exceeded.
	Allow(ctx context.Context, action, email string) error
}

type valkeyLimiter struct {
	client vk.Client
	limit  int64
	window time.Duration
}

func NewValkeyLimiter(client vk.Client, limit int64, window time.Duration) Limiter {
	return &valkeyLimiter{
		client: client,
		limit:  limit,
		window: window,
	}
}

func (l *valkeyLimiter) Allow(ctx context.Context, action, email string) error {
	key := valkey.AuthRateLimitKey(action, email)

	// INCR atomically increments the counter and returns the new value.
	// On first call the key does not exist — Valkey treats it as 0 and returns 1.
	count, err := l.client.Do(ctx, l.client.B().Incr().Key(key).Build()).AsInt64()
	if err != nil {
		return fmt.Errorf("rate limit incr: %w", err)
	}

	// Set TTL only on first increment so the window starts fresh.
	if count == 1 {
		err = l.client.Do(ctx, l.client.B().Expire().Key(key).Seconds(int64(l.window.Seconds())).Build()).Error()
		if err != nil {
			return fmt.Errorf("rate limit expire: %w", err)
		}
	}

	if count > l.limit {
		return &ErrRateLimitExceeded{Action: action, Email: email}
	}

	return nil
}
