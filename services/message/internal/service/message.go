package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/message/internal/repository"
)

var (
	ErrMessageNotFound    = errors.New("message not found")
	ErrNotMember          = errors.New("must be a room member")
	ErrNotOwner           = errors.New("only sender can delete message")
	ErrInvalidReplyTarget = errors.New("reply target not found or belongs to a different room")
)

//go:generate mockery --name Publisher
type Publisher interface {
	PublishMessageCreated(ctx context.Context, msg *Message, recipientIDs []uuid.UUID) error
}

//go:generate mockery --name MessageService
type MessageService interface {
	SendMessage(ctx context.Context, input *SendMessageInput) (*Message, error)
	GetMessages(ctx context.Context, roomID, requesterID uuid.UUID, cursor *uuid.UUID, limit int) ([]*Message, *uuid.UUID, error)
	GetMessageByID(ctx context.Context, messageID, requesterID uuid.UUID) (*Message, error)
	DeleteMessage(ctx context.Context, messageID, requesterID uuid.UUID) error
}

//go:generate mockery --name RoomClient
type RoomClient interface {
	IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
	GetRoomMembers(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error)
}

type messageService struct {
	repo       repository.MessageRepository
	roomClient RoomClient
	publisher  Publisher
}

func NewMessageService(repo repository.MessageRepository, roomClient RoomClient, publisher Publisher) MessageService {
	return &messageService{
		repo:       repo,
		roomClient: roomClient,
		publisher:  publisher,
	}
}

func (s *messageService) SendMessage(ctx context.Context, input *SendMessageInput) (*Message, error) {
	isMember, err := s.roomClient.IsMember(ctx, input.RoomID, input.SenderID)
	if err != nil {
		return nil, fmt.Errorf("check room membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	if input.ReplyToID != nil {
		original, err := s.repo.GetMessageByID(ctx, *input.ReplyToID)
		if err != nil {
			if errors.Is(err, repository.ErrMessageNotFound) {
				return nil, ErrInvalidReplyTarget
			}
			return nil, fmt.Errorf("get reply target message: %w", err)
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
		Type:      repository.MessageType(input.Type),
		Content:   toRepoContent(input.Content),
		ReplyToID: input.ReplyToID,
	})
	if err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	domainMsg := toMessage(msg)

	recipientIDs, err := s.roomClient.GetRoomMembers(ctx, input.RoomID)
	if err != nil {
		slog.Error("failed to get room members for publish",
			"event_lost", true,
			"error", err,
			"room_id", input.RoomID.String(),
		)
	} else {
		if err := s.publisher.PublishMessageCreated(ctx, domainMsg, recipientIDs); err != nil {
			slog.Error("failed to publish message created event, event may be lost",
				"event_lost", true,
				"error", err,
				"message_id", domainMsg.ID.String(),
				"room_id", domainMsg.RoomID.String(),
			)
		}
	}

	return domainMsg, nil
}

func (s *messageService) GetMessages(ctx context.Context, roomID, requesterID uuid.UUID, cursor *uuid.UUID, limit int) ([]*Message, *uuid.UUID, error) {
	isMember, err := s.roomClient.IsMember(ctx, roomID, requesterID)
	if err != nil {
		return nil, nil, fmt.Errorf("check room membership: %w", err)
	}
	if !isMember {
		return nil, nil, ErrNotMember
	}

	repoMsgs, nextCursor, err := s.repo.GetMessagesByRoomID(ctx, roomID, cursor, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("get messages: %w", err)
	}

	msgs := make([]*Message, len(repoMsgs))
	for i, m := range repoMsgs {
		msgs[i] = toMessage(m)
	}

	return msgs, nextCursor, nil
}

func (s *messageService) GetMessageByID(ctx context.Context, messageID, requesterID uuid.UUID) (*Message, error) {
	msg, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		if errors.Is(err, repository.ErrMessageNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, fmt.Errorf("get message: %w", err)
	}

	isMember, err := s.roomClient.IsMember(ctx, msg.RoomID, requesterID)
	if err != nil {
		return nil, fmt.Errorf("check room membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotMember
	}

	return toMessage(msg), nil
}

func (s *messageService) DeleteMessage(ctx context.Context, messageID, requesterID uuid.UUID) error {
	// DeleteMessage allows the original sender to soft-delete their message.
	// Membership is intentionally not re-checked — a user who sent a message
	// retains the right to delete it even after leaving the room.
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
