package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/message/internal/nats"
	"github.com/sudobytemebaby/efir/services/message/internal/repository"
)

var (
	ErrMessageNotFound    = errors.New("message not found")
	ErrNotMember          = errors.New("must be a room member")
	ErrNotOwner           = errors.New("only sender can delete message")
	ErrInvalidReplyTarget = errors.New("reply target not found or belongs to a different room")
)

type SendMessageInput struct {
	RoomID    uuid.UUID
	SenderID  uuid.UUID
	Type      repository.MessageType
	Content   repository.MessageContent
	ReplyToID *uuid.UUID
}

type MessageService interface {
	SendMessage(ctx context.Context, input *SendMessageInput) (*repository.Message, error)
	GetMessages(ctx context.Context, roomID, requesterID uuid.UUID, cursor *uuid.UUID, limit int) ([]*repository.Message, *uuid.UUID, error)
	GetMessageByID(ctx context.Context, messageID, requesterID uuid.UUID) (*repository.Message, error)
	DeleteMessage(ctx context.Context, messageID, requesterID uuid.UUID) error
}

type RoomClient interface {
	IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
	GetRoomMembers(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error)
}

type messageService struct {
	repo       repository.MessageRepository
	roomClient RoomClient
	publisher  nats.Publisher
}

func NewMessageService(repo repository.MessageRepository, roomClient RoomClient, publisher nats.Publisher) MessageService {
	return &messageService{
		repo:       repo,
		roomClient: roomClient,
		publisher:  publisher,
	}
}

func (s *messageService) SendMessage(ctx context.Context, input *SendMessageInput) (*repository.Message, error) {
	isMember, err := s.roomClient.IsMember(ctx, input.RoomID, input.SenderID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotMember
	}

	if input.ReplyToID != nil {
		original, err := s.repo.GetMessageByID(ctx, *input.ReplyToID)
		if err != nil {
			return nil, ErrInvalidReplyTarget
		}
		if original.DeletedAt != nil {
			return nil, ErrInvalidReplyTarget
		}
		if original.RoomID != input.RoomID {
			return nil, ErrInvalidReplyTarget
		}
	}

	msg, err := s.repo.CreateMessage(ctx, &repository.CreateMessageInput{
		RoomID:    input.RoomID,
		SenderID:  input.SenderID,
		Type:      input.Type,
		Content:   input.Content,
		ReplyToID: input.ReplyToID,
	})
	if err != nil {
		return nil, err
	}

	recipientIDs, err := s.roomClient.GetRoomMembers(ctx, input.RoomID)
	if err != nil {
		slog.Error("failed to get room members for publish",
			"error", err,
			"room_id", input.RoomID.String(),
		)
	} else {
		if err := s.publisher.PublishMessageCreated(ctx, msg, recipientIDs); err != nil {
			slog.Error("failed to publish message created event, event may be lost",
				"error", err,
				"message_id", msg.ID.String(),
				"room_id", msg.RoomID.String(),
			)
		}
	}

	return msg, nil
}

func (s *messageService) GetMessages(ctx context.Context, roomID, requesterID uuid.UUID, cursor *uuid.UUID, limit int) ([]*repository.Message, *uuid.UUID, error) {
	isMember, err := s.roomClient.IsMember(ctx, roomID, requesterID)
	if err != nil {
		return nil, nil, err
	}
	if !isMember {
		return nil, nil, ErrNotMember
	}

	return s.repo.GetMessagesByRoomID(ctx, roomID, cursor, limit)
}

func (s *messageService) GetMessageByID(ctx context.Context, messageID, requesterID uuid.UUID) (*repository.Message, error) {
	msg, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		if errors.Is(err, repository.ErrMessageNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, fmt.Errorf("get message: %w", err)
	}

	isMember, err := s.roomClient.IsMember(ctx, msg.RoomID, requesterID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotMember
	}

	return msg, nil
}

func (s *messageService) DeleteMessage(ctx context.Context, messageID, requesterID uuid.UUID) error {
	msg, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		if errors.Is(err, repository.ErrMessageNotFound) {
			return ErrMessageNotFound
		}
		return fmt.Errorf("get message: %w", err)
	}

	if msg.SenderID != requesterID {
		return ErrNotOwner
	}

	return s.repo.SoftDeleteMessage(ctx, messageID)
}
