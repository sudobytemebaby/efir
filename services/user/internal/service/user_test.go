package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/sudobytemebaby/efir/services/user/internal/repository"
	"github.com/sudobytemebaby/efir/services/user/internal/repository/mocks"
)

func TestCreateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := mocks.NewUserRepository(t)
		svc := NewUserService(mockRepo)

		ctx := context.Background()
		userID := uuid.New()
		email := "john@example.com"

		expectedUser := &repository.User{
			ID:          userID,
			Username:    "john",
			DisplayName: "john",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockRepo.On("CreateUser", ctx, userID, "john", "john").Return(expectedUser, nil).Once()

		user, err := svc.CreateUser(ctx, userID, email)

		require.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user already exists (idempotent)", func(t *testing.T) {
		mockRepo := mocks.NewUserRepository(t)
		svc := NewUserService(mockRepo)

		ctx := context.Background()
		userID := uuid.New()
		email := "john@example.com"

		existingUser := &repository.User{
			ID:          userID,
			Username:    "john",
			DisplayName: "john",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockRepo.On("CreateUser", ctx, userID, "john", "john").Return(nil, repository.ErrUserAlreadyExists).Once()
		mockRepo.On("GetUserByID", ctx, userID).Return(existingUser, nil).Once()

		user, err := svc.CreateUser(ctx, userID, email)

		require.NoError(t, err)
		assert.Equal(t, existingUser, user)
		mockRepo.AssertExpectations(t)
	})
}

func TestGetUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := mocks.NewUserRepository(t)
		svc := NewUserService(mockRepo)

		ctx := context.Background()
		userID := uuid.New()

		expectedUser := &repository.User{
			ID:          userID,
			Username:    "john",
			DisplayName: "John Doe",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockRepo.On("GetUserByID", ctx, userID).Return(expectedUser, nil).Once()

		user, err := svc.GetUser(ctx, userID)

		require.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := mocks.NewUserRepository(t)
		svc := NewUserService(mockRepo)

		ctx := context.Background()
		userID := uuid.New()

		mockRepo.On("GetUserByID", ctx, userID).Return(nil, repository.ErrUserNotFound).Once()

		user, err := svc.GetUser(ctx, userID)

		require.ErrorIs(t, err, ErrUserNotFound)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})
}

func TestGetUsers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := mocks.NewUserRepository(t)
		svc := NewUserService(mockRepo)

		ctx := context.Background()
		userID1 := uuid.New()
		userID2 := uuid.New()

		expectedUsers := []repository.User{
			{
				ID:          userID1,
				Username:    "john",
				DisplayName: "John",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				ID:          userID2,
				Username:    "jane",
				DisplayName: "Jane",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		}

		mockRepo.On("GetUsersByIDs", ctx, mock.Anything).Return(expectedUsers, nil).Once()

		users, err := svc.GetUsers(ctx, []uuid.UUID{userID1, userID2})

		require.NoError(t, err)
		assert.Len(t, users, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty list", func(t *testing.T) {
		mockRepo := mocks.NewUserRepository(t)
		svc := NewUserService(mockRepo)

		ctx := context.Background()

		mockRepo.On("GetUsersByIDs", ctx, mock.Anything).Return([]repository.User{}, nil).Once()

		users, err := svc.GetUsers(ctx, []uuid.UUID{})

		require.NoError(t, err)
		assert.Empty(t, users)
		mockRepo.AssertExpectations(t)
	})
}

func TestUpdateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := mocks.NewUserRepository(t)
		svc := NewUserService(mockRepo)

		ctx := context.Background()
		userID := uuid.New()
		displayName := "John Updated"
		avatarURL := "https://example.com/avatar.png"
		bio := "Hello world"

		expectedUser := &repository.User{
			ID:          userID,
			Username:    "john",
			DisplayName: displayName,
			AvatarURL:   &avatarURL,
			Bio:         &bio,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockRepo.On("UpdateUser", ctx, userID, &displayName, &avatarURL, &bio).Return(expectedUser, nil).Once()

		user, err := svc.UpdateUser(ctx, userID, &displayName, &avatarURL, &bio)

		require.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := mocks.NewUserRepository(t)
		svc := NewUserService(mockRepo)

		ctx := context.Background()
		userID := uuid.New()
		displayName := "John Updated"
		avatarURL := "https://example.com/avatar.png"
		bio := "Hello world"

		mockRepo.On("UpdateUser", ctx, userID, &displayName, &avatarURL, &bio).Return(nil, repository.ErrUserNotFound).Once()

		user, err := svc.UpdateUser(ctx, userID, &displayName, &avatarURL, &bio)

		require.ErrorIs(t, err, ErrUserNotFound)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})
}

func TestGenerateUsernameFromEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected string
	}{
		{"john@example.com", "john"},
		{"Jane.Doe@company.org", "jane.doe"},
		{"user+tag@domain.com", "user+tag"},
		{"just-a-name", "just-a-name"},
	}

	for _, tt := range tests {
		result := generateUsernameFromEmail(tt.email)
		assert.Equal(t, tt.expected, result)
	}
}
