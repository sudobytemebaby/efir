//go:build integration

package repository_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudobytemebaby/efir/services/room/internal/repository"
	"github.com/sudobytemebaby/efir/services/shared/pkg/testutil"
)

var pgContainer *testutil.PostgresContainer

func TestMain(m *testing.M) {
	ctx := context.Background()
	pgContainer = testutil.NewPostgresContainer(ctx, "../../migrations")
	exitCode := m.Run()
	_ = pgContainer.Terminate(ctx)
	os.Exit(exitCode)
}

// createRoom creates a group room (owner is added automatically by CreateRoom).
func createRoom(t *testing.T, repo repository.RoomRepository, createdBy uuid.UUID) *repository.Room {
	t.Helper()
	room, err := repo.CreateRoom(context.Background(), testutil.RandomRoomName(), repository.RoomTypeGroup, createdBy, uuid.Nil)
	require.NoError(t, err)
	return room
}

func TestCreateRoom(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewRoomRepository(pool)
	ctx := context.Background()

	ownerID := testutil.RandomUUID()

	tests := []struct {
		name     string
		roomType repository.RoomType
	}{
		{name: "success group room", roomType: repository.RoomTypeGroup},
		{name: "success direct room", roomType: repository.RoomTypeDirect},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			participantID := uuid.Nil
			if tc.roomType == repository.RoomTypeDirect {
				participantID = testutil.RandomUUID()
			}
			room, err := repo.CreateRoom(ctx, testutil.RandomRoomName(), tc.roomType, ownerID, participantID)
			require.NoError(t, err)
			require.NotNil(t, room)
			assert.NotEmpty(t, room.ID)
			assert.Equal(t, tc.roomType, room.Type)
			assert.Equal(t, ownerID, room.CreatedBy)
		})
	}
}

func TestGetRoomByID(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewRoomRepository(pool)
	ctx := context.Background()

	ownerID := testutil.RandomUUID()
	created := createRoom(t, repo, ownerID)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{name: "found", id: created.ID},
		{name: "not found", id: testutil.RandomUUID(), wantErr: repository.ErrRoomNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			room, err := repo.GetRoomByID(ctx, tc.id)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, room)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, created.ID, room.ID)
			assert.Equal(t, created.Name, room.Name)
		})
	}
}

func TestUpdateRoom(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewRoomRepository(pool)
	ctx := context.Background()

	ownerID := testutil.RandomUUID()
	created := createRoom(t, repo, ownerID)

	tests := []struct {
		name    string
		id      uuid.UUID
		newName string
		wantErr error
	}{
		{name: "success", id: created.ID, newName: "Updated Room"},
		{name: "not found", id: testutil.RandomUUID(), newName: "x", wantErr: repository.ErrRoomNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			room, err := repo.UpdateRoom(ctx, tc.id, tc.newName)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, room)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.newName, room.Name)
		})
	}
}

func TestDeleteRoom(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewRoomRepository(pool)
	ctx := context.Background()

	ownerID := testutil.RandomUUID()

	t.Run("success", func(t *testing.T) {
		room := createRoom(t, repo, ownerID)
		err := repo.DeleteRoom(ctx, room.ID)
		require.NoError(t, err)

		_, err = repo.GetRoomByID(ctx, room.ID)
		require.ErrorIs(t, err, repository.ErrRoomNotFound)
	})

	t.Run("not found returns error", func(t *testing.T) {
		err := repo.DeleteRoom(ctx, testutil.RandomUUID())
		require.ErrorIs(t, err, repository.ErrRoomNotFound)
	})

	t.Run("cascades to members", func(t *testing.T) {
		room := createRoom(t, repo, ownerID)
		memberID := testutil.RandomUUID()
		_, err := repo.AddMember(ctx, room.ID, memberID, repository.MemberRoleMember)
		require.NoError(t, err)

		require.NoError(t, repo.DeleteRoom(ctx, room.ID))

		// Members should also be gone (via ON DELETE CASCADE).
		isMember, err := repo.IsMember(ctx, room.ID, memberID)
		require.NoError(t, err)
		assert.False(t, isMember)
	})
}

func TestAddMember(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewRoomRepository(pool)
	ctx := context.Background()

	ownerID := testutil.RandomUUID()
	room := createRoom(t, repo, ownerID)
	memberID := testutil.RandomUUID()

	t.Run("success", func(t *testing.T) {
		m, err := repo.AddMember(ctx, room.ID, memberID, repository.MemberRoleMember)
		require.NoError(t, err)
		assert.Equal(t, room.ID, m.RoomID)
		assert.Equal(t, memberID, m.UserID)
		assert.Equal(t, repository.MemberRoleMember, m.Role)
	})

	t.Run("duplicate returns ErrMemberAlreadyExists", func(t *testing.T) {
		_, err := repo.AddMember(ctx, room.ID, memberID, repository.MemberRoleMember)
		require.ErrorIs(t, err, repository.ErrMemberAlreadyExists)
	})
}

