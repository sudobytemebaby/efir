package handler

import (
	userv1 "github.com/sudobytemebaby/efir/services/shared/gen/user"
	"github.com/sudobytemebaby/efir/services/shared/pkg/mapper"
	"github.com/sudobytemebaby/efir/services/user/internal/service"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func userToProto(u service.User) *userv1.User {
	return &userv1.User{
		UserId:      u.ID.String(),
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarUrl:   u.AvatarURL,
		Bio:         u.Bio,
		CreatedAt:   timestamppb.New(u.CreatedAt),
		UpdatedAt:   timestamppb.New(u.UpdatedAt),
	}
}

func usersToProto(users []service.User) []*userv1.User {
	return mapper.Slice(users, userToProto)
}
