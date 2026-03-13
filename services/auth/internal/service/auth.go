package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sudobytemebaby/efir/services/auth/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrAccountAlreadyExists = errors.New("account already exists")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrInvalidToken         = errors.New("invalid token")
	ErrExpiredToken         = errors.New("expired token")
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

//go:generate mockery --name AuthService
type AuthService interface {
	Register(ctx context.Context, email, password string) (*repository.Account, *TokenPair, error)
	Login(ctx context.Context, email, password string) (*repository.Account, *TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
	ValidateToken(ctx context.Context, accessToken string) (uuid.UUID, error)
}

//go:generate mockery --name Publisher
type Publisher interface {
	PublishUserRegistered(ctx context.Context, userID uuid.UUID, email string) error
}

type authService struct {
	accountRepo repository.AccountRepository
	tokenRepo   repository.TokenRepository
	publisher   Publisher
	jwtSecret   []byte
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

func NewAuthService(
	accountRepo repository.AccountRepository,
	tokenRepo repository.TokenRepository,
	publisher Publisher,
	jwtSecret string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) AuthService {
	return &authService{
		accountRepo: accountRepo,
		tokenRepo:   tokenRepo,
		publisher:   publisher,
		jwtSecret:   []byte(jwtSecret),
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
	}
}

func (s *authService) Register(ctx context.Context, email, password string) (*repository.Account, *TokenPair, error) {
	existing, err := s.accountRepo.GetAccountByEmail(ctx, email)
	if err != nil && !errors.Is(err, repository.ErrAccountNotFound) {
		return nil, nil, fmt.Errorf("check existing account: %w", err)
	}
	if existing != nil {
		return nil, nil, ErrAccountAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	acc, err := s.accountRepo.CreateAccount(ctx, email, string(hashedPassword))
	if err != nil {
		return nil, nil, fmt.Errorf("create account: %w", err)
	}

	tokenPair, err := s.generateTokenPair(ctx, acc.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	if err := s.publisher.PublishUserRegistered(ctx, acc.ID, acc.Email); err != nil {
		// Let's keep it simple and return error if it fails
		return nil, nil, fmt.Errorf("publish user registered event: %w", err)
	}

	return acc, tokenPair, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (*repository.Account, *TokenPair, error) {
	acc, err := s.accountRepo.GetAccountByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrAccountNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("get account: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	tokenPair, err := s.generateTokenPair(ctx, acc.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	return acc, tokenPair, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	if err := s.tokenRepo.DeleteRefreshToken(ctx, refreshToken); err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}

	return nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	userID, err := s.tokenRepo.GetUserIDByRefreshToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, repository.ErrTokenNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("get user id by refresh token: %w", err)
	}

	// Delete old token (rotation)
	if err := s.tokenRepo.DeleteRefreshToken(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("delete old refresh token: %w", err)
	}

	tokenPair, err := s.generateTokenPair(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	return tokenPair, nil
}

func (s *authService) ValidateToken(ctx context.Context, accessToken string) (uuid.UUID, error) {
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return uuid.Nil, ErrExpiredToken
		}
		return uuid.Nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		sub, ok := claims["sub"].(string)
		if !ok {
			return uuid.Nil, ErrInvalidToken
		}

		userID, err := uuid.Parse(sub)
		if err != nil {
			return uuid.Nil, ErrInvalidToken
		}

		return userID, nil
	}

	return uuid.Nil, ErrInvalidToken
}

func (s *authService) generateTokenPair(ctx context.Context, userID uuid.UUID) (*TokenPair, error) {
	// Access Token
	accessTokenClaims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(s.accessTTL).Unix(),
		"iat": time.Now().Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)
	at, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	// Refresh Token
	rt := uuid.New().String()
	if err := s.tokenRepo.SaveRefreshToken(ctx, userID, rt, s.refreshTTL); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  at,
		RefreshToken: rt,
	}, nil
}
