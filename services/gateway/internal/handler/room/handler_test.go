package room_test

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

	"github.com/sudobytemebaby/efir/services/gateway/internal/handler/room"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler/room/mocks"
	"github.com/sudobytemebaby/efir/services/gateway/internal/middleware"
	"github.com/sudobytemebaby/efir/services/gateway/internal/testutil"
	roomv1 "github.com/sudobytemebaby/efir/services/shared/gen/room"
)

// newRouter returns a chi router with JWT auth middleware and the room handler registered.
func newRouter(t *testing.T) (*mocks.RoomServiceClient, chi.Router) {
	t.Helper()
	client := mocks.NewRoomServiceClient(t)
	h := room.NewHandler(client)
	r := chi.NewRouter()
	r.Use(middleware.JWTAuth(testutil.TestSecret))
	h.Register(r)
	return client, r
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

func TestCreateRoom_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	roomID := uuid.New().String()

	client, r := newRouter(t)
	client.On("CreateRoom", mock.Anything, mock.MatchedBy(func(req *roomv1.CreateRoomRequest) bool {
		return req.Name == "test-room" && req.CreatedBy == userID
	})).Return(&roomv1.CreateRoomResponse{
		Room: &roomv1.Room{RoomId: roomID, Name: "test-room"},
	}, nil)

	body := jsonBody(t, map[string]string{"name": "test-room"})
	req := httptest.NewRequest(http.MethodPost, "/rooms", body)
	testutil.SetAccessCookie(req, userID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, roomID, got["room_id"])
}

func TestCreateRoom_Unauthenticated(t *testing.T) {
	t.Parallel()
	_, r := newRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/rooms", jsonBody(t, map[string]string{"name": "x"}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetRoom_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	roomID := uuid.New().String()

	client, r := newRouter(t)
	client.On("GetRoom", mock.Anything, &roomv1.GetRoomRequest{RoomId: roomID}).
		Return(&roomv1.GetRoomResponse{
			Room: &roomv1.Room{RoomId: roomID, Name: "my-room"},
		}, nil)

	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID, nil)
	testutil.SetAccessCookie(req, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "my-room", got["name"])
}

func TestGetRoom_NotFound(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	roomID := uuid.New().String()

	client, r := newRouter(t)
	client.On("GetRoom", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.NotFound, "room not found"))

	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID, nil)
	testutil.SetAccessCookie(req, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateRoom_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	roomID := uuid.New().String()

	client, r := newRouter(t)
	client.On("UpdateRoom", mock.Anything, mock.MatchedBy(func(req *roomv1.UpdateRoomRequest) bool {
		return req.RoomId == roomID && req.RequesterId == userID && req.GetName() == "New Name"
	})).Return(&roomv1.UpdateRoomResponse{
		Room: &roomv1.Room{RoomId: roomID, Name: "New Name"},
	}, nil)

	body := jsonBody(t, map[string]string{"name": "New Name"})
	req := httptest.NewRequest(http.MethodPatch, "/rooms/"+roomID, body)
	testutil.SetAccessCookie(req, userID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateRoom_InvalidBody(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	roomID := uuid.New().String()

	_, r := newRouter(t)

	req := httptest.NewRequest(http.MethodPatch, "/rooms/"+roomID, bytes.NewBufferString(`{invalid`))
	testutil.SetAccessCookie(req, userID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateRoom_NotOwner(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	roomID := uuid.New().String()

	client, r := newRouter(t)
	client.On("UpdateRoom", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.PermissionDenied, "not owner"))

	body := jsonBody(t, map[string]string{"name": "New"})
	req := httptest.NewRequest(http.MethodPatch, "/rooms/"+roomID, body)
	testutil.SetAccessCookie(req, userID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdateRoom_NotFound(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	roomID := uuid.New().String()

	client, r := newRouter(t)
	client.On("UpdateRoom", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.NotFound, "room not found"))

	body := jsonBody(t, map[string]string{"name": "New"})
	req := httptest.NewRequest(http.MethodPatch, "/rooms/"+roomID, body)
	testutil.SetAccessCookie(req, userID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteRoom_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	roomID := uuid.New().String()

	client, r := newRouter(t)
	client.On("DeleteRoom", mock.Anything, mock.MatchedBy(func(req *roomv1.DeleteRoomRequest) bool {
		return req.RoomId == roomID && req.RequesterId == userID
	})).Return(&roomv1.DeleteRoomResponse{}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/rooms/"+roomID, nil)
	testutil.SetAccessCookie(req, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteRoom_PermissionDenied(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	roomID := uuid.New().String()

	client, r := newRouter(t)
	client.On("DeleteRoom", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.PermissionDenied, "not owner"))

	req := httptest.NewRequest(http.MethodDelete, "/rooms/"+roomID, nil)
	testutil.SetAccessCookie(req, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAddMember_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	memberID := uuid.New().String()
	roomID := uuid.New().String()

	client, r := newRouter(t)
	client.On("AddMember", mock.Anything, mock.MatchedBy(func(req *roomv1.AddMemberRequest) bool {
		return req.RoomId == roomID && req.UserId == memberID && req.RequesterId == userID
	})).Return(&roomv1.AddMemberResponse{}, nil)

	body := jsonBody(t, map[string]string{"user_id": memberID})
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID+"/members", body)
	testutil.SetAccessCookie(req, userID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestRemoveMember_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	memberID := uuid.New().String()
	roomID := uuid.New().String()

	client, r := newRouter(t)
	client.On("RemoveMember", mock.Anything, mock.MatchedBy(func(req *roomv1.RemoveMemberRequest) bool {
		return req.RoomId == roomID && req.UserId == memberID && req.RequesterId == userID
	})).Return(&roomv1.RemoveMemberResponse{}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/rooms/"+roomID+"/members/"+memberID, nil)
	testutil.SetAccessCookie(req, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}
