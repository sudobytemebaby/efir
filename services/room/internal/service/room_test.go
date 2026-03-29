package service_test

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
	"github.com/sudobytemebaby/efir/services/room/internal/service"
	svcmocks "github.com/sudobytemebaby/efir/services/room/internal/service/mocks"
)

func newSvc(t *testing.T) (service.RoomService, *repomocks.RoomRepository, *svcmocks.Publisher) {
	t.Helper()
	mockRepo := repomocks.NewRoomRepository(t)
	pub := svcmocks.NewPublisher(t)
	svc := service.NewRoomService(mockRepo, pub)
	return svc, mockRepo, pub
}

func repoRoom(roomID, createdBy uuid.UUID, name string, typ repository.RoomType) *repository.Room {
	return &repository.Room{
		ID:        roomID,
		Name:      name,
		Type:      typ,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestCreateRoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()

		mockRepo.On("CreateRoom", ctx, "Test Room", repository.RoomTypeGroup, userID).
			Return(repoRoom(roomID, userID, "Test Room", repository.RoomTypeGroup), nil).Once()
		mockRepo.On("AddMember", ctx, roomID, userID, repository.MemberRoleOwner).
			Return(&repository.RoomMember{}, nil).Once()

		room, err := svc.CreateRoom(ctx, "Test Room", service.RoomTypeGroup, userID, uuid.Nil)

		require.NoError(t, err)
		assert.Equal(t, roomID, room.ID)
		assert.Equal(t, "Test Room", room.Name)
		assert.Equal(t, service.RoomTypeGroup, room.Type)
	})

	t.Run("direct room already exists", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
		ctx := context.Background()
		userID := uuid.New()
		participantID := uuid.New()
		existingRoomID := uuid.New()

		mockRepo.On("GetDirectRoomByUsers", ctx, userID, participantID).
			Return(repoRoom(existingRoomID, userID, "Existing", repository.RoomTypeDirect), nil).Once()

		room, err := svc.CreateRoom(ctx, "New Room", service.RoomTypeDirect, userID, participantID)

		require.ErrorIs(t, err, service.ErrDirectRoomExists)
		assert.Nil(t, room)
	})

	t.Run("direct room success", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()
		participantID := uuid.New()

		mockRepo.On("GetDirectRoomByUsers", ctx, userID, participantID).
			Return(nil, repository.ErrRoomNotFound).Once()
		mockRepo.On("CreateRoom", ctx, "Direct Room", repository.RoomTypeDirect, userID).
			Return(repoRoom(roomID, userID, "Direct Room", repository.RoomTypeDirect), nil).Once()
		mockRepo.On("AddMember", ctx, roomID, userID, repository.MemberRoleOwner).
			Return(&repository.RoomMember{}, nil).Once()
		mockRepo.On("AddMember", ctx, roomID, participantID, repository.MemberRoleMember).
			Return(&repository.RoomMember{}, nil).Once()

		room, err := svc.CreateRoom(ctx, "Direct Room", service.RoomTypeDirect, userID, participantID)

		require.NoError(t, err)
		assert.Equal(t, roomID, room.ID)
		assert.Equal(t, service.RoomTypeDirect, room.Type)
	})
}

func TestGetRoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()

		mockRepo.On("GetRoomByID", ctx, roomID).
			Return(repoRoom(roomID, uuid.New(), "Test Room", repository.RoomTypeGroup), nil).Once()

		room, err := svc.GetRoom(ctx, roomID)

		require.NoError(t, err)
		assert.Equal(t, roomID, room.ID)
		assert.Equal(t, "Test Room", room.Name)
	})

	t.Run("not found", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()

		mockRepo.On("GetRoomByID", ctx, roomID).Return(nil, repository.ErrRoomNotFound).Once()

		room, err := svc.GetRoom(ctx, roomID)

		require.ErrorIs(t, err, service.ErrRoomNotFound)
		assert.Nil(t, room)
	})
}

func TestUpdateRoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mockRepo, pub := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		requesterID := uuid.New()

		mockRepo.On("GetRoomByID", ctx, roomID).
			Return(repoRoom(roomID, requesterID, "Old Name", repository.RoomTypeGroup), nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRoleOwner, nil).Once()
		mockRepo.On("UpdateRoom", ctx, roomID, "New Name").
			Return(repoRoom(roomID, requesterID, "New Name", repository.RoomTypeGroup), nil).Once()
		mockRepo.On("GetRoomMembers", ctx, roomID).Return([]repository.RoomMember{}, nil).Once()
		pub.On("PublishRoomUpdated", ctx, roomID, "New Name", []uuid.UUID{}).Return(nil).Once()

		result, err := svc.UpdateRoom(ctx, roomID, requesterID, "New Name")

		require.NoError(t, err)
		assert.Equal(t, "New Name", result.Name)
	})

	t.Run("not owner", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		requesterID := uuid.New()

		mockRepo.On("GetRoomByID", ctx, roomID).
			Return(repoRoom(roomID, requesterID, "Old Name", repository.RoomTypeGroup), nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRoleMember, nil).Once()

		_, err := svc.UpdateRoom(ctx, roomID, requesterID, "New Name")

		require.ErrorIs(t, err, service.ErrNotOwner)
	})

	t.Run("not member", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		requesterID := uuid.New()

		mockRepo.On("GetRoomByID", ctx, roomID).
			Return(repoRoom(roomID, requesterID, "Old Name", repository.RoomTypeGroup), nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).
			Return(repository.MemberRole(""), repository.ErrMemberNotFound).Once()

		_, err := svc.UpdateRoom(ctx, roomID, requesterID, "New Name")

		require.ErrorIs(t, err, service.ErrNotMember)
	})
}

func TestDeleteRoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		requesterID := uuid.New()

		mockRepo.On("GetRoomByID", ctx, roomID).
			Return(repoRoom(roomID, requesterID, "Test Room", repository.RoomTypeGroup), nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRoleOwner, nil).Once()
		mockRepo.On("DeleteRoom", ctx, roomID).Return(nil).Once()

		err := svc.DeleteRoom(ctx, roomID, requesterID)

		require.NoError(t, err)
	})

	t.Run("not owner", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		requesterID := uuid.New()

		mockRepo.On("GetRoomByID", ctx, roomID).
			Return(repoRoom(roomID, requesterID, "Test Room", repository.RoomTypeGroup), nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRoleMember, nil).Once()

		err := svc.DeleteRoom(ctx, roomID, requesterID)

		require.ErrorIs(t, err, service.ErrNotOwner)
	})

	t.Run("not member", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		requesterID := uuid.New()

		mockRepo.On("GetRoomByID", ctx, roomID).
			Return(repoRoom(roomID, requesterID, "Test Room", repository.RoomTypeGroup), nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).
			Return(repository.MemberRole(""), repository.ErrMemberNotFound).Once()

		err := svc.DeleteRoom(ctx, roomID, requesterID)

		require.ErrorIs(t, err, service.ErrNotMember)
	})
}

func TestAddMember(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mockRepo, pub := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()
		requesterID := uuid.New()

		members := []repository.RoomMember{
			{RoomID: roomID, UserID: requesterID, Role: repository.MemberRoleOwner, JoinedAt: time.Now()},
			{RoomID: roomID, UserID: userID, Role: repository.MemberRoleMember, JoinedAt: time.Now()},
		}

		mockRepo.On("GetRoomByID", ctx, roomID).
			Return(repoRoom(roomID, requesterID, "Test Room", repository.RoomTypeGroup), nil).Once()
		mockRepo.On("IsMember", ctx, roomID, requesterID).Return(true, nil).Once()
		mockRepo.On("AddMember", ctx, roomID, userID, repository.MemberRoleMember).
			Return(&repository.RoomMember{}, nil).Once()
		mockRepo.On("GetRoomMembers", ctx, roomID).Return(members, nil).Once()
		pub.On("PublishMembershipChanged", ctx, roomID, userID, "added", mock.Anything).Return(nil).Once()

		err := svc.AddMember(ctx, roomID, userID, requesterID)

		require.NoError(t, err)
	})
}

