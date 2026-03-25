package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authv1 "github.com/sudobytemebaby/efir/services/shared/gen/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockAuthClient struct {
	registerFunc     func(ctx context.Context, in *authv1.RegisterRequest, opts ...grpc.CallOption) (*authv1.RegisterResponse, error)
	loginFunc        func(ctx context.Context, in *authv1.LoginRequest, opts ...grpc.CallOption) (*authv1.LoginResponse, error)
	logoutFunc       func(ctx context.Context, in *authv1.LogoutRequest, opts ...grpc.CallOption) (*authv1.LogoutResponse, error)
	refreshTokenFunc func(ctx context.Context, in *authv1.RefreshTokenRequest, opts ...grpc.CallOption) (*authv1.RefreshTokenResponse, error)
}

func (m *mockAuthClient) Register(ctx context.Context, in *authv1.RegisterRequest, opts ...grpc.CallOption) (*authv1.RegisterResponse, error) {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, in, opts...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockAuthClient) Login(ctx context.Context, in *authv1.LoginRequest, opts ...grpc.CallOption) (*authv1.LoginResponse, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, in, opts...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockAuthClient) Logout(ctx context.Context, in *authv1.LogoutRequest, opts ...grpc.CallOption) (*authv1.LogoutResponse, error) {
	if m.logoutFunc != nil {
		return m.logoutFunc(ctx, in, opts...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockAuthClient) RefreshToken(ctx context.Context, in *authv1.RefreshTokenRequest, opts ...grpc.CallOption) (*authv1.RefreshTokenResponse, error) {
	if m.refreshTokenFunc != nil {
		return m.refreshTokenFunc(ctx, in, opts...)
	}
	return nil, errors.New("not implemented")
}

// authFields mirrors the proto JSON response fields for test assertions.
type authFields struct {
	UserID       string `json:"user_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type testHandler struct {
	*Handler
	mockClient *mockAuthClient
}

func newTestHandler() *testHandler {
	mockClient := &mockAuthClient{}
	h := &Handler{client: mockClient}
	return &testHandler{Handler: h, mockClient: mockClient}
}

func TestHandler_Register_Success(t *testing.T) {
	h := newTestHandler()
	r := chi.NewRouter()
	h.Register(r)

	userID := uuid.New().String()
	h.mockClient.registerFunc = func(_ context.Context, in *authv1.RegisterRequest, _ ...grpc.CallOption) (*authv1.RegisterResponse, error) {
		return &authv1.RegisterResponse{
			UserId:       userID,
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
		}, nil
	}

	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authFields
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, userID, resp.UserID)
	assert.Equal(t, "access-token", resp.AccessToken)
	assert.Equal(t, "refresh-token", resp.RefreshToken)
}

func TestHandler_Register_InvalidBody(t *testing.T) {
	h := newTestHandler()
	r := chi.NewRouter()
	h.Register(r)

	body := `{invalid json}`
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_GrpcError(t *testing.T) {
	h := newTestHandler()
	r := chi.NewRouter()
	h.Register(r)

	h.mockClient.registerFunc = func(_ context.Context, _ *authv1.RegisterRequest, _ ...grpc.CallOption) (*authv1.RegisterResponse, error) {
		return nil, status.Error(codes.AlreadyExists, "user already exists")
	}

	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_Login_Success(t *testing.T) {
	h := newTestHandler()
	r := chi.NewRouter()
	h.Register(r)

	userID := uuid.New().String()
	h.mockClient.loginFunc = func(_ context.Context, _ *authv1.LoginRequest, _ ...grpc.CallOption) (*authv1.LoginResponse, error) {
		return &authv1.LoginResponse{
			UserId:       userID,
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
		}, nil
	}

	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authFields
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, userID, resp.UserID)
	assert.Equal(t, "access-token", resp.AccessToken)
	assert.Equal(t, "refresh-token", resp.RefreshToken)
}

func TestHandler_Login_InvalidCredentials(t *testing.T) {
	h := newTestHandler()
	r := chi.NewRouter()
	h.Register(r)

	h.mockClient.loginFunc = func(_ context.Context, _ *authv1.LoginRequest, _ ...grpc.CallOption) (*authv1.LoginResponse, error) {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	body := `{"email":"test@example.com","password":"wrongpassword"}`
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Logout_Success(t *testing.T) {
	h := newTestHandler()
	r := chi.NewRouter()
	h.Register(r)

	h.mockClient.logoutFunc = func(_ context.Context, _ *authv1.LogoutRequest, _ ...grpc.CallOption) (*authv1.LogoutResponse, error) {
		return &authv1.LogoutResponse{}, nil
	}

	body := `{"refresh_token":"some-refresh-token"}`
	req := httptest.NewRequest("POST", "/auth/logout", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_Refresh_Success(t *testing.T) {
	h := newTestHandler()
	r := chi.NewRouter()
	h.Register(r)

	h.mockClient.refreshTokenFunc = func(_ context.Context, _ *authv1.RefreshTokenRequest, _ ...grpc.CallOption) (*authv1.RefreshTokenResponse, error) {
		return &authv1.RefreshTokenResponse{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
		}, nil
	}

	body := `{"refresh_token":"old-refresh-token"}`
	req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authFields
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "new-access-token", resp.AccessToken)
	assert.Equal(t, "new-refresh-token", resp.RefreshToken)
	assert.Empty(t, resp.UserID)
}

func TestHandler_Refresh_ExpiredToken(t *testing.T) {
	h := newTestHandler()
	r := chi.NewRouter()
	h.Register(r)

	h.mockClient.refreshTokenFunc = func(_ context.Context, _ *authv1.RefreshTokenRequest, _ ...grpc.CallOption) (*authv1.RefreshTokenResponse, error) {
		return nil, status.Error(codes.Unauthenticated, "token expired")
	}

	body := `{"refresh_token":"expired-token"}`
	req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
