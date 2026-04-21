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

func newTestHandler(t *testing.T) (*mocks.AuthServiceClient, *auth.Handler) {
	t.Helper()
	client := mocks.NewAuthServiceClient(t)
	return client, auth.NewHandler(client, false)
}

func newPublicRouter(t *testing.T) (*mocks.AuthServiceClient, chi.Router) {
	t.Helper()
	client, h := newTestHandler(t)
	r := chi.NewRouter()
	h.RegisterPublic(r)
	return client, r
}

func newSessionRouter(t *testing.T) (*mocks.AuthServiceClient, chi.Router) {
	t.Helper()
	client, h := newTestHandler(t)
	r := chi.NewRouter()
	h.RegisterSession(r)
	return client, r
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestHandler_Register_Success(t *testing.T) {
	t.Parallel()
	client, r := newPublicRouter(t)

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

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, userID, resp["user_id"])
	assert.Empty(t, resp["access_token"])
	assert.Empty(t, resp["refresh_token"])

	cookies := w.Result().Cookies()
	accessCookie := findCookie(cookies, "access_token")
	refreshCookie := findCookie(cookies, "refresh_token")
	require.NotNil(t, accessCookie)
	require.NotNil(t, refreshCookie)
	assert.Equal(t, "access-token", accessCookie.Value)
	assert.Equal(t, "refresh-token", refreshCookie.Value)
	assert.True(t, accessCookie.HttpOnly)
	assert.True(t, refreshCookie.HttpOnly)
}

func TestHandler_Register_InvalidBody(t *testing.T) {
	t.Parallel()
	_, r := newPublicRouter(t)

	body := `{invalid json}`
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_GrpcError(t *testing.T) {
	t.Parallel()
	client, r := newPublicRouter(t)

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
	client, r := newPublicRouter(t)

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

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, userID, resp["user_id"])
	assert.Empty(t, resp["access_token"])
	assert.Empty(t, resp["refresh_token"])

	cookies := w.Result().Cookies()
	accessCookie := findCookie(cookies, "access_token")
	refreshCookie := findCookie(cookies, "refresh_token")
	require.NotNil(t, accessCookie)
	require.NotNil(t, refreshCookie)
	assert.Equal(t, "access-token", accessCookie.Value)
	assert.Equal(t, "refresh-token", refreshCookie.Value)
}

func TestHandler_Login_InvalidCredentials(t *testing.T) {
	t.Parallel()
	client, r := newPublicRouter(t)

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
	client, r := newSessionRouter(t)

	client.On("Logout", mock.Anything, mock.Anything).Return(&authv1.LogoutResponse{}, nil)

	req := httptest.NewRequest("POST", "/auth/session/logout", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "some-refresh-token"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	cookies := w.Result().Cookies()
	accessCookie := findCookie(cookies, "access_token")
	refreshCookie := findCookie(cookies, "refresh_token")
	require.NotNil(t, accessCookie)
	require.NotNil(t, refreshCookie)
	assert.Equal(t, -1, accessCookie.MaxAge)
	assert.Equal(t, -1, refreshCookie.MaxAge)
}

func TestHandler_Logout_NoCookie(t *testing.T) {
	t.Parallel()
	_, r := newSessionRouter(t)

	req := httptest.NewRequest("POST", "/auth/session/logout", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_Logout_GrpcError(t *testing.T) {
	t.Parallel()
	client, r := newSessionRouter(t)

	client.On("Logout", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Internal, "internal error"))

	req := httptest.NewRequest("POST", "/auth/session/logout", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "some-token"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_Refresh_Success(t *testing.T) {
	t.Parallel()
	client, r := newSessionRouter(t)

	client.On("RefreshToken", mock.Anything, mock.Anything).Return(&authv1.RefreshTokenResponse{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
	}, nil)

	req := httptest.NewRequest("POST", "/auth/session/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "old-refresh-token"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	cookies := w.Result().Cookies()
	accessCookie := findCookie(cookies, "access_token")
	refreshCookie := findCookie(cookies, "refresh_token")
	require.NotNil(t, accessCookie)
	require.NotNil(t, refreshCookie)
	assert.Equal(t, "new-access-token", accessCookie.Value)
	assert.Equal(t, "new-refresh-token", refreshCookie.Value)
}

func TestHandler_Refresh_NoCookie(t *testing.T) {
	t.Parallel()
	_, r := newSessionRouter(t)

	req := httptest.NewRequest("POST", "/auth/session/refresh", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Refresh_ExpiredToken(t *testing.T) {
	t.Parallel()
	client, r := newSessionRouter(t)

	client.On("RefreshToken", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Unauthenticated, "token expired"))

	req := httptest.NewRequest("POST", "/auth/session/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "expired-token"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
