package client

import (
	"context"
	"time"

	authv1 "github.com/sudobytemebaby/efir/services/shared/gen/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthClientInterface interface {
	Register(ctx context.Context, email, password string) (*authv1.RegisterResponse, error)
	Login(ctx context.Context, email, password string) (*authv1.LoginResponse, error)
	Logout(ctx context.Context, refreshToken string) (*authv1.LogoutResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*authv1.RefreshTokenResponse, error)
}

var _ AuthClientInterface = (*AuthClient)(nil)

type AuthClient struct {
	client  authv1.AuthServiceClient
	timeout time.Duration
	conn    *grpc.ClientConn
}

func (c *AuthClient) Close() error {
	return c.conn.Close()
}

func NewAuthClient(addr string, timeout time.Duration) (*AuthClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &AuthClient{
		client:  authv1.NewAuthServiceClient(conn),
		timeout: timeout,
		conn:    conn,
	}, nil
}

func (c *AuthClient) Register(ctx context.Context, email, password string) (*authv1.RegisterResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.Register(ctx, &authv1.RegisterRequest{
		Email:    email,
		Password: password,
	})
}

func (c *AuthClient) Login(ctx context.Context, email, password string) (*authv1.LoginResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.Login(ctx, &authv1.LoginRequest{
		Email:    email,
		Password: password,
	})
}

func (c *AuthClient) Logout(ctx context.Context, refreshToken string) (*authv1.LogoutResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.Logout(ctx, &authv1.LogoutRequest{
		RefreshToken: refreshToken,
	})
}

func (c *AuthClient) RefreshToken(ctx context.Context, refreshToken string) (*authv1.RefreshTokenResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.RefreshToken(ctx, &authv1.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})
}
