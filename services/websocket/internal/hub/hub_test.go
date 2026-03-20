package hub

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConn struct {
	mu          sync.Mutex
	writes      []Envelope
	closed      bool
	closeCode   StatusCode
	closeReason string
	writeSignal chan struct{}
}

func newMockConn() *mockConn {
	return &mockConn{
		writeSignal: make(chan struct{}, 1),
	}
}

func (m *mockConn) WriteJSON(v any) error {
	m.mu.Lock()
	if env, ok := v.(Envelope); ok {
		m.writes = append(m.writes, env)
		select {
		case m.writeSignal <- struct{}{}:
		default:
		}
	}
	m.mu.Unlock()
	return nil
}

func (m *mockConn) Close(code StatusCode, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	m.closeCode = code
	m.closeReason = reason
	return nil
}

func (m *mockConn) getWrites() []Envelope {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writes
}

func (m *mockConn) waitForWrite(timeout time.Duration) bool {
	select {
	case <-m.writeSignal:
		return true
	case <-time.After(timeout):
		return false
	}
}

func waitForHubProcess(timeout time.Duration) {
	time.Sleep(time.Millisecond)
}

func TestNewHub(t *testing.T) {
	hub := NewHub()
	require.NotNil(t, hub)
	assert.NotNil(t, hub.rooms)
	assert.NotNil(t, hub.userIDs)
	assert.NotNil(t, hub.connRooms)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.disconnect)
	assert.NotNil(t, hub.broadcast)
}

func TestHub_RegisterUnregister(t *testing.T) {
	hub := NewHub()
	conn := newMockConn()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	hub.Register(conn, "user1", "room1")
	waitForHubProcess(10 * time.Millisecond)

	count := hub.GetRoomUserCount("room1")
	assert.Equal(t, 1, count)

	hub.Unregister(conn, "room1")
	waitForHubProcess(10 * time.Millisecond)

	count = hub.GetRoomUserCount("room1")
	assert.Equal(t, 0, count)
}

func TestHub_BroadcastToRoom(t *testing.T) {
	hub := NewHub()
	conn1 := newMockConn()
	conn2 := newMockConn()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	hub.Register(conn1, "user1", "room1")
	hub.Register(conn2, "user2", "room1")
	waitForHubProcess(10 * time.Millisecond)

	envelope := Envelope{
		Type:    TypeMessageCreated,
		Payload: []byte(`{"message_id":"123"}`),
	}
	hub.BroadcastToRoom("room1", envelope)

	require.True(t, conn1.waitForWrite(100*time.Millisecond), "conn1 did not receive write")
	require.True(t, conn2.waitForWrite(100*time.Millisecond), "conn2 did not receive write")

	writes1 := conn1.getWrites()
	writes2 := conn2.getWrites()

	assert.Len(t, writes1, 1)
	assert.Len(t, writes2, 1)
	assert.Equal(t, TypeMessageCreated, writes1[0].Type)
	assert.Equal(t, TypeMessageCreated, writes2[0].Type)
}

func TestHub_BroadcastToRoom_NoConns(t *testing.T) {
	hub := NewHub()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	envelope := Envelope{
		Type:    TypeMessageCreated,
		Payload: []byte(`{"message_id":"123"}`),
	}
	hub.BroadcastToRoom("nonexistent", envelope)
	waitForHubProcess(10 * time.Millisecond)
}

func TestHub_MultipleRooms(t *testing.T) {
	hub := NewHub()
	conn1 := newMockConn()
	conn2 := newMockConn()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	hub.Register(conn1, "user1", "room1")
	hub.Register(conn2, "user2", "room2")
	waitForHubProcess(10 * time.Millisecond)

	envelope := Envelope{
		Type:    TypeMessageCreated,
		Payload: []byte(`{"message_id":"123"}`),
	}
	hub.BroadcastToRoom("room1", envelope)

	require.True(t, conn1.waitForWrite(100*time.Millisecond), "conn1 did not receive write")

	writes1 := conn1.getWrites()
	writes2 := conn2.getWrites()

	assert.Len(t, writes1, 1)
	assert.Len(t, writes2, 0)
}

