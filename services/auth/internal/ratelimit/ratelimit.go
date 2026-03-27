package ratelimit

import (
	"context"
	"fmt"
	"strconv"
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
	return fmt.Sprintf("rate limit exceeded for %s", e.Action)
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
	limit  int
	window time.Duration
}

func NewValkeyLimiter(client vk.Client, limit int, window time.Duration) Limiter {
	return &valkeyLimiter{
		client: client,
		limit:  limit,
		window: window,
	}
}

func (l *valkeyLimiter) Allow(ctx context.Context, action, email string) error {
	key := valkey.AuthRateLimitKey(action, email)
	ttlSeconds := strconv.Itoa(int(l.window.Seconds()))

	result, err := l.client.Do(ctx, l.client.B().Eval().Script(valkey.IncrWithExpiryScript).Numkeys(1).Key(key).Arg(ttlSeconds).Build()).AsInt64()
	if err != nil {
		return fmt.Errorf("rate limit check: %w", err)
	}

	if result > int64(l.limit) {
		return &ErrRateLimitExceeded{Action: action, Email: email}
	}

	return nil
}
