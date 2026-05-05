package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sudobytemebaby/efir/services/message/internal/repository"
	repomocks "github.com/sudobytemebaby/efir/services/message/internal/repository/mocks"
	"github.com/sudobytemebaby/efir/services/message/internal/service"
	svcmocks "github.com/sudobytemebaby/efir/services/message/internal/service/mocks"
)

func newSvc(t *testing.T) (service.MessageService, *repomocks.MessageRepository, *svcmocks.RoomClient, *svcmocks.Publisher) {
	t.Helper()
	repo := repomocks.NewMessageRepository(t)
	roomClient := svcmocks.NewRoomClient(t)
	publisher := svcmocks.NewPublisher(t)
	svc := service.NewMessageService(repo, roomClient, publisher)
	return svc, repo, roomClient, publisher
}

func repoMsg(msgID, roomID, senderID uuid.UUID) *repository.Message {
	now := time.Now()
	return &repository.Message{
		ID:        msgID,
		RoomID:    roomID,
		SenderID:  senderID,
		Type:      repository.MessageTypeText,
		Content:   repository.TextContent{Text: "Hello"},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestSendMessage_HappyPath(t *testing.T) {
	t.Parallel()
	roomID, senderID, msgID := uuid.New(), uuid.New(), uuid.New()

	svc, repo, roomClient, publisher := newSvc(t)
	roomClient.On("IsMember", mock.Anything, roomID, senderID).Return(true, nil)
	repo.On("CreateMessage", mock.Anything, mock.MatchedBy(func(in *repository.CreateMessageInput) bool {
		return in.RoomID == roomID && in.SenderID == senderID
	})).Return(repoMsg(msgID, roomID, senderID), nil)
	roomClient.On("GetRoomMembers", mock.Anything, roomID).Return([]uuid.UUID{senderID}, nil)
	publisher.On("PublishMessageCreated", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	msg, err := svc.SendMessage(context.Background(), &service.SendMessageInput{
		RoomID:   roomID,
		SenderID: senderID,
		Type:     service.MessageTypeText,
		Content:  service.TextContent{Text: "Hello"},
	})
	require.NoError(t, err)
	assert.Equal(t, msgID, msg.ID)
	assert.Equal(t, roomID, msg.RoomID)
}

func TestSendMessage_NotMember(t *testing.T) {
	t.Parallel()
	roomID, senderID := uuid.New(), uuid.New()

	svc, _, roomClient, _ := newSvc(t)
	roomClient.On("IsMember", mock.Anything, roomID, senderID).Return(false, nil)

	_, err := svc.SendMessage(context.Background(), &service.SendMessageInput{
		RoomID:   roomID,
		SenderID: senderID,
		Type:     service.MessageTypeText,
		Content:  service.TextContent{Text: "Hello"},
	})
	assert.ErrorIs(t, err, service.ErrNotMember)
}

func TestSendMessage_InvalidReplyTarget_NotFound(t *testing.T) {
	t.Parallel()
	roomID, senderID, replyToID := uuid.New(), uuid.New(), uuid.New()

	svc, repo, roomClient, _ := newSvc(t)
	roomClient.On("IsMember", mock.Anything, roomID, senderID).Return(true, nil)
	repo.On("GetMessageByID", mock.Anything, replyToID).Return(nil, repository.ErrMessageNotFound)

	_, err := svc.SendMessage(context.Background(), &service.SendMessageInput{
		RoomID:    roomID,
		SenderID:  senderID,
		Type:      service.MessageTypeText,
		Content:   service.TextContent{Text: "Hello"},
		ReplyToID: &replyToID,
	})
	assert.ErrorIs(t, err, service.ErrInvalidReplyTarget)
}

func TestSendMessage_InvalidReplyTarget_DeletedMessage(t *testing.T) {
	t.Parallel()
	roomID, senderID, replyToID := uuid.New(), uuid.New(), uuid.New()
	deletedAt := time.Now()
	original := repoMsg(replyToID, roomID, senderID)
	original.DeletedAt = &deletedAt

	svc, repo, roomClient, _ := newSvc(t)
	roomClient.On("IsMember", mock.Anything, roomID, senderID).Return(true, nil)
	repo.On("GetMessageByID", mock.Anything, replyToID).Return(original, nil)

	_, err := svc.SendMessage(context.Background(), &service.SendMessageInput{
		RoomID:    roomID,
		SenderID:  senderID,
		Type:      service.MessageTypeText,
		Content:   service.TextContent{Text: "reply"},
		ReplyToID: &replyToID,
	})
	assert.ErrorIs(t, err, service.ErrInvalidReplyTarget)
}

func TestSendMessage_InvalidReplyTarget_DifferentRoom(t *testing.T) {
	t.Parallel()
	roomID, senderID, replyToID, otherRoom := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	original := repoMsg(replyToID, otherRoom, senderID)

	svc, repo, roomClient, _ := newSvc(t)
	roomClient.On("IsMember", mock.Anything, roomID, senderID).Return(true, nil)
	repo.On("GetMessageByID", mock.Anything, replyToID).Return(original, nil)

	_, err := svc.SendMessage(context.Background(), &service.SendMessageInput{
		RoomID:    roomID,
		SenderID:  senderID,
		Type:      service.MessageTypeText,
		Content:   service.TextContent{Text: "Hello"},
		ReplyToID: &replyToID,
	})
	assert.ErrorIs(t, err, service.ErrInvalidReplyTarget)
}

func TestSendMessage_NATSFailure_NoError(t *testing.T) {
	t.Parallel()
	roomID, senderID, msgID := uuid.New(), uuid.New(), uuid.New()

	svc, repo, roomClient, publisher := newSvc(t)
	roomClient.On("IsMember", mock.Anything, roomID, senderID).Return(true, nil)
	repo.On("CreateMessage", mock.Anything, mock.Anything).Return(repoMsg(msgID, roomID, senderID), nil)
	roomClient.On("GetRoomMembers", mock.Anything, roomID).Return([]uuid.UUID{senderID}, nil)
	publisher.On("PublishMessageCreated", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("nats unavailable"))

	msg, err := svc.SendMessage(context.Background(), &service.SendMessageInput{
		RoomID:   roomID,
		SenderID: senderID,
		Type:     service.MessageTypeText,
		Content:  service.TextContent{Text: "Hello"},
	})
	require.NoError(t, err)
	assert.Equal(t, msgID, msg.ID)
}

func TestSendMessage_GetRoomMembersFailure_PublishSkipped(t *testing.T) {
	t.Parallel()
	roomID, senderID, msgID := uuid.New(), uuid.New(), uuid.New()

	svc, repo, roomClient, _ := newSvc(t)
	roomClient.On("IsMember", mock.Anything, roomID, senderID).Return(true, nil)
	repo.On("CreateMessage", mock.Anything, mock.Anything).Return(repoMsg(msgID, roomID, senderID), nil)
	roomClient.On("GetRoomMembers", mock.Anything, roomID).Return(nil, errors.New("room service unavailable"))
	// Publisher must NOT be called — mockery will fail the test if an unexpected call occurs.

	msg, err := svc.SendMessage(context.Background(), &service.SendMessageInput{
		RoomID:   roomID,
		SenderID: senderID,
		Type:     service.MessageTypeText,
		Content:  service.TextContent{Text: "Hello"},
	})
	require.NoError(t, err)
	assert.Equal(t, msgID, msg.ID)
}

func TestSendMessage_MediaType(t *testing.T) {
	t.Parallel()
	roomID, senderID, msgID := uuid.New(), uuid.New(), uuid.New()
	thumb := "thumb_01"
	now := time.Now()
	mediaMsg := &repository.Message{
		ID:        msgID,
		RoomID:    roomID,
		SenderID:  senderID,
		Type:      repository.MessageTypeImage,
		Content:   repository.MediaContent{FileID: "file_01", MimeType: "image/jpeg", FileSize: 102400, Width: 800, Height: 600, ThumbnailID: &thumb},
		CreatedAt: now,
		UpdatedAt: now,
	}

	svc, repo, roomClient, publisher := newSvc(t)
	roomClient.On("IsMember", mock.Anything, roomID, senderID).Return(true, nil)
	repo.On("CreateMessage", mock.Anything, mock.Anything).Return(mediaMsg, nil)
	roomClient.On("GetRoomMembers", mock.Anything, roomID).Return([]uuid.UUID{senderID}, nil)
	publisher.On("PublishMessageCreated", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	msg, err := svc.SendMessage(context.Background(), &service.SendMessageInput{
		RoomID:   roomID,
		SenderID: senderID,
		Type:     service.MessageTypeImage,
		Content:  service.MediaContent{FileID: "file_01", MimeType: "image/jpeg", FileSize: 102400, Width: 800, Height: 600},
	})
	require.NoError(t, err)
	assert.Equal(t, service.MessageTypeImage, msg.Type)
}

func TestGetMessages_Success(t *testing.T) {
	t.Parallel()
	roomID, requesterID := uuid.New(), uuid.New()
	msgs := []*repository.Message{repoMsg(uuid.New(), roomID, requesterID)}

	svc, repo, roomClient, _ := newSvc(t)
	roomClient.On("IsMember", mock.Anything, roomID, requesterID).Return(true, nil)
	repo.On("GetMessagesByRoomID", mock.Anything, roomID, (*uuid.UUID)(nil), 50).
		Return(msgs, (*uuid.UUID)(nil), nil)

	got, cursor, err := svc.GetMessages(context.Background(), roomID, requesterID, nil, 50)
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Nil(t, cursor)
}

func TestGetMessages_NotMember(t *testing.T) {
	t.Parallel()
	roomID, requesterID := uuid.New(), uuid.New()

	svc, _, roomClient, _ := newSvc(t)
	roomClient.On("IsMember", mock.Anything, roomID, requesterID).Return(false, nil)

	_, _, err := svc.GetMessages(context.Background(), roomID, requesterID, nil, 50)
	assert.ErrorIs(t, err, service.ErrNotMember)
}

func TestGetMessageByID_Success(t *testing.T) {
	t.Parallel()
	msgID, roomID, requesterID := uuid.New(), uuid.New(), uuid.New()

	svc, repo, roomClient, _ := newSvc(t)
	repo.On("GetMessageByID", mock.Anything, msgID).Return(repoMsg(msgID, roomID, requesterID), nil)
	roomClient.On("IsMember", mock.Anything, roomID, requesterID).Return(true, nil)

	msg, err := svc.GetMessageByID(context.Background(), msgID, requesterID)
	require.NoError(t, err)
	assert.Equal(t, msgID, msg.ID)
}

func TestGetMessageByID_NotFound(t *testing.T) {
	t.Parallel()
	msgID, requesterID := uuid.New(), uuid.New()

	svc, repo, _, _ := newSvc(t)
	repo.On("GetMessageByID", mock.Anything, msgID).Return(nil, repository.ErrMessageNotFound)

	_, err := svc.GetMessageByID(context.Background(), msgID, requesterID)
	assert.ErrorIs(t, err, service.ErrMessageNotFound)
}

func TestGetMessageByID_NotMember(t *testing.T) {
	t.Parallel()
	msgID, roomID, senderID, requesterID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	svc, repo, roomClient, _ := newSvc(t)
	repo.On("GetMessageByID", mock.Anything, msgID).Return(repoMsg(msgID, roomID, senderID), nil)
	roomClient.On("IsMember", mock.Anything, roomID, requesterID).Return(false, nil)

	_, err := svc.GetMessageByID(context.Background(), msgID, requesterID)
	assert.ErrorIs(t, err, service.ErrNotMember)
}

func TestDeleteMessage_NotOwner(t *testing.T) {
	t.Parallel()
	msgID, roomID, senderID, requesterID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	svc, repo, roomClient, _ := newSvc(t)
	repo.On("GetMessageByID", mock.Anything, msgID).Return(repoMsg(msgID, roomID, senderID), nil)
	roomClient.On("IsMember", mock.Anything, roomID, requesterID).Return(true, nil)

	err := svc.DeleteMessage(context.Background(), msgID, requesterID)
	assert.ErrorIs(t, err, service.ErrNotOwner)
}

func TestDeleteMessage_NotFound(t *testing.T) {
	t.Parallel()
	msgID, requesterID := uuid.New(), uuid.New()

	svc, repo, _, _ := newSvc(t)
	repo.On("GetMessageByID", mock.Anything, msgID).Return(nil, repository.ErrMessageNotFound)

	err := svc.DeleteMessage(context.Background(), msgID, requesterID)
	assert.ErrorIs(t, err, service.ErrMessageNotFound)
}

func TestDeleteMessage_AfterLeaving(t *testing.T) {
	t.Parallel()
	// A user who left the room loses write access — delete is rejected even for their own messages.
	msgID, senderID, roomID := uuid.New(), uuid.New(), uuid.New()

	svc, repo, roomClient, _ := newSvc(t)
	repo.On("GetMessageByID", mock.Anything, msgID).Return(repoMsg(msgID, roomID, senderID), nil)
	roomClient.On("IsMember", mock.Anything, roomID, senderID).Return(false, nil)

	err := svc.DeleteMessage(context.Background(), msgID, senderID)
	assert.ErrorIs(t, err, service.ErrNotMember)
}

func TestDeleteMessage_Success(t *testing.T) {
	t.Parallel()
	msgID, senderID, roomID := uuid.New(), uuid.New(), uuid.New()

	svc, repo, roomClient, _ := newSvc(t)
	repo.On("GetMessageByID", mock.Anything, msgID).Return(repoMsg(msgID, roomID, senderID), nil)
	roomClient.On("IsMember", mock.Anything, roomID, senderID).Return(true, nil)
	repo.On("SoftDeleteMessage", mock.Anything, msgID).Return(nil)

	err := svc.DeleteMessage(context.Background(), msgID, senderID)
	require.NoError(t, err)
}
