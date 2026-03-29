package subscriber_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	natsclient "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sharednats "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
	"github.com/sudobytemebaby/efir/services/shared/pkg/testutil"
	"github.com/sudobytemebaby/efir/services/websocket/internal/hub"
	wsnats "github.com/sudobytemebaby/efir/services/websocket/internal/nats"
	"github.com/sudobytemebaby/efir/services/websocket/internal/subscriber"
)

// captureConn records broadcast messages written to it via the hub.
type captureConn struct {
	mu       sync.Mutex
	messages []hub.Envelope
	signal   chan struct{}
}

func newCaptureConn() *captureConn {
	return &captureConn{signal: make(chan struct{}, 16)}
}

func (c *captureConn) WriteJSON(v any) error {
	if env, ok := v.(hub.Envelope); ok {
		c.mu.Lock()
		c.messages = append(c.messages, env)
		c.mu.Unlock()
		select {
		case c.signal <- struct{}{}:
		default:
		}
	}
	return nil
}

func (c *captureConn) Close(_ hub.StatusCode, _ string) error { return nil }

func (c *captureConn) waitForN(n int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		count := len(c.messages)
		c.mu.Unlock()
		if count >= n {
			return true
		}
		select {
		case <-c.signal:
		case <-time.After(10 * time.Millisecond):
		}
	}
	return false
}

func (c *captureConn) all() []hub.Envelope {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]hub.Envelope, len(c.messages))
	copy(out, c.messages)
	return out
}

func provisionStreams(t *testing.T, js jetstream.JetStream) {
	t.Helper()
	err := sharednats.ProvisionStreams(context.Background(), js, []sharednats.StreamConfig{
		{Name: wsnats.StreamMessage, Subjects: []string{"message.>"}, Storage: jetstream.MemoryStorage},
		{Name: wsnats.StreamRoom, Subjects: []string{"room.>"}, Storage: jetstream.MemoryStorage},
	})
	require.NoError(t, err)
}

func publish(t *testing.T, url, subject string, payload any) {
	t.Helper()
	nc, err := natsclient.Connect(url)
	require.NoError(t, err)
	defer nc.Close()
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	require.NoError(t, nc.Publish(subject, b))
}

func newSub(t *testing.T, h *hub.Hub, js jetstream.JetStream) *subscriber.Subscriber {
	t.Helper()
	return subscriber.NewSubscriber(h, js, subscriber.SubscriberConfig{
		MaxDeliver: 3,
		AckWait:    5 * time.Second,
		RetryWait:  100 * time.Millisecond,
	})
}

func TestSubscriber_MessageCreated(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ns := testutil.NewNATSServer(t)
	js := ns.JetStream(t)
	provisionStreams(t, js)

	h := hub.NewHub(64)
	go h.Run(ctx)

	conn := newCaptureConn()
	roomID := testutil.RandomUUID().String()
	h.Register(conn, "user-1", roomID)
	time.Sleep(10 * time.Millisecond)

	sub := newSub(t, h, js)
	require.NoError(t, sub.Start(ctx))

	msgID := testutil.RandomUUID().String()
	publish(t, ns.URL(), wsnats.SubjectMessageCreated, subscriber.MessageCreatedEvent{
		MessageID: msgID,
		RoomID:    roomID,
		UserID:    "user-1",
	})

	require.True(t, conn.waitForN(1, 3*time.Second), "timeout waiting for message.created broadcast")
	msgs := conn.all()
	require.Len(t, msgs, 1)
	assert.Equal(t, hub.TypeMessageCreated, msgs[0].Type)

	var payload hub.MessageCreatedPayload
	require.NoError(t, json.Unmarshal(msgs[0].Payload, &payload))
	assert.Equal(t, msgID, payload.MessageID)
	assert.Equal(t, roomID, payload.RoomID)
}

func TestSubscriber_RoomMembershipChanged(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ns := testutil.NewNATSServer(t)
	js := ns.JetStream(t)
	provisionStreams(t, js)

	h := hub.NewHub(64)
	go h.Run(ctx)

	conn := newCaptureConn()
	roomID := testutil.RandomUUID().String()
	h.Register(conn, "user-1", roomID)
	time.Sleep(10 * time.Millisecond)

	sub := newSub(t, h, js)
	require.NoError(t, sub.Start(ctx))

	publish(t, ns.URL(), wsnats.SubjectRoomMembershipChanged, subscriber.RoomMembershipChangedEvent{
		RoomID: roomID,
		UserID: "user-2",
		Joined: true,
	})

	require.True(t, conn.waitForN(1, 3*time.Second), "timeout waiting for membership.changed broadcast")
	msgs := conn.all()
	require.Len(t, msgs, 1)
	assert.Equal(t, hub.TypeRoomMembershipChange, msgs[0].Type)

	var payload hub.RoomMembershipChangedPayload
	require.NoError(t, json.Unmarshal(msgs[0].Payload, &payload))
	assert.Equal(t, roomID, payload.RoomID)
	assert.True(t, payload.Joined)
}

func TestSubscriber_RoomUpdated(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ns := testutil.NewNATSServer(t)
	js := ns.JetStream(t)
	provisionStreams(t, js)

	h := hub.NewHub(64)
	go h.Run(ctx)

	conn := newCaptureConn()
	roomID := testutil.RandomUUID().String()
	h.Register(conn, "user-1", roomID)
	time.Sleep(10 * time.Millisecond)

	sub := newSub(t, h, js)
	require.NoError(t, sub.Start(ctx))

	publish(t, ns.URL(), wsnats.SubjectRoomUpdated, subscriber.RoomUpdatedEvent{
		RoomID:    roomID,
		UpdatedBy: "user-1",
	})

	require.True(t, conn.waitForN(1, 3*time.Second))
	msgs := conn.all()
	require.Len(t, msgs, 1)
	assert.Equal(t, hub.TypeRoomUpdated, msgs[0].Type)
}
