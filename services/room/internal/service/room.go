package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/room/internal/repository"
)

var (
	ErrRoomNotFound     = errors.New("room not found")
	ErrNotOwner         = errors.New("only owner can perform this action")
	ErrNotMember        = errors.New("must be a room member to perform this action")
	ErrDirectRoomExists = errors.New("direct room already exists between these users")
)

//go:generate mockery --name Publisher
type Publisher interface {
	PublishMembershipChanged(ctx context.Context, roomID, userID uuid.UUID, action string, recipientIDs []uuid.UUID) error
	PublishRoomUpdated(ctx context.Context, roomID uuid.UUID, name string, recipientIDs []uuid.UUID) error
}

//go:generate mockery --name RoomService
type RoomService interface {
	CreateRoom(ctx context.Context, name string, roomType repository.RoomType, createdBy, participantID uuid.UUID) (*repository.Room, error)
	GetRoom(ctx context.Context, roomID uuid.UUID) (*repository.Room, error)
	UpdateRoom(ctx context.Context, roomID uuid.UUID, requesterID uuid.UUID, name string) (*repository.Room, error)
	DeleteRoom(ctx context.Context, roomID uuid.UUID, requesterID uuid.UUID) error
	AddMember(ctx context.Context, roomID, userID, requesterID uuid.UUID) error
	RemoveMember(ctx context.Context, roomID, userID, requesterID uuid.UUID) error
	GetRoomMembers(ctx context.Context, roomID uuid.UUID) ([]repository.RoomMember, error)
	IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
}

type roomService struct {
	roomRepo  repository.RoomRepository
	publisher Publisher
}

func NewRoomService(roomRepo repository.RoomRepository, publisher Publisher) RoomService {
	return &roomService{
		roomRepo:  roomRepo,
		publisher: publisher,
	}
}

func (s *roomService) CreateRoom(ctx context.Context, name string, roomType repository.RoomType, createdBy, participantID uuid.UUID) (*repository.Room, error) {
	if roomType == repository.RoomTypeDirect && participantID != uuid.Nil {
		existing, err := s.roomRepo.GetDirectRoomByUsers(ctx, createdBy, participantID)
		if err != nil && !errors.Is(err, repository.ErrRoomNotFound) {
			return nil, fmt.Errorf("check existing direct room: %w", err)
		}
		if existing != nil {
			return nil, ErrDirectRoomExists
		}
	}

	room, err := s.roomRepo.CreateRoom(ctx, name, roomType, createdBy)
	if err != nil {
		return nil, fmt.Errorf("create room: %w", err)
	}

	if _, err := s.roomRepo.AddMember(ctx, room.ID, createdBy, repository.MemberRoleOwner); err != nil {
		if !errors.Is(err, repository.ErrMemberAlreadyExists) {
			return nil, fmt.Errorf("add owner as member: %w", err)
		}
	}

	if roomType == repository.RoomTypeDirect && participantID != uuid.Nil {
		if _, err := s.roomRepo.AddMember(ctx, room.ID, participantID, repository.MemberRoleMember); err != nil {
			if !errors.Is(err, repository.ErrMemberAlreadyExists) {
				return nil, fmt.Errorf("add participant as member: %w", err)
			}
		}
	}

	return room, nil
}

func (s *roomService) GetRoom(ctx context.Context, roomID uuid.UUID) (*repository.Room, error) {
	room, err := s.roomRepo.GetRoomByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, repository.ErrRoomNotFound) {
			return nil, ErrRoomNotFound
		}
		return nil, fmt.Errorf("get room: %w", err)
	}

	return room, nil
}

