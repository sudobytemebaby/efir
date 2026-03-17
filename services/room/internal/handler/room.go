package handler

import (
	"context"
	"errors"

	"buf.build/go/protovalidate"
	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/room/internal/repository"
	"github.com/sudobytemebaby/efir/services/room/internal/service"
	roomv1 "github.com/sudobytemebaby/efir/services/shared/gen/room"
	sharederrors "github.com/sudobytemebaby/efir/services/shared/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type roomHandler struct {
	roomv1.UnimplementedRoomServiceServer
	svc       service.RoomService
	validator protovalidate.Validator
}

func NewRoomHandler(svc service.RoomService) (roomv1.RoomServiceServer, error) {
	v, err := protovalidate.New()
	if err != nil {
		return nil, err
	}
	return &roomHandler{svc: svc, validator: v}, nil
}

func (h *roomHandler) validate(msg proto.Message) error {
	if err := h.validator.Validate(msg); err != nil {
		return sharederrors.CodeInvalidArgument.Error(err.Error())
	}
	return nil
}

func (h *roomHandler) CreateRoom(ctx context.Context, req *roomv1.CreateRoomRequest) (*roomv1.CreateRoomResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid created_by")
	}

	var participantID uuid.UUID
	if req.ParticipantId != "" {
		participantID, err = uuid.Parse(req.ParticipantId)
		if err != nil {
			return nil, sharederrors.CodeInvalidArgument.Error("invalid participant_id")
		}
	}

	var roomType repository.RoomType
	switch req.Type {
	case roomv1.RoomType_ROOM_TYPE_DIRECT:
		roomType = repository.RoomTypeDirect
	case roomv1.RoomType_ROOM_TYPE_GROUP:
		roomType = repository.RoomTypeGroup
	default:
		roomType = repository.RoomTypeGroup
	}

	room, err := h.svc.CreateRoom(ctx, req.Name, roomType, createdBy, participantID)
	if err != nil {
		if errors.Is(err, service.ErrDirectRoomExists) {
			return nil, sharederrors.CodeAlreadyExists.Error("direct room already exists between these users")
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &roomv1.CreateRoomResponse{
		Room: mapRoomToProto(room),
	}, nil
}

func (h *roomHandler) GetRoom(ctx context.Context, req *roomv1.GetRoomRequest) (*roomv1.GetRoomResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
	}

	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid room_id")
	}

	room, err := h.svc.GetRoom(ctx, roomID)
	if err != nil {
		if errors.Is(err, service.ErrRoomNotFound) {
			return nil, sharederrors.CodeNotFound.Error("room not found")
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &roomv1.GetRoomResponse{
		Room: mapRoomToProto(room),
	}, nil
}

func (h *roomHandler) UpdateRoom(ctx context.Context, req *roomv1.UpdateRoomRequest) (*roomv1.UpdateRoomResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
	}

	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid room_id")
	}

	requesterID, err := uuid.Parse(req.RequesterId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid requester_id")
	}

	if req.Name == nil || *req.Name == "" {
		return nil, sharederrors.CodeInvalidArgument.Error("name is required")
	}

	room, err := h.svc.UpdateRoom(ctx, roomID, requesterID, *req.Name)
	if err != nil {
		if errors.Is(err, service.ErrRoomNotFound) {
			return nil, sharederrors.CodeNotFound.Error("room not found")
		}
		if errors.Is(err, service.ErrNotOwner) {
			return nil, sharederrors.CodePermissionDenied.Error("only owner can update room")
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &roomv1.UpdateRoomResponse{
		Room: mapRoomToProto(room),
	}, nil
}

func (h *roomHandler) DeleteRoom(ctx context.Context, req *roomv1.DeleteRoomRequest) (*roomv1.DeleteRoomResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
	}

	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid room_id")
	}

	requesterID, err := uuid.Parse(req.RequesterId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid requester_id")
	}

	err = h.svc.DeleteRoom(ctx, roomID, requesterID)
	if err != nil {
		if errors.Is(err, service.ErrRoomNotFound) {
			return nil, sharederrors.CodeNotFound.Error("room not found")
		}
		if errors.Is(err, service.ErrNotOwner) {
			return nil, sharederrors.CodePermissionDenied.Error("only owner can delete room")
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &roomv1.DeleteRoomResponse{}, nil
}

func (h *roomHandler) AddMember(ctx context.Context, req *roomv1.AddMemberRequest) (*roomv1.AddMemberResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
	}

	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid room_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid user_id")
	}

	requesterID, err := uuid.Parse(req.RequesterId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid requester_id")
	}

	err = h.svc.AddMember(ctx, roomID, userID, requesterID)
	if err != nil {
		if errors.Is(err, service.ErrRoomNotFound) {
			return nil, sharederrors.CodeNotFound.Error("room not found")
		}
		if errors.Is(err, service.ErrNotMember) {
			return nil, sharederrors.CodePermissionDenied.Error("must be a room member to add new members")
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &roomv1.AddMemberResponse{}, nil
}

func (h *roomHandler) RemoveMember(ctx context.Context, req *roomv1.RemoveMemberRequest) (*roomv1.RemoveMemberResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
	}

	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid room_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid user_id")
	}

	requesterID, err := uuid.Parse(req.RequesterId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid requester_id")
	}

	err = h.svc.RemoveMember(ctx, roomID, userID, requesterID)
	if err != nil {
		if errors.Is(err, service.ErrRoomNotFound) {
			return nil, sharederrors.CodeNotFound.Error("room not found")
		}
		if errors.Is(err, service.ErrNotOwner) {
			return nil, sharederrors.CodePermissionDenied.Error("only owner can remove members")
		}
		if errors.Is(err, repository.ErrMemberNotFound) {
			return nil, sharederrors.CodeNotFound.Error("member not found")
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &roomv1.RemoveMemberResponse{}, nil
}

func (h *roomHandler) GetRoomMembers(ctx context.Context, req *roomv1.GetRoomMembersRequest) (*roomv1.GetRoomMembersResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
	}

	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid room_id")
	}

	members, err := h.svc.GetRoomMembers(ctx, roomID)
	if err != nil {
		if errors.Is(err, service.ErrRoomNotFound) {
			return nil, sharederrors.CodeNotFound.Error("room not found")
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	var userIDs []string
	for _, m := range members {
		userIDs = append(userIDs, m.UserID.String())
	}

	return &roomv1.GetRoomMembersResponse{
		UserIds: userIDs,
	}, nil
}

func (h *roomHandler) IsMember(ctx context.Context, req *roomv1.IsMemberRequest) (*roomv1.IsMemberResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
	}

	roomID, err := uuid.Parse(req.RoomId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid room_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, sharederrors.CodeInvalidArgument.Error("invalid user_id")
	}

	isMember, err := h.svc.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &roomv1.IsMemberResponse{
		IsMember: isMember,
	}, nil
}

func mapRoomToProto(room *repository.Room) *roomv1.Room {
	return &roomv1.Room{
		RoomId:    room.ID.String(),
		Name:      room.Name,
		Type:      mapRoomTypeToProto(room.Type),
		CreatedBy: room.CreatedBy.String(),
		CreatedAt: timestamppb.New(room.CreatedAt),
		UpdatedAt: timestamppb.New(room.UpdatedAt),
	}
}

func mapRoomTypeToProto(t repository.RoomType) roomv1.RoomType {
	switch t {
	case repository.RoomTypeDirect:
		return roomv1.RoomType_ROOM_TYPE_DIRECT
	case repository.RoomTypeGroup:
		return roomv1.RoomType_ROOM_TYPE_GROUP
	default:
		return roomv1.RoomType_ROOM_TYPE_UNSPECIFIED
	}
}
