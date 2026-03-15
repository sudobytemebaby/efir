package handler_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/sudobytemebaby/efir/services/auth/internal/handler"
	"github.com/sudobytemebaby/efir/services/auth/internal/repository"
	"github.com/sudobytemebaby/efir/services/auth/internal/service"
	svcmocks "github.com/sudobytemebaby/efir/services/auth/internal/service/mocks"
	authv1 "github.com/sudobytemebaby/efir/services/shared/gen/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newHandler(t *testing.T) (authv1.AuthServiceServer, *svcmocks.AuthService) {
	t.Helper()
	svc := svcmocks.NewAuthService(t)
	h := handler.NewAuthHandler(svc)
	return h, svc
}

// --- Register ---

func TestRegister_Success(t *testing.T) {
	h, svc := newHandler(t)
	ctx := context.Background()
	userID := uuid.New()

	svc.On("Register", ctx, "user@example.com", "pass123").
		Return(&repository.Account{ID: userID, Email: "user@example.com"}, &service.TokenPair{
			AccessToken:  "access",
			RefreshToken: "refresh",
		}, nil).Once()

	resp, err := h.Register(ctx, &authv1.RegisterRequest{
		Email:    "user@example.com",
		Password: "pass123",
	})

	assert.NoError(t, err)
	assert.Equal(t, userID.String(), resp.UserId)
	assert.Equal(t, "access", resp.AccessToken)
	assert.Equal(t, "refresh", resp.RefreshToken)
}

func TestRegister_EmptyEmail(t *testing.T) {
	h, _ := newHandler(t)
	ctx := context.Background()

	_, err := h.Register(ctx, &authv1.RegisterRequest{Email: "", Password: "pass"})

	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestRegister_EmptyPassword(t *testing.T) {
	h, _ := newHandler(t)
	ctx := context.Background()

	_, err := h.Register(ctx, &authv1.RegisterRequest{Email: "user@example.com", Password: ""})

	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestRegister_AlreadyExists(t *testing.T) {
	h, svc := newHandler(t)
	ctx := context.Background()

	svc.On("Register", ctx, "user@example.com", "pass123").
		Return(nil, nil, service.ErrAccountAlreadyExists).Once()

	_, err := h.Register(ctx, &authv1.RegisterRequest{
		Email:    "user@example.com",
		Password: "pass123",
	})

	assert.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

// --- Login ---

func TestLogin_Success(t *testing.T) {
	h, svc := newHandler(t)
	ctx := context.Background()
	userID := uuid.New()

	svc.On("Login", ctx, "user@example.com", "pass123").
		Return(&repository.Account{ID: userID}, &service.TokenPair{
			AccessToken:  "access",
			RefreshToken: "refresh",
		}, nil).Once()

	resp, err := h.Login(ctx, &authv1.LoginRequest{
		Email:    "user@example.com",
		Password: "pass123",
	})

	assert.NoError(t, err)
	assert.Equal(t, userID.String(), resp.UserId)
}

func TestLogin_EmptyFields(t *testing.T) {
	h, _ := newHandler(t)
	ctx := context.Background()

	_, err := h.Login(ctx, &authv1.LoginRequest{Email: "", Password: ""})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestLogin_InvalidCredentials(t *testing.T) {
	h, svc := newHandler(t)
	ctx := context.Background()

	svc.On("Login", ctx, "user@example.com", "wrongpass").
		Return(nil, nil, service.ErrInvalidCredentials).Once()

	_, err := h.Login(ctx, &authv1.LoginRequest{
		Email:    "user@example.com",
		Password: "wrongpass",
	})

	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// --- Logout ---

func TestLogout_Success(t *testing.T) {
	h, svc := newHandler(t)
	ctx := context.Background()

	svc.On("Logout", ctx, "refresh-token").Return(nil).Once()

	_, err := h.Logout(ctx, &authv1.LogoutRequest{RefreshToken: "refresh-token"})
	assert.NoError(t, err)
}

func TestLogout_EmptyToken(t *testing.T) {
	h, _ := newHandler(t)
	ctx := context.Background()

	_, err := h.Logout(ctx, &authv1.LogoutRequest{RefreshToken: ""})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

// --- RefreshToken ---

func TestRefreshToken_Success(t *testing.T) {
	h, svc := newHandler(t)
	ctx := context.Background()

	svc.On("RefreshToken", ctx, "old-refresh").
		Return(&service.TokenPair{AccessToken: "new-access", RefreshToken: "new-refresh"}, nil).Once()

	resp, err := h.RefreshToken(ctx, &authv1.RefreshTokenRequest{RefreshToken: "old-refresh"})

	assert.NoError(t, err)
	assert.Equal(t, "new-access", resp.AccessToken)
	assert.Equal(t, "new-refresh", resp.RefreshToken)
}

func TestRefreshToken_EmptyToken(t *testing.T) {
	h, _ := newHandler(t)
	ctx := context.Background()

	_, err := h.RefreshToken(ctx, &authv1.RefreshTokenRequest{RefreshToken: ""})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	h, svc := newHandler(t)
	ctx := context.Background()

	svc.On("RefreshToken", ctx, mock.AnythingOfType("string")).
		Return(nil, service.ErrInvalidToken).Once()

	_, err := h.RefreshToken(ctx, &authv1.RefreshTokenRequest{RefreshToken: "expired-token"})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// --- ValidateToken ---

func TestValidateToken_Success(t *testing.T) {
	h, svc := newHandler(t)
	ctx := context.Background()
	userID := uuid.New()

	svc.On("ValidateToken", ctx, "valid-token").Return(userID, nil).Once()

	resp, err := h.ValidateToken(ctx, &authv1.ValidateTokenRequest{AccessToken: "valid-token"})

	assert.NoError(t, err)
	assert.Equal(t, userID.String(), resp.UserId)
}

func TestValidateToken_EmptyToken(t *testing.T) {
	h, _ := newHandler(t)
	ctx := context.Background()

	_, err := h.ValidateToken(ctx, &authv1.ValidateTokenRequest{AccessToken: ""})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestValidateToken_Expired(t *testing.T) {
	h, svc := newHandler(t)
	ctx := context.Background()

	svc.On("ValidateToken", ctx, "expired-token").Return(uuid.Nil, service.ErrExpiredToken).Once()

	_, err := h.ValidateToken(ctx, &authv1.ValidateTokenRequest{AccessToken: "expired-token"})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestValidateToken_Invalid(t *testing.T) {
	h, svc := newHandler(t)
	ctx := context.Background()

	svc.On("ValidateToken", ctx, "garbage").Return(uuid.Nil, service.ErrInvalidToken).Once()

	_, err := h.ValidateToken(ctx, &authv1.ValidateTokenRequest{AccessToken: "garbage"})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}
