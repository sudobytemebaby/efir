package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/user/internal/repository"
)

type User struct {
	ID          uuid.UUID
	Username    string
	DisplayName string
	AvatarURL   *string
	Bio         *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func toUser(r *repository.User) *User {
	return &User{
		ID:          r.ID,
		Username:    r.Username,
		DisplayName: r.DisplayName,
		AvatarURL:   r.AvatarURL,
		Bio:         r.Bio,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}