func TestRemoveMember(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewRoomRepository(pool)
	ctx := context.Background()

	ownerID := testutil.RandomUUID()
	room := createRoom(t, repo, ownerID)

	t.Run("success", func(t *testing.T) {
		memberID := testutil.RandomUUID()
		_, err := repo.AddMember(ctx, room.ID, memberID, repository.MemberRoleMember)
		require.NoError(t, err)

		err = repo.RemoveMember(ctx, room.ID, memberID)
		require.NoError(t, err)

		isMember, err := repo.IsMember(ctx, room.ID, memberID)
		require.NoError(t, err)
		assert.False(t, isMember)
	})

	t.Run("not found returns error", func(t *testing.T) {
		err := repo.RemoveMember(ctx, room.ID, testutil.RandomUUID())
		require.ErrorIs(t, err, repository.ErrMemberNotFound)
	})
}

func TestGetRoomMembers(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewRoomRepository(pool)
	ctx := context.Background()

	ownerID := testutil.RandomUUID()

	t.Run("returns all members", func(t *testing.T) {
		room := createRoom(t, repo, ownerID)
		m1 := testutil.RandomUUID()
		m2 := testutil.RandomUUID()
		_, err := repo.AddMember(ctx, room.ID, m1, repository.MemberRoleMember)
		require.NoError(t, err)
		_, err = repo.AddMember(ctx, room.ID, m2, repository.MemberRoleMember)
		require.NoError(t, err)

		members, err := repo.GetRoomMembers(ctx, room.ID)
		require.NoError(t, err)
		// ownerID was added in createRoom, plus m1 and m2.
		assert.Len(t, members, 3)
	})

	t.Run("room has owner member", func(t *testing.T) {
		room, err := repo.CreateRoom(ctx, testutil.RandomRoomName(), repository.RoomTypeGroup, ownerID, uuid.Nil)
		require.NoError(t, err)

		members, err := repo.GetRoomMembers(ctx, room.ID)
		require.NoError(t, err)
		assert.Len(t, members, 1)
	})
}

func TestIsMember(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewRoomRepository(pool)
	ctx := context.Background()

	ownerID := testutil.RandomUUID()
	room := createRoom(t, repo, ownerID)
	memberID := testutil.RandomUUID()
	_, err := repo.AddMember(ctx, room.ID, memberID, repository.MemberRoleMember)
	require.NoError(t, err)

	tests := []struct {
		name   string
		userID uuid.UUID
		want   bool
	}{
		{name: "is member", userID: memberID, want: true},
		{name: "is not member", userID: testutil.RandomUUID(), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ok, err := repo.IsMember(ctx, room.ID, tc.userID)
			require.NoError(t, err)
			assert.Equal(t, tc.want, ok)
		})
	}
}

func TestGetMemberRole(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewRoomRepository(pool)
	ctx := context.Background()

	ownerID := testutil.RandomUUID()
	room := createRoom(t, repo, ownerID)
	memberID := testutil.RandomUUID()
	_, err := repo.AddMember(ctx, room.ID, memberID, repository.MemberRoleMember)
	require.NoError(t, err)

	tests := []struct {
		name     string
		userID   uuid.UUID
		wantRole repository.MemberRole
		wantErr  error
	}{
		{name: "owner", userID: ownerID, wantRole: repository.MemberRoleOwner},
		{name: "member", userID: memberID, wantRole: repository.MemberRoleMember},
		{name: "not found", userID: testutil.RandomUUID(), wantErr: repository.ErrMemberNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			role, err := repo.GetMemberRole(ctx, room.ID, tc.userID)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantRole, role)
		})
	}
}

func TestGetDirectRoomByUsers(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewRoomRepository(pool)
	ctx := context.Background()

	userA := testutil.RandomUUID()
	userB := testutil.RandomUUID()

	// Create a direct room between A and B.
	direct, err := repo.CreateRoom(ctx, "direct", repository.RoomTypeDirect, userA, userB)
	require.NoError(t, err)

	// Create a group room that both A and B belong to — should NOT be returned.
	group := createRoom(t, repo, userA)
	_, err = repo.AddMember(ctx, group.ID, userB, repository.MemberRoleMember)
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		room, err := repo.GetDirectRoomByUsers(ctx, userA, userB)
		require.NoError(t, err)
		assert.Equal(t, direct.ID, room.ID)
		assert.Equal(t, repository.RoomTypeDirect, room.Type)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetDirectRoomByUsers(ctx, testutil.RandomUUID(), testutil.RandomUUID())
		require.ErrorIs(t, err, repository.ErrRoomNotFound)
	})

	t.Run("only matches direct type not group", func(t *testing.T) {
		// A third user joins neither direct room.
		userC := testutil.RandomUUID()
		_, err := repo.GetDirectRoomByUsers(ctx, userA, userC)
		require.ErrorIs(t, err, repository.ErrRoomNotFound)
	})
}
