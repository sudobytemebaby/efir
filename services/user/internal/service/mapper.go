package service

import (
	"github.com/sudobytemebaby/efir/services/user/internal/repository"
)

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
