package service

import (
	"time"

	"github.com/google/uuid"
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
