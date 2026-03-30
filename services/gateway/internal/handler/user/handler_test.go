package user_test

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

	"github.com/sudobytemebaby/efir/services/gateway/internal/handler/user"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler/user/mocks"
	"github.com/sudobytemebaby/efir/services/gateway/internal/middleware"
	"github.com/sudobytemebaby/efir/services/gateway/internal/testutil"
	userv1 "github.com/sudobytemebaby/efir/services/shared/gen/user"
)

func newRouter(t *testing.T) (*mocks.UserServiceClient, chi.Router) {
	t.Helper()
	client := mocks.NewUserServiceClient(t)
	h := user.NewHandler(client)
	r := chi.NewRouter()
	r.Use(middleware.JWTAuth(testutil.TestSecret))
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
	req.Header.Set("Authorization", testutil.AuthHeader(userID))
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
	req.Header.Set("Authorization", testutil.AuthHeader(callerID))
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
	req.Header.Set("Authorization", testutil.AuthHeader(callerID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateMe_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()

	client, r := newRouter(t)
	client.On("UpdateUser", mock.Anything, mock.MatchedBy(func(req *userv1.UpdateUserRequest) bool {
		return req.UserId == userID && req.GetDisplayName() == "New Name"
	})).Return(&userv1.UpdateUserResponse{
		User: &userv1.User{UserId: userID, DisplayName: "New Name"},
	}, nil)

	body := bytes.NewBufferString(`{"display_name":"New Name"}`)
	req := httptest.NewRequest(http.MethodPatch, "/users/me", body)
	req.Header.Set("Authorization", testutil.AuthHeader(userID))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "New Name", got["display_name"])
}

func TestUpdateMe_InvalidBody(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()

	_, r := newRouter(t)

	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Authorization", testutil.AuthHeader(userID))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateMe_GrpcError(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()

	client, r := newRouter(t)
	client.On("UpdateUser", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.Internal, "internal error"))

	body := bytes.NewBufferString(`{"display_name":"New"}`)
	req := httptest.NewRequest(http.MethodPatch, "/users/me", body)
	req.Header.Set("Authorization", testutil.AuthHeader(userID))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateMe_Unauthenticated(t *testing.T) {
	t.Parallel()
	_, r := newRouter(t)

	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewBufferString(`{"display_name":"New"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
