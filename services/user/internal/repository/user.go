package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

type User struct {
	ID          uuid.UUID
	Username    string
	DisplayName string
	AvatarURL   *string
	Bio         *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

//go:generate mockery --name UserRepository
type UserRepository interface {
	CreateUser(ctx context.Context, id uuid.UUID, username, displayName string) (*User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, displayName, avatarURL, bio *string) (*User, error)
}

type pgUserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &pgUserRepository{pool: pool}
}

func (r *pgUserRepository) CreateUser(ctx context.Context, id uuid.UUID, username, displayName string) (*User, error) {
	const query = `
		INSERT INTO users (id, username, display_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO NOTHING
		RETURNING id, username, display_name, avatar_url, bio, created_at, updated_at
	`

	user := &User{}
	err := r.pool.QueryRow(ctx, query, id, username, displayName).Scan(
		&user.ID, &user.Username, &user.DisplayName, &user.AvatarURL, &user.Bio, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (r *pgUserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const query = `
		SELECT id, username, display_name, avatar_url, bio, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.DisplayName, &user.AvatarURL, &user.Bio, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

func (r *pgUserRepository) GetUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	const query = `
		SELECT id, username, display_name, avatar_url, bio, created_at, updated_at
		FROM users
		WHERE id = ANY($1)
	`

	rows, err := r.pool.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("get users by ids: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(
			&user.ID, &user.Username, &user.DisplayName, &user.AvatarURL, &user.Bio, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return users, nil
}

func (r *pgUserRepository) UpdateUser(ctx context.Context, id uuid.UUID, displayName, avatarURL, bio *string) (*User, error) {
	const query = `
		UPDATE users
		SET display_name = COALESCE($2, display_name),
			avatar_url = COALESCE($3, avatar_url),
			bio = COALESCE($4, bio),
			updated_at = now()
		WHERE id = $1
		RETURNING id, username, display_name, avatar_url, bio, created_at, updated_at
	`

	user := &User{}
	err := r.pool.QueryRow(ctx, query, id, displayName, avatarURL, bio).Scan(
		&user.ID, &user.Username, &user.DisplayName, &user.AvatarURL, &user.Bio, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("update user: %w", err)
	}

	return user, nil
}
