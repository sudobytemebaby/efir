package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/user/internal/repository"
)

var ErrUserNotFound = errors.New("user not found")

//go:generate mockery --name UserService
type UserService interface {
	CreateUser(ctx context.Context, userID uuid.UUID, email string) (*repository.User, error)
	GetUser(ctx context.Context, userID uuid.UUID) (*repository.User, error)
	GetUsers(ctx context.Context, userIDs []uuid.UUID) ([]repository.User, error)
	UpdateUser(ctx context.Context, userID uuid.UUID, displayName, avatarURL, bio *string) (*repository.User, error)
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) CreateUser(ctx context.Context, userID uuid.UUID, email string) (*repository.User, error) {
	username := generateUsernameFromEmail(email)
	displayName := username

	user, err := s.userRepo.CreateUser(ctx, userID, username, displayName)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (s *userService) GetUser(ctx context.Context, userID uuid.UUID) (*repository.User, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	return user, nil
}

func (s *userService) GetUsers(ctx context.Context, userIDs []uuid.UUID) ([]repository.User, error) {
	users, err := s.userRepo.GetUsersByIDs(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("get users: %w", err)
	}

	return users, nil
}

func (s *userService) UpdateUser(ctx context.Context, userID uuid.UUID, displayName, avatarURL, bio *string) (*repository.User, error) {
	user, err := s.userRepo.UpdateUser(ctx, userID, displayName, avatarURL, bio)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("update user: %w", err)
	}

	return user, nil
}

func generateUsernameFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return strings.ToLower(parts[0])
	}
	return strings.ToLower(email)
}
