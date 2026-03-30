//go:build integration

package wsauth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudobytemebaby/efir/services/gateway/internal/handler/wsauth"
	"github.com/sudobytemebaby/efir/services/gateway/internal/middleware"
	"github.com/sudobytemebaby/efir/services/gateway/internal/testutil"
	sharedtestutil "github.com/sudobytemebaby/efir/services/shared/pkg/testutil"
)

var valkeyContainer *sharedtestutil.ValkeyContainer

func TestMain(m *testing.M) {
	ctx := context.Background()
	valkeyContainer = sharedtestutil.NewValkeyContainer(ctx)
	exitCode := m.Run()
	_ = valkeyContainer.Terminate(ctx)
	os.Exit(exitCode)
}

func newRouter(t *testing.T) chi.Router {
	t.Helper()
	client := valkeyContainer.Client(t)
	h := wsauth.NewHandler(client, 5*time.Minute)
	r := chi.NewRouter()
	r.With(middleware.JWTAuth(testutil.TestSecret)).Post("/ws/ticket", h.CreateTicket)
	h.Register(r)
	return r
}

func TestCreateTicket_Success(t *testing.T) {
	userID := uuid.New().String()
	r := newRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/ws/ticket", nil)
	req.Header.Set("Authorization", testutil.AuthHeader(userID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["ticket"])
}

func TestCreateTicket_Unauthenticated(t *testing.T) {
	r := newRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/ws/ticket", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestValidateTicket_Success(t *testing.T) {
	userID := uuid.New().String()
	r := newRouter(t)

	// Step 1: create a ticket.
	createReq := httptest.NewRequest(http.MethodPost, "/ws/ticket", nil)
	createReq.Header.Set("Authorization", testutil.AuthHeader(userID))
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var createResp map[string]string
	require.NoError(t, json.Unmarshal(createW.Body.Bytes(), &createResp))
	ticket := createResp["ticket"]
	require.NotEmpty(t, ticket)

	// Step 2: validate the ticket.
	validateReq := httptest.NewRequest(http.MethodGet, "/auth/validate", nil)
	validateReq.Header.Set("X-Ws-Ticket", ticket)
	validateW := httptest.NewRecorder()
	r.ServeHTTP(validateW, validateReq)

	require.Equal(t, http.StatusOK, validateW.Code)
	assert.Equal(t, userID, validateW.Header().Get("X-User-Id"))

	var validateResp map[string]string
	require.NoError(t, json.Unmarshal(validateW.Body.Bytes(), &validateResp))
	assert.Equal(t, userID, validateResp["user_id"])
}

func TestValidateTicket_TicketIsConsumedOnce(t *testing.T) {
	userID := uuid.New().String()
	r := newRouter(t)

	// Create ticket.
	createReq := httptest.NewRequest(http.MethodPost, "/ws/ticket", nil)
	createReq.Header.Set("Authorization", testutil.AuthHeader(userID))
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	require.Equal(t, http.StatusCreated, createW.Code)

	var createResp map[string]string
	require.NoError(t, json.Unmarshal(createW.Body.Bytes(), &createResp))
	ticket := createResp["ticket"]

	// First validation succeeds.
	req1 := httptest.NewRequest(http.MethodGet, "/auth/validate", nil)
	req1.Header.Set("X-Ws-Ticket", ticket)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second validation fails — GETDEL deleted the key.
	req2 := httptest.NewRequest(http.MethodGet, "/auth/validate", nil)
	req2.Header.Set("X-Ws-Ticket", ticket)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestCreateTicket_InvalidUserID(t *testing.T) {
	r := newRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/ws/ticket", nil)
	req.Header.Set("Authorization", testutil.AuthHeader("not-a-uuid"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestValidateTicket_MissingTicket(t *testing.T) {
	r := newRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/auth/validate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
