package handler

import (
	"github.com/sudobytemebaby/efir/services/room/internal/service"
	roomv1 "github.com/sudobytemebaby/efir/services/shared/gen/room"
	"github.com/sudobytemebaby/efir/services/shared/pkg/mapper"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var roomTypeToProto = map[service.RoomType]roomv1.RoomType{
	service.RoomTypeDirect: roomv1.RoomType_ROOM_TYPE_DIRECT,
	service.RoomTypeGroup:  roomv1.RoomType_ROOM_TYPE_GROUP,
}

const roomTypeDefaultProto = roomv1.RoomType_ROOM_TYPE_UNSPECIFIED

var protoToRoomTypeMap = map[roomv1.RoomType]service.RoomType{
	roomv1.RoomType_ROOM_TYPE_DIRECT: service.RoomTypeDirect,
	roomv1.RoomType_ROOM_TYPE_GROUP:  service.RoomTypeGroup,
}

func roomToProto(room *service.Room) *roomv1.Room {
	return &roomv1.Room{
		RoomId:    room.ID.String(),
		Name:      room.Name,
		Type:      mapper.Enum(roomTypeToProto, room.Type, roomTypeDefaultProto),
		CreatedBy: room.CreatedBy.String(),
		CreatedAt: timestamppb.New(room.CreatedAt),
		UpdatedAt: timestamppb.New(room.UpdatedAt),
	}
}

func protoToRoomType(rt roomv1.RoomType) (service.RoomType, bool) {
	return mapper.EnumWithOk(protoToRoomTypeMap, rt)
}