func TestHub_MultipleConnsSameUser(t *testing.T) {
	hub := NewHub()
	conn1 := newMockConn()
	conn2 := newMockConn()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	hub.Register(conn1, "user1", "room1")
	hub.Register(conn2, "user1", "room1")
	waitForHubProcess(10 * time.Millisecond)

	envelope := Envelope{
		Type:    TypeMessageCreated,
		Payload: []byte(`{"message_id":"123"}`),
	}
	hub.BroadcastToRoom("room1", envelope)

	require.True(t, conn1.waitForWrite(100*time.Millisecond), "conn1 did not receive write")
	require.True(t, conn2.waitForWrite(100*time.Millisecond), "conn2 did not receive write")

	writes1 := conn1.getWrites()
	writes2 := conn2.getWrites()

	assert.Len(t, writes1, 1)
	assert.Len(t, writes2, 1)
}

func TestHub_GetRoomUserCount(t *testing.T) {
	hub := NewHub()
	conn := newMockConn()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	count := hub.GetRoomUserCount("nonexistent")
	assert.Equal(t, 0, count)

	hub.Register(conn, "user1", "room1")
	waitForHubProcess(10 * time.Millisecond)

	count = hub.GetRoomUserCount("room1")
	assert.Equal(t, 1, count)
}

func TestHub_Disconnect(t *testing.T) {
	hub := NewHub()
	conn := newMockConn()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	hub.Register(conn, "user1", "room1")
	hub.Register(conn, "user1", "room2")
	hub.Register(conn, "user1", "room3")
	waitForHubProcess(10 * time.Millisecond)

	assert.Equal(t, 1, hub.GetRoomUserCount("room1"))
	assert.Equal(t, 1, hub.GetRoomUserCount("room2"))
	assert.Equal(t, 1, hub.GetRoomUserCount("room3"))

	hub.Disconnect(conn)
	waitForHubProcess(10 * time.Millisecond)

	assert.Equal(t, 0, hub.GetRoomUserCount("room1"))
	assert.Equal(t, 0, hub.GetRoomUserCount("room2"))
	assert.Equal(t, 0, hub.GetRoomUserCount("room3"))
}

func TestHub_SubscribeUnsubscribeMultipleRooms(t *testing.T) {
	hub := NewHub()
	conn := newMockConn()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	hub.Register(conn, "user1", "room1")
	hub.Register(conn, "user1", "room2")
	waitForHubProcess(10 * time.Millisecond)

	envelope := Envelope{
		Type:    TypeMessageCreated,
		Payload: []byte(`{"message_id":"123"}`),
	}
	hub.BroadcastToRoom("room1", envelope)
	hub.BroadcastToRoom("room2", envelope)

	require.True(t, conn.waitForWrite(200*time.Millisecond), "conn did not receive writes")

	hub.Unregister(conn, "room1")
	waitForHubProcess(10 * time.Millisecond)

	hub.BroadcastToRoom("room1", envelope)
	hub.BroadcastToRoom("room2", envelope)

	hub.Disconnect(conn)
	waitForHubProcess(10 * time.Millisecond)

	writes := conn.getWrites()
	assert.Len(t, writes, 2)
}

func TestHub_CloseOnWriteError(t *testing.T) {
	hub := NewHub()
	conn := &errorMockConn{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	hub.Register(conn, "user1", "room1")
	waitForHubProcess(10 * time.Millisecond)

	envelope := Envelope{
		Type:    TypeMessageCreated,
		Payload: []byte(`{"message_id":"123"}`),
	}
	hub.BroadcastToRoom("room1", envelope)
	waitForHubProcess(10 * time.Millisecond)

	assert.True(t, conn.closed)
	assert.Equal(t, StatusAbnormalClosure, conn.closeCode)
}

type errorMockConn struct {
	closed      bool
	closeCode   StatusCode
	closeReason string
}

func (m *errorMockConn) WriteJSON(v any) error {
	return assert.AnError
}

func (m *errorMockConn) Close(code StatusCode, reason string) error {
	m.closed = true
	m.closeCode = code
	m.closeReason = reason
	return nil
}
