package handler

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sudobytemebaby/efir/services/message/internal/repository"
	"github.com/sudobytemebaby/efir/services/message/internal/service"
	messagev1 "github.com/sudobytemebaby/efir/services/shared/gen/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockMessageService struct {
	SendMessageFunc    func(ctx context.Context, input *service.SendMessageInput) (*repository.Message, error)
	GetMessagesFunc    func(ctx context.Context, roomID, requesterID uuid.UUID, cursor *uuid.UUID, limit int) ([]*repository.Message, *uuid.UUID, error)
	GetMessageByIDFunc func(ctx context.Context, messageID, requesterID uuid.UUID) (*repository.Message, error)
	DeleteMessageFunc  func(ctx context.Context, messageID, requesterID uuid.UUID) error
}

func (m *mockMessageService) SendMessage(ctx context.Context, input *service.SendMessageInput) (*repository.Message, error) {
	if m.SendMessageFunc != nil {
		return m.SendMessageFunc(ctx, input)
	}
	return nil, nil
}

func (m *mockMessageService) GetMessages(ctx context.Context, roomID, requesterID uuid.UUID, cursor *uuid.UUID, limit int) ([]*repository.Message, *uuid.UUID, error) {
	if m.GetMessagesFunc != nil {
		return m.GetMessagesFunc(ctx, roomID, requesterID, cursor, limit)
	}
	return nil, nil, nil
}

func (m *mockMessageService) GetMessageByID(ctx context.Context, messageID, requesterID uuid.UUID) (*repository.Message, error) {
	if m.GetMessageByIDFunc != nil {
		return m.GetMessageByIDFunc(ctx, messageID, requesterID)
	}
	return nil, nil
}

func (m *mockMessageService) DeleteMessage(ctx context.Context, messageID, requesterID uuid.UUID) error {
	if m.DeleteMessageFunc != nil {
		return m.DeleteMessageFunc(ctx, messageID, requesterID)
	}
	return nil
}

func TestSendMessage_Validation(t *testing.T) {
	h, err := NewMessageHandler(&mockMessageService{})
	require.NoError(t, err)

	_, err = h.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   "",
		SenderId: "user-123",
	})
	assert.Error(t, err)

	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}

func TestSendMessage_EmptyContent(t *testing.T) {
	h, err := NewMessageHandler(&mockMessageService{})
	require.NoError(t, err)

	_, err = h.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   uuid.New().String(),
		SenderId: uuid.New().String(),
	})
	assert.Error(t, err)

	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}

func TestSendMessage_ErrorMapping(t *testing.T) {
	svc := &mockMessageService{
		SendMessageFunc: func(ctx context.Context, input *service.SendMessageInput) (*repository.Message, error) {
			return nil, service.ErrNotMember
		},
	}

	h, err := NewMessageHandler(svc)
	require.NoError(t, err)

	_, err = h.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:   uuid.New().String(),
		SenderId: uuid.New().String(),
		Type:     messagev1.MessageType_MESSAGE_TYPE_TEXT,
		Content: &messagev1.SendMessageRequest_Text{
			Text: &messagev1.SendTextContent{Text: "Hello"},
		},
	})
	assert.Error(t, err)

	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, s.Code())
}

func TestSendMessage_InvalidReplyTargetMapping(t *testing.T) {
	svc := &mockMessageService{
		SendMessageFunc: func(ctx context.Context, input *service.SendMessageInput) (*repository.Message, error) {
			return nil, service.ErrInvalidReplyTarget
		},
	}

	h, err := NewMessageHandler(svc)
	require.NoError(t, err)

	replyToID := uuid.New().String()
	_, err = h.SendMessage(context.Background(), &messagev1.SendMessageRequest{
		RoomId:    uuid.New().String(),
		SenderId:  uuid.New().String(),
		Type:      messagev1.MessageType_MESSAGE_TYPE_TEXT,
		ReplyToId: &replyToID,
		Content: &messagev1.SendMessageRequest_Text{
			Text: &messagev1.SendTextContent{Text: "Hello"},
		},
	})
	assert.Error(t, err)

	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}

