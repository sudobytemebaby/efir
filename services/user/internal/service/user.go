package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/shared/pkg"
	"github.com/sudobytemebaby/efir/services/user/internal/repository"
)

var ErrUserNotFound = errors.New("user not found")

//go:generate mockery --name UserService
type UserService interface {
	CreateUser(ctx context.Context, userID uuid.UUID, email string) (*User, error)
	GetUser(ctx context.Context, userID uuid.UUID) (*User, error)
	GetUsers(ctx context.Context, userIDs []uuid.UUID) ([]User, error)
	UpdateUser(ctx context.Context, userID uuid.UUID, displayName, avatarURL, bio *string) (*User, error)
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

// CreateUser is idempotent: if the user already exists it returns the existing record.
// This handles duplicate delivery of the auth.user.registered NATS event.
// Username generation is retried up to maxAttempts times to resolve random collisions.
func (s *userService) CreateUser(ctx context.Context, userID uuid.UUID, email string) (*User, error) {
	const maxAttempts = 3

	for attempt := 0; attempt < maxAttempts; attempt++ {
		username, err := pkg.GenerateUsername()
		if err != nil {
			return nil, fmt.Errorf("generate username: %w", err)
		}

		user, err := s.userRepo.CreateUser(ctx, userID, username, username)
		if err == nil {
			return toUser(user), nil
		}

		if errors.Is(err, repository.ErrUserAlreadyExists) {
			existing, err := s.userRepo.GetUserByID(ctx, userID)
			if err != nil {
				return nil, fmt.Errorf("get existing user: %w", err)
			}
			return toUser(existing), nil
		}

		if errors.Is(err, repository.ErrUsernameAlreadyExists) {
			continue
		}

		return nil, fmt.Errorf("create user: %w", err)
	}

	return nil, fmt.Errorf("failed to generate unique username after %d attempts", maxAttempts)
}

func (s *userService) GetUser(ctx context.Context, userID uuid.UUID) (*User, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	return toUser(user), nil
}

func (s *userService) GetUsers(ctx context.Context, userIDs []uuid.UUID) ([]User, error) {
	users, err := s.userRepo.GetUsersByIDs(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("get users: %w", err)
	}

	result := make([]User, len(users))
	for i, u := range users {
		result[i] = *toUser(&u)
	}

	return result, nil
}

func (s *userService) UpdateUser(ctx context.Context, userID uuid.UUID, displayName, avatarURL, bio *string) (*User, error) {
	user, err := s.userRepo.UpdateUser(ctx, userID, displayName, avatarURL, bio)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("update user: %w", err)
	}

	return toUser(user), nil
}