func TestRemoveMember(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mockRepo, pub := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()
		requesterID := uuid.New()

		members := []repository.RoomMember{
			{RoomID: roomID, UserID: requesterID, Role: repository.MemberRoleOwner, JoinedAt: time.Now()},
		}

		mockRepo.On("GetRoomByID", ctx, roomID).
			Return(repoRoom(roomID, requesterID, "Test Room", repository.RoomTypeGroup), nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRoleOwner, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, userID).Return(repository.MemberRoleMember, nil).Once()
		mockRepo.On("RemoveMember", ctx, roomID, userID).Return(nil).Once()
		mockRepo.On("GetRoomMembers", ctx, roomID).Return(members, nil).Once()
		pub.On("PublishMembershipChanged", ctx, roomID, userID, "removed", mock.Anything).Return(nil).Once()

		err := svc.RemoveMember(ctx, roomID, userID, requesterID)

		require.NoError(t, err)
	})

	t.Run("cannot remove owner", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		ownerID := uuid.New()
		requesterID := uuid.New()

		mockRepo.On("GetRoomByID", ctx, roomID).
			Return(repoRoom(roomID, requesterID, "Test Room", repository.RoomTypeGroup), nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRoleOwner, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, ownerID).Return(repository.MemberRoleOwner, nil).Once()

		err := svc.RemoveMember(ctx, roomID, ownerID, requesterID)

		require.ErrorIs(t, err, service.ErrCannotRemoveOwner)
	})

	t.Run("not owner", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()
		requesterID := uuid.New()

		mockRepo.On("GetRoomByID", ctx, roomID).
			Return(repoRoom(roomID, requesterID, "Test Room", repository.RoomTypeGroup), nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRoleMember, nil).Once()

		err := svc.RemoveMember(ctx, roomID, userID, requesterID)

		require.ErrorIs(t, err, service.ErrNotOwner)
	})

	t.Run("target not member", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()
		requesterID := uuid.New()

		mockRepo.On("GetRoomByID", ctx, roomID).
			Return(repoRoom(roomID, requesterID, "Test Room", repository.RoomTypeGroup), nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, requesterID).Return(repository.MemberRoleOwner, nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, userID).
			Return(repository.MemberRole(""), repository.ErrMemberNotFound).Once()

		err := svc.RemoveMember(ctx, roomID, userID, requesterID)

		require.ErrorIs(t, err, service.ErrMemberNotFound)
	})

	t.Run("member can remove themselves", func(t *testing.T) {
		svc, mockRepo, pub := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()

		members := []repository.RoomMember{
			{RoomID: roomID, UserID: uuid.New(), Role: repository.MemberRoleOwner, JoinedAt: time.Now()},
		}

		mockRepo.On("GetRoomByID", ctx, roomID).
			Return(repoRoom(roomID, uuid.New(), "Test Room", repository.RoomTypeGroup), nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, userID).Return(repository.MemberRoleMember, nil).Once()
		mockRepo.On("RemoveMember", ctx, roomID, userID).Return(nil).Once()
		mockRepo.On("GetRoomMembers", ctx, roomID).Return(members, nil).Once()
		pub.On("PublishMembershipChanged", ctx, roomID, userID, "removed", mock.Anything).Return(nil).Once()

		err := svc.RemoveMember(ctx, roomID, userID, userID)

		require.NoError(t, err)
	})

	t.Run("owner cannot remove themselves", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		ownerID := uuid.New()

		mockRepo.On("GetRoomByID", ctx, roomID).
			Return(repoRoom(roomID, ownerID, "Test Room", repository.RoomTypeGroup), nil).Once()
		mockRepo.On("GetMemberRole", ctx, roomID, ownerID).Return(repository.MemberRoleOwner, nil).Once()

		err := svc.RemoveMember(ctx, roomID, ownerID, ownerID)

		require.ErrorIs(t, err, service.ErrCannotRemoveOwner)
	})
}

func TestGetRoomMembers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
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
	})
}

func TestIsMember(t *testing.T) {
	t.Run("is member", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()

		mockRepo.On("IsMember", ctx, roomID, userID).Return(true, nil).Once()

		isMember, err := svc.IsMember(ctx, roomID, userID)

		require.NoError(t, err)
		assert.True(t, isMember)
	})

	t.Run("not member", func(t *testing.T) {
		svc, mockRepo, _ := newSvc(t)
		ctx := context.Background()
		roomID := uuid.New()
		userID := uuid.New()

		mockRepo.On("IsMember", ctx, roomID, userID).Return(false, nil).Once()

		isMember, err := svc.IsMember(ctx, roomID, userID)

		require.NoError(t, err)
		assert.False(t, isMember)
	})
}
