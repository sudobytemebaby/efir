package message_test

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

	"github.com/sudobytemebaby/efir/services/gateway/internal/handler/message"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler/message/mocks"
	"github.com/sudobytemebaby/efir/services/gateway/internal/middleware"
	"github.com/sudobytemebaby/efir/services/gateway/internal/testutil"
	messagev1 "github.com/sudobytemebaby/efir/services/shared/gen/message"
)

func newRouter(t *testing.T) (*mocks.MessageServiceClient, chi.Router) {
	t.Helper()
	client := mocks.NewMessageServiceClient(t)
	h := message.NewHandler(client)
	r := chi.NewRouter()
	r.Use(middleware.JWTAuth(testutil.TestSecret))
	h.Register(r)
	return client, r
}

func TestSendMessage_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	roomID := uuid.New().String()
	msgID := uuid.New().String()

	client, r := newRouter(t)
	client.On("SendMessage", mock.Anything, mock.MatchedBy(func(req *messagev1.SendMessageRequest) bool {
		return req.RoomId == roomID && req.SenderId == userID
	})).Return(&messagev1.SendMessageResponse{
		Message: &messagev1.Message{MessageId: msgID, RoomId: roomID},
	}, nil)

	body, _ := json.Marshal(map[string]any{"type": "TEXT", "text": map[string]string{"text": "hello"}})
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID+"/messages", bytes.NewBuffer(body))
	testutil.SetAccessCookie(req, userID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, msgID, got["message_id"])
}

func TestSendMessage_PermissionDenied(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	roomID := uuid.New().String()

	client, r := newRouter(t)
	client.On("SendMessage", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.PermissionDenied, "not a member"))

	body, _ := json.Marshal(map[string]any{"type": "TEXT"})
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID+"/messages", bytes.NewBuffer(body))
	testutil.SetAccessCookie(req, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetMessages_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	roomID := uuid.New().String()

	client, r := newRouter(t)
	client.On("GetMessages", mock.Anything, mock.MatchedBy(func(req *messagev1.GetMessagesRequest) bool {
		return req.RoomId == roomID && req.RequesterId == userID && req.Limit == 50
	})).Return(&messagev1.GetMessagesResponse{
		Messages: []*messagev1.Message{
			{MessageId: uuid.New().String(), RoomId: roomID},
		},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID+"/messages", nil)
	testutil.SetAccessCookie(req, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetMessages_WithCursorAndLimit(t *testing.T) {
	t.Parallel()
	userID := uuid.New().String()
	roomID := uuid.New().String()
	cursor := uuid.New().String()

	client, r := newRouter(t)
	client.On("GetMessages", mock.Anything, mock.MatchedBy(func(req *messagev1.GetMessagesRequest) bool {
		return req.Limit == 10 && req.Cursor != nil && *req.Cursor == cursor
	})).Return(&messagev1.GetMessagesResponse{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID+"/messages?limit=10&cursor="+cursor, nil)
	testutil.SetAccessCookie(req, userID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
