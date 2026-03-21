package client

import (
	"context"
	"time"

	userv1 "github.com/sudobytemebaby/efir/services/shared/gen/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type User = userv1.User

type UserClient struct {
	client  userv1.UserServiceClient
	timeout time.Duration
}

func NewUserClient(addr string, timeout time.Duration) (*UserClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &UserClient{
		client:  userv1.NewUserServiceClient(conn),
		timeout: timeout,
	}, nil
}

func (c *UserClient) GetUser(ctx context.Context, userID string) (*userv1.GetUserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.GetUser(ctx, &userv1.GetUserRequest{
		UserId: userID,
	})
}

func (c *UserClient) GetUsersByIds(ctx context.Context, userIDs []string) (*userv1.GetUsersByIdsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.GetUsersByIds(ctx, &userv1.GetUsersByIdsRequest{
		UserIds: userIDs,
	})
}

func (c *UserClient) UpdateUser(ctx context.Context, userID string, displayName, avatarURL, bio *string) (*userv1.UpdateUserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.UpdateUser(ctx, &userv1.UpdateUserRequest{
		UserId:      userID,
		DisplayName: displayName,
		AvatarUrl:   avatarURL,
		Bio:         bio,
	})
}