func TestGetMessages_Validation(t *testing.T) {
	h, err := NewMessageHandler(&mockMessageService{})
	require.NoError(t, err)

	_, err = h.GetMessages(context.Background(), &messagev1.GetMessagesRequest{
		RoomId:      uuid.New().String(),
		RequesterId: uuid.New().String(),
		Limit:       0,
	})
	assert.Error(t, err)

	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}

func TestDeleteMessage_ErrorMapping_NotOwner(t *testing.T) {
	svc := &mockMessageService{
		DeleteMessageFunc: func(ctx context.Context, messageID, requesterID uuid.UUID) error {
			return service.ErrNotOwner
		},
	}

	h, err := NewMessageHandler(svc)
	require.NoError(t, err)

	_, err = h.DeleteMessage(context.Background(), &messagev1.DeleteMessageRequest{
		MessageId:   uuid.New().String(),
		RequesterId: uuid.New().String(),
	})
	assert.Error(t, err)

	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, s.Code())
}

func TestDeleteMessage_ErrorMapping_NotFound(t *testing.T) {
	svc := &mockMessageService{
		DeleteMessageFunc: func(ctx context.Context, messageID, requesterID uuid.UUID) error {
			return service.ErrMessageNotFound
		},
	}

	h, err := NewMessageHandler(svc)
	require.NoError(t, err)

	_, err = h.DeleteMessage(context.Background(), &messagev1.DeleteMessageRequest{
		MessageId:   uuid.New().String(),
		RequesterId: uuid.New().String(),
	})
	assert.Error(t, err)

	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, s.Code())
}

func TestMapMessageToProto_DeletedMessage(t *testing.T) {
	now := time.Now()
	msg := &repository.Message{
		ID:        uuid.New(),
		RoomID:    uuid.New(),
		SenderID:  uuid.New(),
		Type:      repository.MessageTypeText,
		DeletedAt: &now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	result := mapMessageToProto(msg)
	assert.True(t, result.IsDeleted)
}

func TestMapMessageToProto_WithReplyTo(t *testing.T) {
	now := time.Now()
	replyToID := uuid.New()
	preview := &repository.MessagePreview{
		MessageID:   replyToID,
		SenderID:    uuid.New(),
		Type:        repository.MessageTypeText,
		TextPreview: strPtr("Original message"),
	}

	msg := &repository.Message{
		ID:        uuid.New(),
		RoomID:    uuid.New(),
		SenderID:  uuid.New(),
		Type:      repository.MessageTypeText,
		Content:   repository.TextContent{Text: "Reply"},
		ReplyTo:   preview,
		CreatedAt: now,
		UpdatedAt: now,
	}

	result := mapMessageToProto(msg)
	assert.NotNil(t, result.ReplyTo)
	assert.Equal(t, replyToID.String(), result.ReplyTo.MessageId)
	assert.NotNil(t, result.ReplyTo.TextPreview)
	assert.Equal(t, "Original message", *result.ReplyTo.TextPreview)
}

func TestMapPreviewToProto(t *testing.T) {
	preview := &repository.MessagePreview{
		MessageID:   uuid.New(),
		SenderID:    uuid.New(),
		Type:        repository.MessageTypeText,
		TextPreview: strPtr("Preview text"),
		FileName:    strPtr("file.pdf"),
		MimeType:    strPtr("application/pdf"),
	}

	result := mapPreviewToProto(preview)
	assert.Equal(t, preview.MessageID.String(), result.MessageId)
	assert.Equal(t, preview.SenderID.String(), result.SenderId)
	assert.NotNil(t, result.TextPreview)
	assert.Equal(t, "Preview text", *result.TextPreview)
}

func strPtr(s string) *string {
	return &s
}
