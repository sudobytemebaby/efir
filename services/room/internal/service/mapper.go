package service

import (
	"github.com/sudobytemebaby/efir/services/room/internal/repository"
)

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
