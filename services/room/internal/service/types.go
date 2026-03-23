package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/room/internal/repository"
)

type RoomType string

const (
	RoomTypeDirect RoomType = "direct"
	RoomTypeGroup  RoomType = "group"
)

type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleMember MemberRole = "member"
)

type Room struct {
	ID        uuid.UUID
	Name      string
	Type      RoomType
	CreatedBy uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RoomMember struct {
	RoomID   uuid.UUID
	UserID   uuid.UUID
	Role     MemberRole
	JoinedAt time.Time
}

func toRoom(r *repository.Room) *Room {
	return &Room{
		ID:        r.ID,
		Name:      r.Name,
		Type:      RoomType(r.Type),
		CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func toRoomMember(m repository.RoomMember) RoomMember {
	return RoomMember{
		RoomID:   m.RoomID,
		UserID:   m.UserID,
		Role:     MemberRole(m.Role),
		JoinedAt: m.JoinedAt,
	}
}
