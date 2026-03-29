package user_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sudobytemebaby/efir/services/gateway/internal/handler/user"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler/user/mocks"
	"github.com/sudobytemebaby/efir/services/gateway/internal/middleware"
	userv1 "github.com/sudobytemebaby/efir/services/shared/gen/user"
)

const testSecret = "test-secret"

func authHeader(userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte(testSecret))
	return "Bearer " + signed
}

func newRouter(t *testing.T) (*mocks.UserServiceClient, chi.Router) {
	t.Helper()
	client := mocks.NewUserServiceClient(t)
	h := user.NewHandler(client)
	r := chi.NewRouter()
	r.Use(middleware.JWTAuth(testSecret))
	h.Register(r)
	return client, r
}

func TestGetMe_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()

	client, r := newRouter(t)
	client.On("GetUser", mock.Anything, &userv1.GetUserRequest{UserId: userID}).
		Return(&userv1.GetUserResponse{
			User: &userv1.User{UserId: userID, DisplayName: "Test User"},
		}, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("Authorization", authHeader(userID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "Test User", got["display_name"])
}

func TestGetMe_Unauthenticated(t *testing.T) {
	t.Parallel()
	_, r := newRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetByID_Success(t *testing.T) {
	t.Parallel()
	callerID := uuid.New().String()
	targetID := uuid.New().String()

	client, r := newRouter(t)
	client.On("GetUser", mock.Anything, &userv1.GetUserRequest{UserId: targetID}).
		Return(&userv1.GetUserResponse{
			User: &userv1.User{UserId: targetID, DisplayName: "Other User"},
		}, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/"+targetID, nil)
	req.Header.Set("Authorization", authHeader(callerID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetByID_NotFound(t *testing.T) {
	t.Parallel()
	callerID := uuid.New().String()
	targetID := uuid.New().String()

	client, r := newRouter(t)
	client.On("GetUser", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.NotFound, "user not found"))

	req := httptest.NewRequest(http.MethodGet, "/users/"+targetID, nil)
	req.Header.Set("Authorization", authHeader(callerID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