func (s *roomService) UpdateRoom(ctx context.Context, roomID uuid.UUID, requesterID uuid.UUID, name string) (*repository.Room, error) {
	room, err := s.roomRepo.GetRoomByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, repository.ErrRoomNotFound) {
			return nil, ErrRoomNotFound
		}
		return nil, fmt.Errorf("get room: %w", err)
	}

	if room.CreatedBy != requesterID {
		return nil, ErrNotOwner
	}

	updatedRoom, err := s.roomRepo.UpdateRoom(ctx, roomID, name)
	if err != nil {
		if errors.Is(err, repository.ErrRoomNotFound) {
			return nil, ErrRoomNotFound
		}
		return nil, fmt.Errorf("update room: %w", err)
	}

	members, err := s.roomRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		slog.Error("failed to get room members for room.updated event",
			"room_id", roomID,
			"error", err,
		)
		return updatedRoom, nil
	}

	recipientIDs := make([]uuid.UUID, len(members))
	for i, m := range members {
		recipientIDs[i] = m.UserID
	}

	if err := s.publisher.PublishRoomUpdated(ctx, roomID, updatedRoom.Name, recipientIDs); err != nil {
		slog.Error("failed to publish room updated event, event may be lost",
			"room_id", roomID,
			"error", err,
		)
	}

	return updatedRoom, nil
}

func (s *roomService) DeleteRoom(ctx context.Context, roomID uuid.UUID, requesterID uuid.UUID) error {
	room, err := s.roomRepo.GetRoomByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, repository.ErrRoomNotFound) {
			return ErrRoomNotFound
		}
		return fmt.Errorf("get room: %w", err)
	}

	if room.CreatedBy != requesterID {
		return ErrNotOwner
	}

	if err := s.roomRepo.DeleteRoom(ctx, roomID); err != nil {
		return fmt.Errorf("delete room: %w", err)
	}

	return nil
}

func (s *roomService) AddMember(ctx context.Context, roomID, userID, requesterID uuid.UUID) error {
	_, err := s.roomRepo.GetRoomByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, repository.ErrRoomNotFound) {
			return ErrRoomNotFound
		}
		return fmt.Errorf("get room: %w", err)
	}

	isMember, err := s.roomRepo.IsMember(ctx, roomID, requesterID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return ErrNotMember
	}

	_, err = s.roomRepo.AddMember(ctx, roomID, userID, repository.MemberRoleMember)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	members, err := s.roomRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return fmt.Errorf("get members for event: %w", err)
	}

	var recipientIDs []uuid.UUID
	for _, m := range members {
		recipientIDs = append(recipientIDs, m.UserID)
	}

	if s.publisher != nil {
		if err := s.publisher.PublishMembershipChanged(ctx, roomID, userID, "added", recipientIDs); err != nil {
			slog.Error("failed to publish membership changed event, event may be lost",
				"room_id", roomID,
				"user_id", userID,
				"action", "added",
				"error", err,
			)
		}
	}

	return nil
}

// TODO: Add support for members to leave rooms themselves (not just owner removing them).
func (s *roomService) RemoveMember(ctx context.Context, roomID, userID, requesterID uuid.UUID) error {
	room, err := s.roomRepo.GetRoomByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, repository.ErrRoomNotFound) {
			return ErrRoomNotFound
		}
		return fmt.Errorf("get room: %w", err)
	}

	if room.CreatedBy != requesterID {
		return ErrNotOwner
	}

	if err := s.roomRepo.RemoveMember(ctx, roomID, userID); err != nil {
		if errors.Is(err, repository.ErrMemberNotFound) {
			return repository.ErrMemberNotFound
		}
		return fmt.Errorf("remove member: %w", err)
	}

	members, err := s.roomRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return fmt.Errorf("get members for event: %w", err)
	}

	var recipientIDs []uuid.UUID
	for _, m := range members {
		recipientIDs = append(recipientIDs, m.UserID)
	}

	if s.publisher != nil {
		if err := s.publisher.PublishMembershipChanged(ctx, roomID, userID, "removed", recipientIDs); err != nil {
			slog.Error("failed to publish membership changed event, event may be lost",
				"room_id", roomID,
				"user_id", userID,
				"action", "removed",
				"error", err,
			)
		}
	}

	return nil
}

func (s *roomService) GetRoomMembers(ctx context.Context, roomID uuid.UUID) ([]repository.RoomMember, error) {
	// Explicitly check room exists to return a clear error rather than empty slice
	_, err := s.roomRepo.GetRoomByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, repository.ErrRoomNotFound) {
			return nil, ErrRoomNotFound
		}
		return nil, fmt.Errorf("get room: %w", err)
	}

	members, err := s.roomRepo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get room members: %w", err)
	}

	return members, nil
}

func (s *roomService) IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	isMember, err := s.roomRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}

	return isMember, nil
}
