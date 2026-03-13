package handler

import (
	"context"
	"errors"

	"github.com/sudobytemebaby/efir/services/auth/internal/service"
	authv1 "github.com/sudobytemebaby/efir/services/shared/gen/auth"
	sharederrors "github.com/sudobytemebaby/efir/services/shared/pkg/errors"
)

type authHandler struct {
	authv1.UnimplementedAuthServiceServer
	svc service.AuthService
}

func NewAuthHandler(svc service.AuthService) authv1.AuthServiceServer {
	return &authHandler{svc: svc}
}

func (h *authHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, sharederrors.CodeInvalidArgument.Error("email and password are required")
	}

	acc, tokens, err := h.svc.Register(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrAccountAlreadyExists) {
			return nil, sharederrors.CodeAlreadyExists.Error("account already exists")
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &authv1.RegisterResponse{
		UserId:       acc.ID.String(),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (h *authHandler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, sharederrors.CodeInvalidArgument.Error("email and password are required")
	}

	acc, tokens, err := h.svc.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, sharederrors.CodeUnauthenticated.Error("invalid credentials")
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &authv1.LoginResponse{
		UserId:       acc.ID.String(),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (h *authHandler) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if req.RefreshToken == "" {
		return nil, sharederrors.CodeInvalidArgument.Error("refresh token is required")
	}

	if err := h.svc.Logout(ctx, req.RefreshToken); err != nil {
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &authv1.LogoutResponse{}, nil
}

func (h *authHandler) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, sharederrors.CodeInvalidArgument.Error("refresh token is required")
	}

	tokens, err := h.svc.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			return nil, sharederrors.CodeUnauthenticated.Error("invalid or expired refresh token")
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &authv1.RefreshTokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (h *authHandler) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	if req.AccessToken == "" {
		return nil, sharederrors.CodeInvalidArgument.Error("access token is required")
	}

	userID, err := h.svc.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) || errors.Is(err, service.ErrExpiredToken) {
			return nil, sharederrors.CodeUnauthenticated.Error(err.Error())
		}
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &authv1.ValidateTokenResponse{
		UserId: userID.String(),
	}, nil
}
