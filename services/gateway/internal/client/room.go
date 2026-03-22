package client

import (
	"context"
	"time"

	roomv1 "github.com/sudobytemebaby/efir/services/shared/gen/room"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Room = roomv1.Room
type RoomType = roomv1.RoomType

const (
	RoomTypeDirect RoomType = roomv1.RoomType_ROOM_TYPE_DIRECT
	RoomTypeGroup  RoomType = roomv1.RoomType_ROOM_TYPE_GROUP
)

type RoomClient struct {
	client  roomv1.RoomServiceClient
	timeout time.Duration
	conn    *grpc.ClientConn
}

func NewRoomClient(addr string, timeout time.Duration) (*RoomClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &RoomClient{
		client:  roomv1.NewRoomServiceClient(conn),
		timeout: timeout,
		conn:    conn,
	}, nil
}

func (c *RoomClient) Close() error {
	return c.conn.Close()
}

func (c *RoomClient) CreateRoom(ctx context.Context, name string, roomType RoomType, createdBy, participantID string) (*roomv1.CreateRoomResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.CreateRoom(ctx, &roomv1.CreateRoomRequest{
		Name:          name,
		Type:          roomType,
		CreatedBy:     createdBy,
		ParticipantId: participantID,
	})
}

func (c *RoomClient) GetRoom(ctx context.Context, roomID string) (*roomv1.GetRoomResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.GetRoom(ctx, &roomv1.GetRoomRequest{
		RoomId: roomID,
	})
}

func (c *RoomClient) UpdateRoom(ctx context.Context, roomID, requesterID string, name *string) (*roomv1.UpdateRoomResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.UpdateRoom(ctx, &roomv1.UpdateRoomRequest{
		RoomId:      roomID,
		RequesterId: requesterID,
		Name:        name,
	})
}

func (c *RoomClient) DeleteRoom(ctx context.Context, roomID, requesterID string) (*roomv1.DeleteRoomResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.DeleteRoom(ctx, &roomv1.DeleteRoomRequest{
		RoomId:      roomID,
		RequesterId: requesterID,
	})
}

func (c *RoomClient) AddMember(ctx context.Context, roomID, userID, requesterID string) (*roomv1.AddMemberResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.AddMember(ctx, &roomv1.AddMemberRequest{
		RoomId:      roomID,
		UserId:      userID,
		RequesterId: requesterID,
	})
}

func (c *RoomClient) RemoveMember(ctx context.Context, roomID, userID, requesterID string) (*roomv1.RemoveMemberResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.RemoveMember(ctx, &roomv1.RemoveMemberRequest{
		RoomId:      roomID,
		UserId:      userID,
		RequesterId: requesterID,
	})
}
