package client

import (
	"context"
	"time"

	"github.com/google/uuid"
	roomv1 "github.com/sudobytemebaby/efir/services/shared/gen/room"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type RoomClient struct {
	client  roomv1.RoomServiceClient
	timeout time.Duration
}

func NewRoomClient(addr string, timeout time.Duration) (*RoomClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &RoomClient{
		client:  roomv1.NewRoomServiceClient(conn),
		timeout: timeout,
	}, nil
}

func (c *RoomClient) IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	return withRetry(ctx, func(ctx context.Context) (bool, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()

		resp, err := c.client.IsMember(ctx, &roomv1.IsMemberRequest{
			RoomId: roomID.String(),
			UserId: userID.String(),
		})
		if err != nil {
			return false, err
		}
		return resp.IsMember, nil
	})
}

func (c *RoomClient) GetRoomMembers(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error) {
	return withRetry(ctx, func(ctx context.Context) ([]uuid.UUID, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()

		resp, err := c.client.GetRoomMembers(ctx, &roomv1.GetRoomMembersRequest{
			RoomId: roomID.String(),
		})
		if err != nil {
			return nil, err
		}

		result := make([]uuid.UUID, 0, len(resp.UserIds))
		for _, idStr := range resp.UserIds {
			id, err := uuid.Parse(idStr)
			if err != nil {
				continue
			}
			result = append(result, id)
		}
		return result, nil
	})
}

func withRetry[T any](ctx context.Context, fn func(ctx context.Context) (T, error)) (T, error) {
	delays := []time.Duration{100 * time.Millisecond, 300 * time.Millisecond, 900 * time.Millisecond}
	var result T
	var err error

	for i, delay := range delays {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		result, err = fn(ctx)
		if err == nil {
			return result, nil
		}

		st, ok := status.FromError(err)
		if !ok || (st.Code() != codes.Unavailable && st.Code() != codes.DeadlineExceeded) {
			return result, err
		}

		if i < len(delays)-1 {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return result, err
}
