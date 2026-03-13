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
	ErrAccountNotFound = errors.New("account not found")
)

type Account struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

//go:generate mockery --name AccountRepository
type AccountRepository interface {
	CreateAccount(ctx context.Context, email, passwordHash string) (*Account, error)
	GetAccountByEmail(ctx context.Context, email string) (*Account, error)
	GetAccountByID(ctx context.Context, id uuid.UUID) (*Account, error)
}

type pgAccountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) AccountRepository {
	return &pgAccountRepository{pool: pool}
}

func (r *pgAccountRepository) CreateAccount(ctx context.Context, email, passwordHash string) (*Account, error) {
	const query = `
		INSERT INTO accounts (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at, updated_at
	`

	acc := &Account{}
	err := r.pool.QueryRow(ctx, query, email, passwordHash).Scan(
		&acc.ID, &acc.Email, &acc.PasswordHash, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	return acc, nil
}

func (r *pgAccountRepository) GetAccountByEmail(ctx context.Context, email string) (*Account, error) {
	const query = `
		SELECT id, email, password_hash, created_at, updated_at
		FROM accounts
		WHERE email = $1
	`

	acc := &Account{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&acc.ID, &acc.Email, &acc.PasswordHash, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("get account by email: %w", err)
	}

	return acc, nil
}

func (r *pgAccountRepository) GetAccountByID(ctx context.Context, id uuid.UUID) (*Account, error) {
	const query = `
		SELECT id, email, password_hash, created_at, updated_at
		FROM accounts
		WHERE id = $1
	`

	acc := &Account{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&acc.ID, &acc.Email, &acc.PasswordHash, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("get account by id: %w", err)
	}

	return acc, nil
}
