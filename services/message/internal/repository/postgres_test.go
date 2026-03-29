//go:build integration

package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudobytemebaby/efir/services/message/internal/repository"
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

func createTextMessage(t *testing.T, repo repository.MessageRepository, roomID, senderID uuid.UUID) *repository.Message {
	t.Helper()
	msg, err := repo.CreateMessage(context.Background(), &repository.CreateMessageInput{
		RoomID:   roomID,
		SenderID: senderID,
		Type:     repository.MessageTypeText,
		Content:  repository.TextContent{Text: "hello"},
	})
	require.NoError(t, err)
	return msg
}

func TestCreateMessage(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewMessageRepository(pool)
	ctx := context.Background()

	roomID := testutil.RandomUUID()
	senderID := testutil.RandomUUID()

	t.Run("text message", func(t *testing.T) {
		msg, err := repo.CreateMessage(ctx, &repository.CreateMessageInput{
			RoomID:   roomID,
			SenderID: senderID,
			Type:     repository.MessageTypeText,
			Content:  repository.TextContent{Text: "hello world"},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, msg.ID)
		assert.Equal(t, roomID, msg.RoomID)
		assert.Equal(t, senderID, msg.SenderID)
		assert.Equal(t, repository.MessageTypeText, msg.Type)
	})

	t.Run("media message", func(t *testing.T) {
		thumb := "thumb_01"
		msg, err := repo.CreateMessage(ctx, &repository.CreateMessageInput{
			RoomID:   roomID,
			SenderID: senderID,
			Type:     repository.MessageTypeImage,
			Content: repository.MediaContent{
				FileID:      "file_01",
				MimeType:    "image/jpeg",
				FileSize:    102400,
				Width:       800,
				Height:      600,
				ThumbnailID: &thumb,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, repository.MessageTypeImage, msg.Type)
	})

	t.Run("with reply_to_id", func(t *testing.T) {
		original := createTextMessage(t, repo, roomID, senderID)

		reply, err := repo.CreateMessage(ctx, &repository.CreateMessageInput{
			RoomID:    roomID,
			SenderID:  senderID,
			Type:      repository.MessageTypeText,
			Content:   repository.TextContent{Text: "reply"},
			ReplyToID: &original.ID,
		})
		require.NoError(t, err)
		require.NotNil(t, reply)
		assert.NotEmpty(t, reply.ID)
	})
}

func TestGetMessageByID(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewMessageRepository(pool)
	ctx := context.Background()

	roomID := testutil.RandomUUID()
	senderID := testutil.RandomUUID()
	created := createTextMessage(t, repo, roomID, senderID)

	t.Run("found", func(t *testing.T) {
		msg, err := repo.GetMessageByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, msg.ID)
		assert.Equal(t, roomID, msg.RoomID)
	})

	t.Run("not found returns ErrMessageNotFound", func(t *testing.T) {
		_, err := repo.GetMessageByID(ctx, testutil.RandomUUID())
		require.ErrorIs(t, err, repository.ErrMessageNotFound)
	})

	t.Run("soft-deleted message is still returned", func(t *testing.T) {
		msg := createTextMessage(t, repo, roomID, senderID)
		require.NoError(t, repo.SoftDeleteMessage(ctx, msg.ID))

		got, err := repo.GetMessageByID(ctx, msg.ID)
		require.NoError(t, err)
		assert.NotNil(t, got.DeletedAt, "deleted_at should be set")
	})
}

func TestGetMessagesByRoomID(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewMessageRepository(pool)
	ctx := context.Background()

	roomID := testutil.RandomUUID()
	senderID := testutil.RandomUUID()

	t.Run("empty room returns empty slice", func(t *testing.T) {
		msgs, cursor, err := repo.GetMessagesByRoomID(ctx, testutil.RandomUUID(), nil, 10)
		require.NoError(t, err)
		assert.Empty(t, msgs)
		assert.Nil(t, cursor)
	})

	t.Run("returns messages ordered by created_at desc", func(t *testing.T) {
		localRoom := testutil.RandomUUID()
		m1 := createTextMessage(t, repo, localRoom, senderID)
		time.Sleep(5 * time.Millisecond)
		m2 := createTextMessage(t, repo, localRoom, senderID)
		time.Sleep(5 * time.Millisecond)
		m3 := createTextMessage(t, repo, localRoom, senderID)

		msgs, _, err := repo.GetMessagesByRoomID(ctx, localRoom, nil, 10)
		require.NoError(t, err)
		require.Len(t, msgs, 3)
		// Newest first.
		assert.Equal(t, m3.ID, msgs[0].ID)
		assert.Equal(t, m2.ID, msgs[1].ID)
		assert.Equal(t, m1.ID, msgs[2].ID)
	})

	t.Run("deleted messages excluded", func(t *testing.T) {
		localRoom := testutil.RandomUUID()
		visible := createTextMessage(t, repo, localRoom, senderID)
		deleted := createTextMessage(t, repo, localRoom, senderID)
		require.NoError(t, repo.SoftDeleteMessage(ctx, deleted.ID))

		msgs, _, err := repo.GetMessagesByRoomID(ctx, localRoom, nil, 10)
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, visible.ID, msgs[0].ID)
	})

	_ = roomID // suppress unused warning — used indirectly via createTextMessage
}

func TestGetMessagesByRoomID_Pagination(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewMessageRepository(pool)
	ctx := context.Background()

	localRoom := testutil.RandomUUID()
	senderID := testutil.RandomUUID()

	// Insert 5 messages with slight delays to ensure distinct created_at.
	var ids []uuid.UUID
	for i := 0; i < 5; i++ {
		m := createTextMessage(t, repo, localRoom, senderID)
		ids = append(ids, m.ID)
		time.Sleep(3 * time.Millisecond)
	}
	// ids[0] = oldest, ids[4] = newest.

	t.Run("first page returns cursor", func(t *testing.T) {
		msgs, cursor, err := repo.GetMessagesByRoomID(ctx, localRoom, nil, 3)
		require.NoError(t, err)
		require.Len(t, msgs, 3)
		require.NotNil(t, cursor, "cursor should be set when more pages exist")
		// Newest first — ids[4], ids[3], ids[2].
		assert.Equal(t, ids[4], msgs[0].ID)
		assert.Equal(t, ids[3], msgs[1].ID)
		assert.Equal(t, ids[2], msgs[2].ID)
	})

	t.Run("second page returns remaining messages and nil cursor", func(t *testing.T) {
		// Get first page to find cursor.
		_, cursor, err := repo.GetMessagesByRoomID(ctx, localRoom, nil, 3)
		require.NoError(t, err)
		require.NotNil(t, cursor)

		msgs2, cursor2, err := repo.GetMessagesByRoomID(ctx, localRoom, cursor, 3)
		require.NoError(t, err)
		require.Len(t, msgs2, 2)
		assert.Nil(t, cursor2, "last page should have nil cursor")
		// ids[1], ids[0].
		assert.Equal(t, ids[1], msgs2[0].ID)
		assert.Equal(t, ids[0], msgs2[1].ID)
	})
}

func TestSoftDeleteMessage(t *testing.T) {
	pool := pgContainer.Pool(t)
	repo := repository.NewMessageRepository(pool)
	ctx := context.Background()

	roomID := testutil.RandomUUID()
	senderID := testutil.RandomUUID()

	t.Run("sets deleted_at and preserves content", func(t *testing.T) {
		msg := createTextMessage(t, repo, roomID, senderID)

		err := repo.SoftDeleteMessage(ctx, msg.ID)
		require.NoError(t, err)

		got, err := repo.GetMessageByID(ctx, msg.ID)
		require.NoError(t, err)
		assert.NotNil(t, got.DeletedAt)
		// Content should still be there.
		assert.Equal(t, repository.MessageTypeText, got.Type)
	})

	t.Run("not found returns ErrMessageNotFound", func(t *testing.T) {
		err := repo.SoftDeleteMessage(ctx, testutil.RandomUUID())
		require.ErrorIs(t, err, repository.ErrMessageNotFound)
	})
}
