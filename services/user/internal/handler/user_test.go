package handler

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	userv1 "github.com/sudobytemebaby/efir/services/shared/gen/user"
	"github.com/sudobytemebaby/efir/services/user/internal/repository"
	"github.com/sudobytemebaby/efir/services/user/internal/service"
	"github.com/sudobytemebaby/efir/services/user/internal/service/mocks"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := mocks.NewUserService(t)
		h, err := NewUserHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()
		userID := uuid.New()

		expectedUser := &repository.User{
			ID:          userID,
			Username:    "john",
			DisplayName: "John Doe",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockSvc.On("GetUser", ctx, userID).Return(expectedUser, nil).Once()

		resp, err := h.GetUser(ctx, &userv1.GetUserRequest{UserId: userID.String()})

		require.NoError(t, err)
		assert.NotNil(t, resp.User)
		assert.Equal(t, userID.String(), resp.User.UserId)
		mockSvc.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := mocks.NewUserService(t)
		h, err := NewUserHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()
		userID := uuid.New()

		mockSvc.On("GetUser", ctx, userID).Return(nil, service.ErrUserNotFound).Once()

		_, err = h.GetUser(ctx, &userv1.GetUserRequest{UserId: userID.String()})

		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid user_id", func(t *testing.T) {
		mockSvc := mocks.NewUserService(t)
		h, err := NewUserHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()

		_, err = h.GetUser(ctx, &userv1.GetUserRequest{UserId: "invalid"})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		mockSvc.AssertExpectations(t)
	})
}

func TestGetUsersByIds(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := mocks.NewUserService(t)
		h, err := NewUserHandler(mockSvc)
		require.NoError(t, err)

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

		mockSvc.On("GetUsers", ctx, mock.Anything).Return(expectedUsers, nil).Once()

		resp, err := h.GetUsersByIds(ctx, &userv1.GetUsersByIdsRequest{
			UserIds: []string{userID1.String(), userID2.String()},
		})

		require.NoError(t, err)
		assert.Len(t, resp.Users, 2)
		mockSvc.AssertExpectations(t)
	})

	t.Run("empty list", func(t *testing.T) {
		mockSvc := mocks.NewUserService(t)
		h, err := NewUserHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()

		_, err = h.GetUsersByIds(ctx, &userv1.GetUsersByIdsRequest{UserIds: []string{}})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid user_id", func(t *testing.T) {
		mockSvc := mocks.NewUserService(t)
		h, err := NewUserHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()

		_, err = h.GetUsersByIds(ctx, &userv1.GetUsersByIdsRequest{
			UserIds: []string{"invalid"},
		})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		mockSvc.AssertExpectations(t)
	})
}

func TestUpdateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := mocks.NewUserService(t)
		h, err := NewUserHandler(mockSvc)
		require.NoError(t, err)

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

		mockSvc.On("UpdateUser", ctx, userID, &displayName, &avatarURL, &bio).Return(expectedUser, nil).Once()

		resp, err := h.UpdateUser(ctx, &userv1.UpdateUserRequest{
			UserId:      userID.String(),
			DisplayName: &displayName,
			AvatarUrl:   &avatarURL,
			Bio:         &bio,
		})

		require.NoError(t, err)
		assert.NotNil(t, resp.User)
		assert.Equal(t, displayName, resp.User.DisplayName)
		mockSvc.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := mocks.NewUserService(t)
		h, err := NewUserHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()
		userID := uuid.New()
		displayName := "John Updated"
		avatarURL := "https://example.com/avatar.png"
		bio := "Hello world"

		mockSvc.On("UpdateUser", ctx, userID, &displayName, &avatarURL, &bio).Return(nil, service.ErrUserNotFound).Once()

		_, err = h.UpdateUser(ctx, &userv1.UpdateUserRequest{
			UserId:      userID.String(),
			DisplayName: &displayName,
			AvatarUrl:   &avatarURL,
			Bio:         &bio,
		})

		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid user_id", func(t *testing.T) {
		mockSvc := mocks.NewUserService(t)
		h, err := NewUserHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()
		displayName := "John Updated"

		_, err = h.UpdateUser(ctx, &userv1.UpdateUserRequest{
			UserId:      "invalid",
			DisplayName: &displayName,
		})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		mockSvc.AssertExpectations(t)
	})
}

func TestMapUserToProto(t *testing.T) {
	avatarURL := "https://example.com/avatar.png"
	bio := "Hello world"
	now := time.Now()

	user := &repository.User{
		ID:          uuid.New(),
		Username:    "john",
		DisplayName: "John Doe",
		AvatarURL:   &avatarURL,
		Bio:         &bio,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	protoUser := mapUserToProto(user)

	assert.Equal(t, user.ID.String(), protoUser.UserId)
	assert.Equal(t, user.Username, protoUser.Username)
	assert.Equal(t, user.DisplayName, protoUser.DisplayName)
	assert.Equal(t, user.AvatarURL, protoUser.AvatarUrl)
	assert.Equal(t, user.Bio, protoUser.Bio)
}
