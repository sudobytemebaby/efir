package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/sudobytemebaby/efir/services/room/internal/repository"
	repomocks "github.com/sudobytemebaby/efir/services/room/internal/repository/mocks"
)

// mockPublisher is a local implementation to avoid import cycle with service/mocks.
type mockPublisher struct {
	mock.Mock
}

func (m *mockPublisher) PublishMembershipChanged(ctx context.Context, roomID, userID uuid.UUID, action string, recipientIDs []uuid.UUID) error {
	args := m.Called(ctx, roomID, userID, action, recipientIDs)
	return args.Error(0)
}

func (m *mockPublisher) PublishRoomUpdated(ctx context.Context, roomID uuid.UUID, name string, recipientIDs []uuid.UUID) error {
	args := m.Called(ctx, roomID, name, recipientIDs)
	return args.Error(0)
}

func TestCreateRoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()

		repoRoom := &repository.Room{
			ID:        roomID,
			Name:      "Test Room",
			Type:      repository.RoomTypeGroup,
			CreatedBy: userID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("CreateRoom", ctx, "Test Room", repository.RoomTypeGroup, userID).Return(repoRoom, nil).Once()
		mockRepo.On("AddMember", ctx, roomID, userID, repository.MemberRoleOwner).Return(&repository.RoomMember{}, nil).Once()

		room, err := svc.CreateRoom(ctx, "Test Room", RoomTypeGroup, userID, uuid.Nil)

		require.NoError(t, err)
		assert.Equal(t, roomID, room.ID)
		assert.Equal(t, "Test Room", room.Name)
		assert.Equal(t, RoomTypeGroup, room.Type)
		mockRepo.AssertExpectations(t)
	})

	t.Run("direct room already exists", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		userID := uuid.New()
		participantID := uuid.New()
		existingRoomID := uuid.New()

		existingRoom := &repository.Room{
			ID:        existingRoomID,
			Name:      "Existing",
			Type:      repository.RoomTypeDirect,
			CreatedBy: userID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("GetDirectRoomByUsers", ctx, userID, participantID).Return(existingRoom, nil).Once()

		room, err := svc.CreateRoom(ctx, "New Room", RoomTypeDirect, userID, participantID)

		require.ErrorIs(t, err, ErrDirectRoomExists)
		assert.Nil(t, room)
		mockRepo.AssertExpectations(t)
	})

	t.Run("direct room success", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()
		participantID := uuid.New()

		repoRoom := &repository.Room{
			ID:        roomID,
			Name:      "Direct Room",
			Type:      repository.RoomTypeDirect,
			CreatedBy: userID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("GetDirectRoomByUsers", ctx, userID, participantID).Return(nil, repository.ErrRoomNotFound).Once()
		mockRepo.On("CreateRoom", ctx, "Direct Room", repository.RoomTypeDirect, userID).Return(repoRoom, nil).Once()
		mockRepo.On("AddMember", ctx, roomID, userID, repository.MemberRoleOwner).Return(&repository.RoomMember{}, nil).Once()
		mockRepo.On("AddMember", ctx, roomID, participantID, repository.MemberRoleMember).Return(&repository.RoomMember{}, nil).Once()

		room, err := svc.CreateRoom(ctx, "Direct Room", RoomTypeDirect, userID, participantID)

		require.NoError(t, err)
		assert.Equal(t, roomID, room.ID)
		assert.Equal(t, RoomTypeDirect, room.Type)
		mockRepo.AssertExpectations(t)
	})
}

func TestGetRoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()

		repoRoom := &repository.Room{
			ID:        roomID,
			Name:      "Test Room",
			Type:      repository.RoomTypeGroup,
			CreatedBy: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("GetRoomByID", ctx, roomID).Return(repoRoom, nil).Once()

		room, err := svc.GetRoom(ctx, roomID)

		require.NoError(t, err)
		assert.Equal(t, roomID, room.ID)
		assert.Equal(t, "Test Room", room.Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()

		mockRepo.On("GetRoomByID", ctx, roomID).Return(nil, repository.ErrRoomNotFound).Once()

		room, err := svc.GetRoom(ctx, roomID)

		require.ErrorIs(t, err, ErrRoomNotFound)
		assert.Nil(t, room)
		mockRepo.AssertExpectations(t)
	})
}

func TestUpdateRoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		requesterID := uuid.New()

		room := &repository.Room{
			ID:        roomID,
			Name:      "Old Name",
			Type:      repository.RoomTypeGroup,
			CreatedBy: requesterID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		updatedRoom := &repository.Room{
			ID:        roomID,
			Name:      "New Name",
			Type:      repository.RoomTypeGroup,
			CreatedBy: requesterID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("GetRoomByID", ctx, roomID).Return(room, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRoleOwner, nil).Once()
		mockRepo.On("UpdateRoom", ctx, roomID, "New Name").Return(updatedRoom, nil).Once()
		mockRepo.On("GetRoomMembers", ctx, roomID).Return([]repository.RoomMember{}, nil).Once()
		pub.On("PublishRoomUpdated", ctx, roomID, "New Name", []uuid.UUID{}).Return(nil).Once()

		result, err := svc.UpdateRoom(ctx, roomID, requesterID, "New Name")

		require.NoError(t, err)
		assert.Equal(t, "New Name", result.Name)
		mockRepo.AssertExpectations(t)
		pub.AssertExpectations(t)
	})

	t.Run("not owner", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		requesterID := uuid.New()

		room := &repository.Room{
			ID:        roomID,
			Name:      "Old Name",
			Type:      repository.RoomTypeGroup,
			CreatedBy: requesterID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("GetRoomByID", ctx, roomID).Return(room, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRoleMember, nil).Once()

		_, err := svc.UpdateRoom(ctx, roomID, requesterID, "New Name")

		require.ErrorIs(t, err, ErrNotOwner)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not member", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		requesterID := uuid.New()

		room := &repository.Room{
			ID:        roomID,
			Name:      "Old Name",
			Type:      repository.RoomTypeGroup,
			CreatedBy: requesterID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("GetRoomByID", ctx, roomID).Return(room, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRole(""), repository.ErrMemberNotFound).Once()

		_, err := svc.UpdateRoom(ctx, roomID, requesterID, "New Name")

		require.ErrorIs(t, err, ErrNotMember)
		mockRepo.AssertExpectations(t)
	})
}

func TestDeleteRoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		requesterID := uuid.New()

		room := &repository.Room{
			ID:        roomID,
			Name:      "Test Room",
			Type:      repository.RoomTypeGroup,
			CreatedBy: requesterID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("GetRoomByID", ctx, roomID).Return(room, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRoleOwner, nil).Once()
		mockRepo.On("DeleteRoom", ctx, roomID).Return(nil).Once()

		err := svc.DeleteRoom(ctx, roomID, requesterID)

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not owner", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		requesterID := uuid.New()

		room := &repository.Room{
			ID:        roomID,
			Name:      "Test Room",
			Type:      repository.RoomTypeGroup,
			CreatedBy: requesterID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("GetRoomByID", ctx, roomID).Return(room, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRoleMember, nil).Once()

		err := svc.DeleteRoom(ctx, roomID, requesterID)

		require.ErrorIs(t, err, ErrNotOwner)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not member", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		requesterID := uuid.New()

		room := &repository.Room{
			ID:        roomID,
			Name:      "Test Room",
			Type:      repository.RoomTypeGroup,
			CreatedBy: requesterID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("GetRoomByID", ctx, roomID).Return(room, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRole(""), repository.ErrMemberNotFound).Once()

		err := svc.DeleteRoom(ctx, roomID, requesterID)

		require.ErrorIs(t, err, ErrNotMember)
		mockRepo.AssertExpectations(t)
	})
}

func TestAddMember(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()
		requesterID := uuid.New()

		room := &repository.Room{
			ID:        roomID,
			Name:      "Test Room",
			Type:      repository.RoomTypeGroup,
			CreatedBy: requesterID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		members := []repository.RoomMember{
			{RoomID: roomID, UserID: requesterID, Role: repository.MemberRoleOwner, JoinedAt: time.Now()},
			{RoomID: roomID, UserID: userID, Role: repository.MemberRoleMember, JoinedAt: time.Now()},
		}

		mockRepo.On("GetRoomByID", ctx, roomID).Return(room, nil).Once()
		mockRepo.On("IsMember", ctx, roomID, requesterID).Return(true, nil).Once()
		mockRepo.On("AddMember", ctx, roomID, userID, repository.MemberRoleMember).Return(&repository.RoomMember{}, nil).Once()
		mockRepo.On("GetRoomMembers", ctx, roomID).Return(members, nil).Once()
		pub.On("PublishMembershipChanged", ctx, roomID, userID, "added", mock.Anything).Return(nil).Once()

		err := svc.AddMember(ctx, roomID, userID, requesterID)

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
		pub.AssertExpectations(t)
	})
}

