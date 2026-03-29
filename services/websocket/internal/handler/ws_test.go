//go:build integration

package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vk "github.com/valkey-io/valkey-go"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"github.com/sudobytemebaby/efir/services/shared/pkg/testutil"
	"github.com/sudobytemebaby/efir/services/shared/pkg/valkey"
	"github.com/sudobytemebaby/efir/services/websocket/internal/config"
	"github.com/sudobytemebaby/efir/services/websocket/internal/handler"
	"github.com/sudobytemebaby/efir/services/websocket/internal/hub"
)

var valkeyContainer *testutil.ValkeyContainer

func TestMain(m *testing.M) {
	ctx := context.Background()
	valkeyContainer = testutil.NewValkeyContainer(ctx)
	exitCode := m.Run()
	_ = valkeyContainer.Terminate(ctx)
	os.Exit(exitCode)
}

func defaultConfig() *config.Config {
	cfg := &config.Config{}
	cfg.WebSocket.WriteDeadline = 10 * time.Second
	cfg.WebSocket.ReadDeadline = 10 * time.Second
	cfg.WebSocket.PingInterval = 30 * time.Second
	cfg.WebSocket.ReadLimit = 512 * 1024
	return cfg
}

func storeTicket(t *testing.T, client vk.Client, userID string) string {
	t.Helper()
	ticket := testutil.RandomUUID().String()
	key := valkey.GatewayWSTicketKey(ticket)
	err := client.Do(context.Background(),
		client.B().Set().Key(key).Value(userID).Ex(5*time.Minute).Build(),
	).Error()
	require.NoError(t, err)
	return ticket
}

func newTestServer(t *testing.T, h *hub.Hub, client vk.Client) *httptest.Server {
	t.Helper()
	wsHandler := handler.NewWebSocketHandler(h, "", client, defaultConfig())
	srv := httptest.NewServer(http.HandlerFunc(wsHandler.HandleWS))
	t.Cleanup(srv.Close)
	return srv
}

func wsURL(srv *httptest.Server, ticket, roomID string) string {
	u := "ws" + srv.URL[4:] + "/ws?ticket=" + ticket
	if roomID != "" {
		u += "&room_id=" + roomID
	}
	return u
}

func TestHandleWS_MissingTicket(t *testing.T) {
	client := valkeyContainer.Client(t)
	h := hub.NewHub(64)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	srv := newTestServer(t, h, client)

	resp, err := http.Get(srv.URL + "/ws")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestHandleWS_InvalidTicket(t *testing.T) {
	client := valkeyContainer.Client(t)
	h := hub.NewHub(64)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	srv := newTestServer(t, h, client)

	// Ticket does not exist in Valkey — use plain HTTP, server rejects before upgrade.
	httpURL := srv.URL + "/ws?ticket=" + testutil.RandomUUID().String()
	resp, err := http.Get(httpURL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestHandleWS_Connect_PingPong(t *testing.T) {
	client := valkeyContainer.Client(t)
	h := hub.NewHub(64)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go h.Run(ctx)

	srv := newTestServer(t, h, client)
	userID := testutil.RandomUUID().String()
	ticket := storeTicket(t, client, userID)

	conn, _, err := websocket.Dial(ctx, wsURL(srv, ticket, ""), nil)
	require.NoError(t, err)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send a ping envelope and expect a pong back.
	pingEnv := hub.Envelope{Type: hub.TypePing, Payload: json.RawMessage(`{}`)}
	require.NoError(t, wsjson.Write(ctx, conn, pingEnv))

	var pongEnv hub.Envelope
	require.NoError(t, wsjson.Read(ctx, conn, &pongEnv))
	assert.Equal(t, hub.TypePong, pongEnv.Type)
}

func TestHandleWS_SubscribeAndReceiveBroadcast(t *testing.T) {
	client := valkeyContainer.Client(t)
	h := hub.NewHub(64)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go h.Run(ctx)

	srv := newTestServer(t, h, client)
	userID := testutil.RandomUUID().String()
	roomID := testutil.RandomUUID().String()
	ticket := storeTicket(t, client, userID)

	conn, _, err := websocket.Dial(ctx, wsURL(srv, ticket, ""), nil)
	require.NoError(t, err)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Subscribe to a room.
	subPayload, _ := json.Marshal(hub.SubscribePayload{RoomID: roomID})
	subEnv := hub.Envelope{Type: hub.TypeSubscribe, Payload: subPayload}
	require.NoError(t, wsjson.Write(ctx, conn, subEnv))

	// Give the hub a moment to register the connection.
	time.Sleep(20 * time.Millisecond)

	// Broadcast a message to the room.
	msgPayload, _ := json.Marshal(hub.MessageCreatedPayload{
		MessageID: testutil.RandomUUID().String(),
		RoomID:    roomID,
		UserID:    "sender",
	})
	h.BroadcastToRoom(roomID, hub.Envelope{
		Type:    hub.TypeMessageCreated,
		Payload: msgPayload,
	})

	var received hub.Envelope
	require.NoError(t, wsjson.Read(ctx, conn, &received))
	assert.Equal(t, hub.TypeMessageCreated, received.Type)
}
