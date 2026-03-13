package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/shared/pkg/valkey"
	vk "github.com/valkey-io/valkey-go"
)

var (
	ErrTokenNotFound = errors.New("refresh token not found")
)

//go:generate mockery --name TokenRepository
type TokenRepository interface {
	SaveRefreshToken(ctx context.Context, userID uuid.UUID, token string, ttl time.Duration) error
	GetUserIDByRefreshToken(ctx context.Context, token string) (uuid.UUID, error)
	DeleteRefreshToken(ctx context.Context, token string) error
}

type valkeyTokenRepository struct {
	client vk.Client
}

func NewTokenRepository(client vk.Client) TokenRepository {
	return &valkeyTokenRepository{client: client}
}

func (r *valkeyTokenRepository) SaveRefreshToken(ctx context.Context, userID uuid.UUID, token string, ttl time.Duration) error {
	key := valkey.AuthRefreshKey(token)
	err := r.client.Do(ctx, r.client.B().Set().Key(key).Value(userID.String()).Ex(ttl).Build()).Error()
	if err != nil {
		return fmt.Errorf("save refresh token to valkey: %w", err)
	}

	return nil
}

func (r *valkeyTokenRepository) GetUserIDByRefreshToken(ctx context.Context, token string) (uuid.UUID, error) {
	key := valkey.AuthRefreshKey(token)
	resp := r.client.Do(ctx, r.client.B().Get().Key(key).Build())
	if err := resp.Error(); err != nil {
		if vk.IsValkeyNil(err) {
			return uuid.Nil, ErrTokenNotFound
		}
		return uuid.Nil, fmt.Errorf("get refresh token from valkey: %w", err)
	}

	userIDStr, err := resp.ToString()
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse user id from valkey: %w", err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse uuid: %w", err)
	}

	return userID, nil
}

func (r *valkeyTokenRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	key := valkey.AuthRefreshKey(token)
	err := r.client.Do(ctx, r.client.B().Del().Key(key).Build()).Error()
	if err != nil {
		return fmt.Errorf("delete refresh token from valkey: %w", err)
	}

	return nil
}
