package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sudobytemebaby/efir/services/message/internal/repository"
)

type mockRepository struct {
	CreateMessageFunc       func(ctx context.Context, input *repository.CreateMessageInput) (*repository.Message, error)
	GetMessagesByRoomIDFunc func(ctx context.Context, roomID uuid.UUID, cursor *uuid.UUID, limit int) ([]*repository.Message, *uuid.UUID, error)
	GetMessageByIDFunc      func(ctx context.Context, messageID uuid.UUID) (*repository.Message, error)
	SoftDeleteMessageFunc   func(ctx context.Context, messageID uuid.UUID) error
}

func (m *mockRepository) CreateMessage(ctx context.Context, input *repository.CreateMessageInput) (*repository.Message, error) {
	if m.CreateMessageFunc != nil {
		return m.CreateMessageFunc(ctx, input)
	}
	return nil, nil
}

func (m *mockRepository) GetMessagesByRoomID(ctx context.Context, roomID uuid.UUID, cursor *uuid.UUID, limit int) ([]*repository.Message, *uuid.UUID, error) {
	if m.GetMessagesByRoomIDFunc != nil {
		return m.GetMessagesByRoomIDFunc(ctx, roomID, cursor, limit)
	}
	return nil, nil, nil
}

func (m *mockRepository) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*repository.Message, error) {
	if m.GetMessageByIDFunc != nil {
		return m.GetMessageByIDFunc(ctx, messageID)
	}
	return nil, nil
}

func (m *mockRepository) SoftDeleteMessage(ctx context.Context, messageID uuid.UUID) error {
	if m.SoftDeleteMessageFunc != nil {
		return m.SoftDeleteMessageFunc(ctx, messageID)
	}
	return nil
}

type mockRoomClient struct {
	IsMemberFunc       func(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
	GetRoomMembersFunc func(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error)
}

func (m *mockRoomClient) IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	if m.IsMemberFunc != nil {
		return m.IsMemberFunc(ctx, roomID, userID)
	}
	return false, nil
}

func (m *mockRoomClient) GetRoomMembers(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error) {
	if m.GetRoomMembersFunc != nil {
		return m.GetRoomMembersFunc(ctx, roomID)
	}
	return nil, nil
}

type mockPublisher struct {
	PublishMessageCreatedFunc func(ctx context.Context, msg *repository.Message, recipientIDs []uuid.UUID) error
}

func (m *mockPublisher) PublishMessageCreated(ctx context.Context, msg *repository.Message, recipientIDs []uuid.UUID) error {
	if m.PublishMessageCreatedFunc != nil {
		return m.PublishMessageCreatedFunc(ctx, msg, recipientIDs)
	}
	return nil
}