func TestRemoveMember(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()
		requesterID := uuid.New()

		room := &repository.Room{
			ID:        roomID,
			Name:      "Test Room",
			Type:      repository.RoomTypeGroup,
			CreatedBy: requesterID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		members := []repository.RoomMember{
			{RoomID: roomID, UserID: requesterID, Role: repository.MemberRoleOwner, JoinedAt: time.Now()},
		}

		mockRepo.On("GetRoomByID", ctx, roomID).Return(room, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRoleOwner, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, userID).Return(repository.MemberRoleMember, nil).Once()
		mockRepo.On("RemoveMember", ctx, roomID, userID).Return(nil).Once()
		mockRepo.On("GetRoomMembers", ctx, roomID).Return(members, nil).Once()
		pub.On("PublishMembershipChanged", ctx, roomID, userID, "removed", mock.Anything).Return(nil).Once()

		err := svc.RemoveMember(ctx, roomID, userID, requesterID)

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
		pub.AssertExpectations(t)
	})

	t.Run("cannot remove owner", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		ownerID := uuid.New()
		requesterID := uuid.New()

		room := &repository.Room{
			ID:        roomID,
			Name:      "Test Room",
			Type:      repository.RoomTypeGroup,
			CreatedBy: requesterID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("GetRoomByID", ctx, roomID).Return(room, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRoleOwner, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, ownerID).Return(repository.MemberRoleOwner, nil).Once()

		err := svc.RemoveMember(ctx, roomID, ownerID, requesterID)

		require.ErrorIs(t, err, ErrCannotRemoveOwner)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not owner", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()
		requesterID := uuid.New()

		room := &repository.Room{
			ID:        roomID,
			Name:      "Test Room",
			Type:      repository.RoomTypeGroup,
			CreatedBy: requesterID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("GetRoomByID", ctx, roomID).Return(room, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRoleMember, nil).Once()

		err := svc.RemoveMember(ctx, roomID, userID, requesterID)

		require.ErrorIs(t, err, ErrNotOwner)
		mockRepo.AssertExpectations(t)
	})

	t.Run("target not member", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()
		requesterID := uuid.New()

		room := &repository.Room{
			ID:        roomID,
			Name:      "Test Room",
			Type:      repository.RoomTypeGroup,
			CreatedBy: requesterID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("GetRoomByID", ctx, roomID).Return(room, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRoleOwner, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, userID).Return(repository.MemberRole(""), repository.ErrMemberNotFound).Once()

		err := svc.RemoveMember(ctx, roomID, userID, requesterID)

		require.ErrorIs(t, err, ErrMemberNotFound)
		mockRepo.AssertExpectations(t)
	})

	t.Run("member can remove themselves", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()

		room := &repository.Room{
			ID:        roomID,
			Name:      "Test Room",
			Type:      repository.RoomTypeGroup,
			CreatedBy: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		members := []repository.RoomMember{
			{RoomID: roomID, UserID: uuid.New(), Role: repository.MemberRoleOwner, JoinedAt: time.Now()},
		}

		mockRepo.On("GetRoomByID", ctx, roomID).Return(room, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, userID).Return(repository.MemberRoleMember, nil).Once()
		mockRepo.On("RemoveMember", ctx, roomID, userID).Return(nil).Once()
		mockRepo.On("GetRoomMembers", ctx, roomID).Return(members, nil).Once()
		pub.On("PublishMembershipChanged", ctx, roomID, userID, "removed", mock.Anything).Return(nil).Once()

		err := svc.RemoveMember(ctx, roomID, userID, userID)

		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
		pub.AssertExpectations(t)
	})

	t.Run("owner cannot remove themselves", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		ownerID := uuid.New()

		room := &repository.Room{
			ID:        roomID,
			Name:      "Test Room",
			Type:      repository.RoomTypeGroup,
			CreatedBy: ownerID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.On("GetRoomByID", ctx, roomID).Return(room, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, ownerID).Return(repository.MemberRoleOwner, nil).Once()

		err := svc.RemoveMember(ctx, roomID, ownerID, ownerID)

		require.ErrorIs(t, err, ErrCannotRemoveOwner)
		mockRepo.AssertExpectations(t)
	})
}

func TestGetRoomMembers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()

		members := []repository.RoomMember{
			{RoomID: roomID, UserID: uuid.New(), Role: repository.MemberRoleOwner, JoinedAt: time.Now()},
			{RoomID: roomID, UserID: uuid.New(), Role: repository.MemberRoleMember, JoinedAt: time.Now()},
		}

		mockRepo.On("GetRoomByID", ctx, roomID).Return(&repository.Room{ID: roomID}, nil).Once()
		mockRepo.On("GetRoomMembers", ctx, roomID).Return(members, nil).Once()

		result, err := svc.GetRoomMembers(ctx, roomID)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		mockRepo.AssertExpectations(t)
	})
}

func TestIsMember(t *testing.T) {
	t.Run("is member", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()

		mockRepo.On("IsMember", ctx, roomID, userID).Return(true, nil).Once()

		isMember, err := svc.IsMember(ctx, roomID, userID)

		require.NoError(t, err)
		assert.True(t, isMember)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not member", func(t *testing.T) {
		mockRepo := repomocks.NewRoomRepository(t)
		pub := &mockPublisher{}
		svc := NewRoomService(mockRepo, pub)

		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()

		mockRepo.On("IsMember", ctx, roomID, userID).Return(false, nil).Once()

		isMember, err := svc.IsMember(ctx, roomID, userID)

		require.NoError(t, err)
		assert.False(t, isMember)
		mockRepo.AssertExpectations(t)
	})
}
