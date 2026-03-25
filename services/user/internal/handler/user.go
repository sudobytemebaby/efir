package handler

import (
	"context"
	"errors"

	"buf.build/go/protovalidate"
	"github.com/google/uuid"
	userv1 "github.com/sudobytemebaby/efir/services/shared/gen/user"
	sharederrors "github.com/sudobytemebaby/efir/services/shared/pkg/errors"
	"github.com/sudobytemebaby/efir/services/user/internal/service"
	"google.golang.org/protobuf/proto"
)

type userHandler struct {
	userv1.UnimplementedUserServiceServer
	svc       service.UserService
	validator protovalidate.Validator
}

func NewUserHandler(svc service.UserService) (userv1.UserServiceServer, error) {
	v, err := protovalidate.New()
	if err != nil {
		return nil, err
	}
	return &userHandler{svc: svc, validator: v}, nil
}

func (h *userHandler) validate(msg proto.Message) error {
	if err := h.validator.Validate(msg); err != nil {
		return sharederrors.CodeInvalidArgument.Error(err.Error())
	}
	return nil
}

func (h *userHandler) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid user_id")
	}

	user, err := h.svc.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return nil, sharederrors.CodeNotFound.Error("user not found")
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &userv1.GetUserResponse{
		User: userToProto(*user),
	}, nil
}

func (h *userHandler) GetUsersByIds(ctx context.Context, req *userv1.GetUsersByIdsRequest) (*userv1.GetUsersByIdsResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
	}

	userIDs := make([]uuid.UUID, 0, len(req.UserIds))
	for _, idStr := range req.UserIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, sharederrors.CodeInvalidArgument.Error("invalid user_id")
		}
		userIDs = append(userIDs, id)
	}

	users, err := h.svc.GetUsers(ctx, userIDs)
	if err != nil {
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &userv1.GetUsersByIdsResponse{
		Users: usersToProto(users),
	}, nil
}

func (h *userHandler) UpdateUser(ctx context.Context, req *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid user_id")
	}

	user, err := h.svc.UpdateUser(ctx, userID, req.DisplayName, req.AvatarUrl, req.Bio)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return nil, sharederrors.CodeNotFound.Error("user not found")
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &userv1.UpdateUserResponse{
		User: userToProto(*user),
	}, nil
}
