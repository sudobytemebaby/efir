package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/sudobytemebaby/efir/services/auth/internal/ratelimit"
	ratelimitmocks "github.com/sudobytemebaby/efir/services/auth/internal/ratelimit/mocks"
	"github.com/sudobytemebaby/efir/services/auth/internal/repository"
	repomocks "github.com/sudobytemebaby/efir/services/auth/internal/repository/mocks"
	"github.com/sudobytemebaby/efir/services/auth/internal/service"
	svcmocks "github.com/sudobytemebaby/efir/services/auth/internal/service/mocks"
	"golang.org/x/crypto/bcrypt"
)

func newSvc(t *testing.T) (service.AuthService, *repomocks.AccountRepository, *repomocks.TokenRepository, *svcmocks.Publisher, *ratelimitmocks.Limiter) {
	t.Helper()
	accountRepo := repomocks.NewAccountRepository(t)
	tokenRepo := repomocks.NewTokenRepository(t)
	publisher := svcmocks.NewPublisher(t)
	limiter := ratelimitmocks.NewLimiter(t)
	svc := service.NewAuthService(accountRepo, tokenRepo, publisher, limiter, "secret", time.Minute, time.Hour)
	return svc, accountRepo, tokenRepo, publisher, limiter
}

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()
	email := "test@example.com"
	password := "password"
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc, accountRepo, tokenRepo, publisher, limiter := newSvc(t)

		limiter.On("Allow", ctx, ratelimit.ActionRegister, email).Return(nil).Once()
		accountRepo.On("GetAccountByEmail", ctx, email).Return(nil, repository.ErrAccountNotFound).Once()
		accountRepo.On("CreateAccount", ctx, email, mock.AnythingOfType("string")).Return(&repository.Account{
			ID:    userID,
			Email: email,
		}, nil).Once()
		tokenRepo.On("SaveRefreshToken", ctx, userID, mock.AnythingOfType("string"), time.Hour).Return(nil).Once()
		publisher.On("PublishUserRegistered", ctx, userID, email).Return(nil).Once()

		acc, tokens, err := svc.Register(ctx, email, password)

		assert.NoError(t, err)
		assert.NotNil(t, acc)
		assert.Equal(t, userID, acc.ID)
		assert.NotNil(t, tokens)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("rate limit exceeded", func(t *testing.T) {
		svc, _, _, _, limiter := newSvc(t)

		limiter.On("Allow", ctx, ratelimit.ActionRegister, email).
			Return(&ratelimit.ErrRateLimitExceeded{Action: ratelimit.ActionRegister, Email: email}).Once()

		acc, tokens, err := svc.Register(ctx, email, password)

		assert.ErrorIs(t, err, service.ErrRateLimitExceeded)
		assert.Nil(t, acc)
		assert.Nil(t, tokens)
	})

	t.Run("account already exists", func(t *testing.T) {
		svc, accountRepo, _, _, limiter := newSvc(t)

		limiter.On("Allow", ctx, ratelimit.ActionRegister, email).Return(nil).Once()
		accountRepo.On("GetAccountByEmail", ctx, email).Return(&repository.Account{ID: userID}, nil).Once()

		acc, tokens, err := svc.Register(ctx, email, password)

		assert.ErrorIs(t, err, service.ErrAccountAlreadyExists)
		assert.Nil(t, acc)
		assert.Nil(t, tokens)
	})

	t.Run("nats publish fails but registration succeeds", func(t *testing.T) {
		svc, accountRepo, tokenRepo, publisher, limiter := newSvc(t)

		limiter.On("Allow", ctx, ratelimit.ActionRegister, email).Return(nil).Once()
		accountRepo.On("GetAccountByEmail", ctx, email).Return(nil, repository.ErrAccountNotFound).Once()
		accountRepo.On("CreateAccount", ctx, email, mock.AnythingOfType("string")).Return(&repository.Account{
			ID:    userID,
			Email: email,
		}, nil).Once()
		tokenRepo.On("SaveRefreshToken", ctx, userID, mock.AnythingOfType("string"), time.Hour).Return(nil).Once()
		publisher.On("PublishUserRegistered", ctx, userID, email).
			Return(errors.New("nats: connection refused")).Once()

		acc, tokens, err := svc.Register(ctx, email, password)

		assert.NoError(t, err)
		assert.NotNil(t, acc)
		assert.NotNil(t, tokens)
	})
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()
	email := "test@example.com"
	password := "password"
	userID := uuid.New()
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	t.Run("success", func(t *testing.T) {
		svc, accountRepo, tokenRepo, _, limiter := newSvc(t)

		limiter.On("Allow", ctx, ratelimit.ActionLogin, email).Return(nil).Once()
		accountRepo.On("GetAccountByEmail", ctx, email).Return(&repository.Account{
			ID:           userID,
			Email:        email,
			PasswordHash: string(hashedPassword),
		}, nil).Once()
		tokenRepo.On("SaveRefreshToken", ctx, userID, mock.AnythingOfType("string"), time.Hour).Return(nil).Once()

		acc, tokens, err := svc.Login(ctx, email, password)

		assert.NoError(t, err)
		assert.NotNil(t, acc)
		assert.Equal(t, userID, acc.ID)
		assert.NotNil(t, tokens)
	})

	t.Run("rate limit exceeded", func(t *testing.T) {
		svc, _, _, _, limiter := newSvc(t)

		limiter.On("Allow", ctx, ratelimit.ActionLogin, email).
			Return(&ratelimit.ErrRateLimitExceeded{Action: ratelimit.ActionLogin, Email: email}).Once()

		acc, tokens, err := svc.Login(ctx, email, password)

		assert.ErrorIs(t, err, service.ErrRateLimitExceeded)
		assert.Nil(t, acc)
		assert.Nil(t, tokens)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		svc, accountRepo, _, _, limiter := newSvc(t)

		limiter.On("Allow", ctx, ratelimit.ActionLogin, email).Return(nil).Once()
		accountRepo.On("GetAccountByEmail", ctx, email).Return(&repository.Account{
			ID:           userID,
			Email:        email,
			PasswordHash: "wrong_hash",
		}, nil).Once()

		acc, tokens, err := svc.Login(ctx, email, password)

		assert.ErrorIs(t, err, service.ErrInvalidCredentials)
		assert.Nil(t, acc)
		assert.Nil(t, tokens)
	})
}

