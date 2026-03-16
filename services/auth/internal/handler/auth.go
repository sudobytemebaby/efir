package handler

import (
	"context"
	"errors"

	"buf.build/go/protovalidate"
	"github.com/sudobytemebaby/efir/services/auth/internal/service"
	authv1 "github.com/sudobytemebaby/efir/services/shared/gen/auth"
	sharederrors "github.com/sudobytemebaby/efir/services/shared/pkg/errors"
	"google.golang.org/protobuf/proto"
)

type authHandler struct {
	authv1.UnimplementedAuthServiceServer
	svc       service.AuthService
	validator protovalidate.Validator
}

func NewAuthHandler(svc service.AuthService) (authv1.AuthServiceServer, error) {
	v, err := protovalidate.New()
	if err != nil {
		return nil, err
	}
	return &authHandler{svc: svc, validator: v}, nil
}

func (h *authHandler) validate(msg proto.Message) error {
	if err := h.validator.Validate(msg); err != nil {
		return sharederrors.CodeInvalidArgument.Error(err.Error())
	}
	return nil
}

func (h *authHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
	}

	acc, tokens, err := h.svc.Register(ctx, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAccountAlreadyExists):
			return nil, sharederrors.CodeAlreadyExists.Error("account already exists")
		case errors.Is(err, service.ErrRateLimitExceeded):
			return nil, sharederrors.CodeUnavailable.Error("too many requests, please try again later")
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
	if err := h.validate(req); err != nil {
		return nil, err
	}

	acc, tokens, err := h.svc.Login(ctx, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			return nil, sharederrors.CodeUnauthenticated.Error("invalid credentials")
		case errors.Is(err, service.ErrRateLimitExceeded):
			return nil, sharederrors.CodeUnavailable.Error("too many requests, please try again later")
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
	if err := h.validate(req); err != nil {
		return nil, err
	}

	if err := h.svc.Logout(ctx, req.RefreshToken); err != nil {
		return nil, sharederrors.CodeInternal.Wrap(err)
	}

	return &authv1.LogoutResponse{}, nil
}

func (h *authHandler) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	if err := h.validate(req); err != nil {
		return nil, err
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
	if err := h.validate(req); err != nil {
		return nil, err
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