func TestSendMessage_HappyPath(t *testing.T) {
	roomID := uuid.New()
	senderID := uuid.New()
	msgID := uuid.New()
	now := time.Now()

	repo := &mockRepository{
		CreateMessageFunc: func(ctx context.Context, input *repository.CreateMessageInput) (*repository.Message, error) {
			return &repository.Message{
				ID:        msgID,
				RoomID:    roomID,
				SenderID:  senderID,
				Type:      repository.MessageTypeText,
				Content:   repository.TextContent{Text: "Hello"},
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}

	roomClient := &mockRoomClient{
		IsMemberFunc: func(ctx context.Context, rid, uid uuid.UUID) (bool, error) {
			return true, nil
		},
		GetRoomMembersFunc: func(ctx context.Context, rid uuid.UUID) ([]uuid.UUID, error) {
			return []uuid.UUID{senderID}, nil
		},
	}

	publisher := &mockPublisher{
		PublishMessageCreatedFunc: func(ctx context.Context, msg *repository.Message, recipientIDs []uuid.UUID) error {
			return nil
		},
	}

	svc := NewMessageService(repo, roomClient, publisher)
	input := &SendMessageInput{
		RoomID:   roomID,
		SenderID: senderID,
		Type:     repository.MessageTypeText,
		Content:  repository.TextContent{Text: "Hello"},
	}

	msg, err := svc.SendMessage(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, msgID, msg.ID)
	assert.Equal(t, roomID, msg.RoomID)
}

func TestSendMessage_NotMember(t *testing.T) {
	roomID := uuid.New()
	senderID := uuid.New()

	roomClient := &mockRoomClient{
		IsMemberFunc: func(ctx context.Context, rid, uid uuid.UUID) (bool, error) {
			return false, nil
		},
	}

	svc := NewMessageService(&mockRepository{}, roomClient, &mockPublisher{})
	input := &SendMessageInput{
		RoomID:   roomID,
		SenderID: senderID,
		Type:     repository.MessageTypeText,
		Content:  repository.TextContent{Text: "Hello"},
	}

	_, err := svc.SendMessage(context.Background(), input)
	assert.ErrorIs(t, err, ErrNotMember)
}

func TestSendMessage_InvalidReplyTarget(t *testing.T) {
	roomID := uuid.New()
	senderID := uuid.New()
	replyToID := uuid.New()

	roomClient := &mockRoomClient{
		IsMemberFunc: func(ctx context.Context, rid, uid uuid.UUID) (bool, error) {
			return true, nil
		},
	}

	repo := &mockRepository{
		GetMessageByIDFunc: func(ctx context.Context, mid uuid.UUID) (*repository.Message, error) {
			return nil, repository.ErrMessageNotFound
		},
	}

	svc := NewMessageService(repo, roomClient, &mockPublisher{})
	input := &SendMessageInput{
		RoomID:    roomID,
		SenderID:  senderID,
		Type:      repository.MessageTypeText,
		Content:   repository.TextContent{Text: "Hello"},
		ReplyToID: &replyToID,
	}

	_, err := svc.SendMessage(context.Background(), input)
	assert.ErrorIs(t, err, ErrInvalidReplyTarget)
}

func TestSendMessage_ReplyFromDifferentRoom(t *testing.T) {
	roomID := uuid.New()
	senderID := uuid.New()
	replyToID := uuid.New()
	otherRoomID := uuid.New()

	roomClient := &mockRoomClient{
		IsMemberFunc: func(ctx context.Context, rid, uid uuid.UUID) (bool, error) {
			return true, nil
		},
	}

	repo := &mockRepository{
		GetMessageByIDFunc: func(ctx context.Context, mid uuid.UUID) (*repository.Message, error) {
			return &repository.Message{
				ID:     replyToID,
				RoomID: otherRoomID,
			}, nil
		},
	}

	svc := NewMessageService(repo, roomClient, &mockPublisher{})
	input := &SendMessageInput{
		RoomID:    roomID,
		SenderID:  senderID,
		Type:      repository.MessageTypeText,
		Content:   repository.TextContent{Text: "Hello"},
		ReplyToID: &replyToID,
	}

	_, err := svc.SendMessage(context.Background(), input)
	assert.ErrorIs(t, err, ErrInvalidReplyTarget)
}

func TestSendMessage_NATSFailure_NoError(t *testing.T) {
	roomID := uuid.New()
	senderID := uuid.New()
	msgID := uuid.New()
	now := time.Now()

	repo := &mockRepository{
		CreateMessageFunc: func(ctx context.Context, input *repository.CreateMessageInput) (*repository.Message, error) {
			return &repository.Message{
				ID:        msgID,
				RoomID:    roomID,
				SenderID:  senderID,
				Type:      repository.MessageTypeText,
				Content:   repository.TextContent{Text: "Hello"},
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}

	roomClient := &mockRoomClient{
		IsMemberFunc: func(ctx context.Context, rid, uid uuid.UUID) (bool, error) {
			return true, nil
		},
		GetRoomMembersFunc: func(ctx context.Context, rid uuid.UUID) ([]uuid.UUID, error) {
			return []uuid.UUID{senderID}, nil
		},
	}

	publisher := &mockPublisher{
		PublishMessageCreatedFunc: func(ctx context.Context, msg *repository.Message, recipientIDs []uuid.UUID) error {
			return errors.New("nats unavailable")
		},
	}

	svc := NewMessageService(repo, roomClient, publisher)
	input := &SendMessageInput{
		RoomID:   roomID,
		SenderID: senderID,
		Type:     repository.MessageTypeText,
		Content:  repository.TextContent{Text: "Hello"},
	}

	msg, err := svc.SendMessage(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, msgID, msg.ID)
}

func TestSendMessage_GetRoomMembersFailure_PublishSkipped(t *testing.T) {
	roomID := uuid.New()
	senderID := uuid.New()
	msgID := uuid.New()
	now := time.Now()
	publishCalled := false

	repo := &mockRepository{
		CreateMessageFunc: func(ctx context.Context, input *repository.CreateMessageInput) (*repository.Message, error) {
			return &repository.Message{
				ID:        msgID,
				RoomID:    roomID,
				SenderID:  senderID,
				Type:      repository.MessageTypeText,
				Content:   repository.TextContent{Text: "Hello"},
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}

	roomClient := &mockRoomClient{
		IsMemberFunc: func(ctx context.Context, rid, uid uuid.UUID) (bool, error) {
			return true, nil
		},
		GetRoomMembersFunc: func(ctx context.Context, rid uuid.UUID) ([]uuid.UUID, error) {
			return nil, errors.New("room service unavailable")
		},
	}

	publisher := &mockPublisher{
		PublishMessageCreatedFunc: func(ctx context.Context, msg *repository.Message, recipientIDs []uuid.UUID) error {
			publishCalled = true
			return nil
		},
	}

	svc := NewMessageService(repo, roomClient, publisher)
	input := &SendMessageInput{
		RoomID:   roomID,
		SenderID: senderID,
		Type:     repository.MessageTypeText,
		Content:  repository.TextContent{Text: "Hello"},
	}

	msg, err := svc.SendMessage(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, msgID, msg.ID)
	assert.False(t, publishCalled)
}

func TestGetMessages_NotMember(t *testing.T) {
	roomID := uuid.New()
	requesterID := uuid.New()

	roomClient := &mockRoomClient{
		IsMemberFunc: func(ctx context.Context, rid, uid uuid.UUID) (bool, error) {
			return false, nil
		},
	}

	svc := NewMessageService(&mockRepository{}, roomClient, &mockPublisher{})

	_, _, err := svc.GetMessages(context.Background(), roomID, requesterID, nil, 50)
	assert.ErrorIs(t, err, ErrNotMember)
}

func TestDeleteMessage_NotOwner(t *testing.T) {
	msgID := uuid.New()
	senderID := uuid.New()
	requesterID := uuid.New()

	repo := &mockRepository{
		GetMessageByIDFunc: func(ctx context.Context, mid uuid.UUID) (*repository.Message, error) {
			return &repository.Message{
				ID:       msgID,
				SenderID: senderID,
			}, nil
		},
	}

	svc := NewMessageService(repo, &mockRoomClient{}, &mockPublisher{})

	err := svc.DeleteMessage(context.Background(), msgID, requesterID)
	assert.ErrorIs(t, err, ErrNotOwner)
}

func TestDeleteMessage_NotFound(t *testing.T) {
	msgID := uuid.New()
	requesterID := uuid.New()

	repo := &mockRepository{
		GetMessageByIDFunc: func(ctx context.Context, mid uuid.UUID) (*repository.Message, error) {
			return nil, repository.ErrMessageNotFound
		},
	}

	svc := NewMessageService(repo, &mockRoomClient{}, &mockPublisher{})

	err := svc.DeleteMessage(context.Background(), msgID, requesterID)
	assert.ErrorIs(t, err, ErrMessageNotFound)
}