func TestAuthService_Logout(t *testing.T) {
	ctx := context.Background()
	svc, _, tokenRepo, _, _ := newSvc(t)
	token := "refresh_token"

	t.Run("success", func(t *testing.T) {
		tokenRepo.On("DeleteRefreshToken", ctx, token).Return(nil).Once()
		err := svc.Logout(ctx, token)
		assert.NoError(t, err)
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	oldToken := "old_refresh_token"

	t.Run("success", func(t *testing.T) {
		svc, _, tokenRepo, _, _ := newSvc(t)

		tokenRepo.On("GetUserIDByRefreshToken", ctx, oldToken).Return(userID, nil).Once()
		tokenRepo.On("SaveRefreshToken", ctx, userID, mock.AnythingOfType("string"), time.Hour).Return(nil).Once()
		tokenRepo.On("DeleteRefreshToken", ctx, oldToken).Return(nil).Once()

		tokens, err := svc.RefreshToken(ctx, oldToken)

		assert.NoError(t, err)
		require.NotNil(t, tokens)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
		assert.NotEqual(t, oldToken, tokens.RefreshToken)
	})

	t.Run("invalid token returns ErrInvalidToken", func(t *testing.T) {
		svc, _, tokenRepo, _, _ := newSvc(t)

		tokenRepo.On("GetUserIDByRefreshToken", ctx, oldToken).Return(uuid.UUID{}, repository.ErrTokenNotFound).Once()

		tokens, err := svc.RefreshToken(ctx, oldToken)

		assert.ErrorIs(t, err, service.ErrInvalidToken)
		assert.Nil(t, tokens)
	})
}
