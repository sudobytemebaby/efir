package handler

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sudobytemebaby/efir/services/room/internal/repository"
	"github.com/sudobytemebaby/efir/services/room/internal/service"
	roommocks "github.com/sudobytemebaby/efir/services/room/internal/service/mocks"
	roomv1 "github.com/sudobytemebaby/efir/services/shared/gen/room"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateRoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := roommocks.NewRoomService(t)
		h, err := NewRoomHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()

		expectedRoom := &repository.Room{
			ID:        roomID,
			Name:      "Test Room",
			Type:      repository.RoomTypeGroup,
			CreatedBy: userID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockSvc.On("CreateRoom", ctx, "Test Room", repository.RoomTypeGroup, userID, uuid.Nil).Return(expectedRoom, nil).Once()

		resp, err := h.CreateRoom(ctx, &roomv1.CreateRoomRequest{
			Name:      "Test Room",
			Type:      roomv1.RoomType_ROOM_TYPE_GROUP,
			CreatedBy: userID.String(),
		})

		require.NoError(t, err)
		assert.NotNil(t, resp.Room)
		assert.Equal(t, roomID.String(), resp.Room.RoomId)
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid created_by", func(t *testing.T) {
		mockSvc := roommocks.NewRoomService(t)
		h, err := NewRoomHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()

		_, err = h.CreateRoom(ctx, &roomv1.CreateRoomRequest{
			Name:      "Test Room",
			Type:      roomv1.RoomType_ROOM_TYPE_GROUP,
			CreatedBy: "invalid",
		})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		mockSvc.AssertExpectations(t)
	})

	t.Run("direct room already exists", func(t *testing.T) {
		mockSvc := roommocks.NewRoomService(t)
		h, err := NewRoomHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()
		userID := uuid.New()
		participantID := uuid.New()

		mockSvc.On("CreateRoom", ctx, "Test", repository.RoomTypeDirect, userID, participantID).Return(nil, service.ErrDirectRoomExists).Once()

		_, err = h.CreateRoom(ctx, &roomv1.CreateRoomRequest{
			Name:          "Test",
			Type:          roomv1.RoomType_ROOM_TYPE_DIRECT,
			CreatedBy:     userID.String(),
			ParticipantId: participantID.String(),
		})

		require.Error(t, err)
		assert.Equal(t, codes.AlreadyExists, status.Code(err))
		mockSvc.AssertExpectations(t)
	})
}

func TestGetRoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := roommocks.NewRoomService(t)
		h, err := NewRoomHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()
		roomID := uuid.New()

		expectedRoom := &repository.Room{
			ID:        roomID,
			Name:      "Test Room",
			Type:      repository.RoomTypeGroup,
			CreatedBy: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockSvc.On("GetRoom", ctx, roomID).Return(expectedRoom, nil).Once()

		resp, err := h.GetRoom(ctx, &roomv1.GetRoomRequest{RoomId: roomID.String()})

		require.NoError(t, err)
		assert.NotNil(t, resp.Room)
		mockSvc.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := roommocks.NewRoomService(t)
		h, err := NewRoomHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()
		roomID := uuid.New()

		mockSvc.On("GetRoom", ctx, roomID).Return(nil, service.ErrRoomNotFound).Once()

		_, err = h.GetRoom(ctx, &roomv1.GetRoomRequest{RoomId: roomID.String()})

		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
		mockSvc.AssertExpectations(t)
	})
}

func TestUpdateRoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := roommocks.NewRoomService(t)
		h, err := NewRoomHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()
		roomID := uuid.New()
		requesterID := uuid.New()
		name := "Updated Name"

		expectedRoom := &repository.Room{
			ID:        roomID,
			Name:      name,
			Type:      repository.RoomTypeGroup,
			CreatedBy: requesterID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockSvc.On("UpdateRoom", ctx, roomID, requesterID, name).Return(expectedRoom, nil).Once()

		resp, err := h.UpdateRoom(ctx, &roomv1.UpdateRoomRequest{
			RoomId:      roomID.String(),
			RequesterId: requesterID.String(),
			Name:        &name,
		})

		require.NoError(t, err)
		assert.Equal(t, name, resp.Room.Name)
		mockSvc.AssertExpectations(t)
	})

	t.Run("not owner", func(t *testing.T) {
		mockSvc := roommocks.NewRoomService(t)
		h, err := NewRoomHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()
		roomID := uuid.New()
		requesterID := uuid.New()
		name := "Updated Name"

		mockSvc.On("UpdateRoom", ctx, roomID, requesterID, name).Return(nil, service.ErrNotOwner).Once()

		_, err = h.UpdateRoom(ctx, &roomv1.UpdateRoomRequest{
			RoomId:      roomID.String(),
			RequesterId: requesterID.String(),
			Name:        &name,
		})

		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		mockSvc.AssertExpectations(t)
	})
}

func TestDeleteRoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := roommocks.NewRoomService(t)
		h, err := NewRoomHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()
		roomID := uuid.New()
		requesterID := uuid.New()

		mockSvc.On("DeleteRoom", ctx, roomID, requesterID).Return(nil).Once()

		_, err = h.DeleteRoom(ctx, &roomv1.DeleteRoomRequest{
			RoomId:      roomID.String(),
			RequesterId: requesterID.String(),
		})

		require.NoError(t, err)
		mockSvc.AssertExpectations(t)
	})
}

func TestAddMember(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := roommocks.NewRoomService(t)
		h, err := NewRoomHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()
		requesterID := uuid.New()

		mockSvc.On("AddMember", ctx, roomID, userID, requesterID).Return(nil).Once()

		_, err = h.AddMember(ctx, &roomv1.AddMemberRequest{
			RoomId:      roomID.String(),
			UserId:      userID.String(),
			RequesterId: requesterID.String(),
		})

		require.NoError(t, err)
		mockSvc.AssertExpectations(t)
	})

	t.Run("not member", func(t *testing.T) {
		mockSvc := roommocks.NewRoomService(t)
		h, err := NewRoomHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()
		requesterID := uuid.New()

		mockSvc.On("AddMember", ctx, roomID, userID, requesterID).Return(service.ErrNotMember).Once()

		_, err = h.AddMember(ctx, &roomv1.AddMemberRequest{
			RoomId:      roomID.String(),
			UserId:      userID.String(),
			RequesterId: requesterID.String(),
		})

		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		mockSvc.AssertExpectations(t)
	})
}

func TestGetRoomMembers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := roommocks.NewRoomService(t)
		h, err := NewRoomHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()
		roomID := uuid.New()

		members := []repository.RoomMember{
			{RoomID: roomID, UserID: uuid.New(), Role: repository.MemberRoleOwner, JoinedAt: time.Now()},
			{RoomID: roomID, UserID: uuid.New(), Role: repository.MemberRoleMember, JoinedAt: time.Now()},
		}

		mockSvc.On("GetRoomMembers", ctx, roomID).Return(members, nil).Once()

		resp, err := h.GetRoomMembers(ctx, &roomv1.GetRoomMembersRequest{RoomId: roomID.String()})

		require.NoError(t, err)
		assert.Len(t, resp.UserIds, 2)
		mockSvc.AssertExpectations(t)
	})
}

func TestIsMember(t *testing.T) {
	t.Run("is member", func(t *testing.T) {
		mockSvc := roommocks.NewRoomService(t)
		h, err := NewRoomHandler(mockSvc)
		require.NoError(t, err)

		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()

		mockSvc.On("IsMember", ctx, roomID, userID).Return(true, nil).Once()

		resp, err := h.IsMember(ctx, &roomv1.IsMemberRequest{
			RoomId: roomID.String(),
			UserId: userID.String(),
		})

		require.NoError(t, err)
		assert.True(t, resp.IsMember)
		mockSvc.AssertExpectations(t)
	})
}
