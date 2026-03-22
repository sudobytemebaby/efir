package handler

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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockAuthClient struct {
	registerFunc     func(ctx context.Context, email, password string) (*authv1.RegisterResponse, error)
	loginFunc        func(ctx context.Context, email, password string) (*authv1.LoginResponse, error)
	logoutFunc       func(ctx context.Context, refreshToken string) (*authv1.LogoutResponse, error)
	refreshTokenFunc func(ctx context.Context, refreshToken string) (*authv1.RefreshTokenResponse, error)
}

func (m *mockAuthClient) Register(ctx context.Context, email, password string) (*authv1.RegisterResponse, error) {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, email, password)
	}
	return nil, errors.New("not implemented")
}

func (m *mockAuthClient) Login(ctx context.Context, email, password string) (*authv1.LoginResponse, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, email, password)
	}
	return nil, errors.New("not implemented")
}

func (m *mockAuthClient) Logout(ctx context.Context, refreshToken string) (*authv1.LogoutResponse, error) {
	if m.logoutFunc != nil {
		return m.logoutFunc(ctx, refreshToken)
	}
	return nil, errors.New("not implemented")
}

func (m *mockAuthClient) RefreshToken(ctx context.Context, refreshToken string) (*authv1.RefreshTokenResponse, error) {
	if m.refreshTokenFunc != nil {
		return m.refreshTokenFunc(ctx, refreshToken)
	}
	return nil, errors.New("not implemented")
}

type testHTTPHandler struct {
	*HTTPHandler
	mockClient *mockAuthClient
}

func newTestHTTPHandler() *testHTTPHandler {
	mockClient := &mockAuthClient{}
	h := &HTTPHandler{authClient: mockClient}
	return &testHTTPHandler{
		HTTPHandler: h,
		mockClient:  mockClient,
	}
}

func TestHTTPHandler_Register_Success(t *testing.T) {
	h := newTestHTTPHandler()
	r := chi.NewRouter()
	h.Register(r)

	userID := uuid.New().String()
	h.mockClient.registerFunc = func(ctx context.Context, email, password string) (*authv1.RegisterResponse, error) {
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

	var resp authResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, userID, resp.UserID)
	assert.Equal(t, "access-token", resp.AccessToken)
	assert.Equal(t, "refresh-token", resp.RefreshToken)
}

func TestHTTPHandler_Register_InvalidBody(t *testing.T) {
	h := newTestHTTPHandler()
	r := chi.NewRouter()
	h.Register(r)

	body := `{invalid json}`
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHTTPHandler_Register_GrpcError(t *testing.T) {
	h := newTestHTTPHandler()
	r := chi.NewRouter()
	h.Register(r)

	h.mockClient.registerFunc = func(ctx context.Context, email, password string) (*authv1.RegisterResponse, error) {
		return nil, status.Error(codes.AlreadyExists, "user already exists")
	}

	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHTTPHandler_Login_Success(t *testing.T) {
	h := newTestHTTPHandler()
	r := chi.NewRouter()
	h.Register(r)

	userID := uuid.New().String()
	h.mockClient.loginFunc = func(ctx context.Context, email, password string) (*authv1.LoginResponse, error) {
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

	var resp authResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, userID, resp.UserID)
	assert.Equal(t, "access-token", resp.AccessToken)
	assert.Equal(t, "refresh-token", resp.RefreshToken)
}

func TestHTTPHandler_Login_InvalidCredentials(t *testing.T) {
	h := newTestHTTPHandler()
	r := chi.NewRouter()
	h.Register(r)

	h.mockClient.loginFunc = func(ctx context.Context, email, password string) (*authv1.LoginResponse, error) {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	body := `{"email":"test@example.com","password":"wrongpassword"}`
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHTTPHandler_Logout_Success(t *testing.T) {
	h := newTestHTTPHandler()
	r := chi.NewRouter()
	h.Register(r)

	h.mockClient.logoutFunc = func(ctx context.Context, refreshToken string) (*authv1.LogoutResponse, error) {
		return &authv1.LogoutResponse{}, nil
	}

	body := `{"refresh_token":"some-refresh-token"}`
	req := httptest.NewRequest("POST", "/auth/logout", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHTTPHandler_Refresh_Success(t *testing.T) {
	h := newTestHTTPHandler()
	r := chi.NewRouter()
	h.Register(r)

	h.mockClient.refreshTokenFunc = func(ctx context.Context, refreshToken string) (*authv1.RefreshTokenResponse, error) {
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

	var resp authResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", resp.AccessToken)
	assert.Equal(t, "new-refresh-token", resp.RefreshToken)
	assert.Empty(t, resp.UserID)
}

func TestHTTPHandler_Refresh_ExpiredToken(t *testing.T) {
	h := newTestHTTPHandler()
	r := chi.NewRouter()
	h.Register(r)

	h.mockClient.refreshTokenFunc = func(ctx context.Context, refreshToken string) (*authv1.RefreshTokenResponse, error) {
		return nil, status.Error(codes.Unauthenticated, "token expired")
	}

	body := `{"refresh_token":"expired-token"}`
	req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
