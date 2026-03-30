package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sudobytemebaby/efir/services/gateway/internal/handler/auth"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler/auth/mocks"
	authv1 "github.com/sudobytemebaby/efir/services/shared/gen/auth"
)

type authFields struct {
	UserID       string `json:"user_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func newTestHandler(t *testing.T) (*mocks.AuthServiceClient, *auth.Handler) {
	t.Helper()
	client := mocks.NewAuthServiceClient(t)
	return client, auth.NewHandler(client)
}

func TestHandler_Register_Success(t *testing.T) {
	t.Parallel()
	client, h := newTestHandler(t)
	r := chi.NewRouter()
	h.Register(r)

	userID := uuid.New().String()
	client.On("Register", mock.Anything, mock.Anything).Return(&authv1.RegisterResponse{
		UserId:       userID,
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}, nil)

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
	t.Parallel()
	_, h := newTestHandler(t)
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
	t.Parallel()
	client, h := newTestHandler(t)
	r := chi.NewRouter()
	h.Register(r)

	client.On("Register", mock.Anything, mock.Anything).Return(nil, status.Error(codes.AlreadyExists, "user already exists"))

	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_Login_Success(t *testing.T) {
	t.Parallel()
	client, h := newTestHandler(t)
	r := chi.NewRouter()
	h.Register(r)

	userID := uuid.New().String()
	client.On("Login", mock.Anything, mock.Anything).Return(&authv1.LoginResponse{
		UserId:       userID,
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}, nil)

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
	t.Parallel()
	client, h := newTestHandler(t)
	r := chi.NewRouter()
	h.Register(r)

	client.On("Login", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Unauthenticated, "invalid credentials"))

	body := `{"email":"test@example.com","password":"wrongpassword"}`
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Logout_Success(t *testing.T) {
	t.Parallel()
	client, h := newTestHandler(t)
	r := chi.NewRouter()
	h.Register(r)

	client.On("Logout", mock.Anything, mock.Anything).Return(&authv1.LogoutResponse{}, nil)

	body := `{"refresh_token":"some-refresh-token"}`
	req := httptest.NewRequest("POST", "/auth/logout", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_Logout_InvalidBody(t *testing.T) {
	t.Parallel()
	_, h := newTestHandler(t)
	r := chi.NewRouter()
	h.Register(r)

	req := httptest.NewRequest("POST", "/auth/logout", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Logout_GrpcError(t *testing.T) {
	t.Parallel()
	client, h := newTestHandler(t)
	r := chi.NewRouter()
	h.Register(r)

	client.On("Logout", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Internal, "internal error"))

	body := `{"refresh_token":"some-token"}`
	req := httptest.NewRequest("POST", "/auth/logout", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_Refresh_Success(t *testing.T) {
	t.Parallel()
	client, h := newTestHandler(t)
	r := chi.NewRouter()
	h.Register(r)

	client.On("RefreshToken", mock.Anything, mock.Anything).Return(&authv1.RefreshTokenResponse{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
	}, nil)

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
	t.Parallel()
	client, h := newTestHandler(t)
	r := chi.NewRouter()
	h.Register(r)

	client.On("RefreshToken", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Unauthenticated, "token expired"))

	body := `{"refresh_token":"expired-token"}`
	req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
