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
}

func (m *mockConn) WriteJSON(v any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if env, ok := v.(Envelope); ok {
		m.writes = append(m.writes, env)
	}
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

func TestNewHub(t *testing.T) {
	hub := NewHub()
	require.NotNil(t, hub)
	assert.NotNil(t, hub.rooms)
	assert.NotNil(t, hub.userIDs)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.broadcast)
}

func TestHub_RegisterUnregister(t *testing.T) {
	hub := NewHub()
	conn := &mockConn{}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		hub.Run()
	}()
	defer cancel()
	_ = ctx

	hub.Register(conn, "user1", "room1")
	time.Sleep(10 * time.Millisecond)

	count := hub.GetRoomUserCount("room1")
	assert.Equal(t, 1, count)

	hub.Unregister(conn, "room1")
	time.Sleep(10 * time.Millisecond)

	count = hub.GetRoomUserCount("room1")
	assert.Equal(t, 0, count)
}

func TestHub_BroadcastToRoom(t *testing.T) {
	hub := NewHub()
	conn1 := &mockConn{}
	conn2 := &mockConn{}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		hub.Run()
	}()
	defer cancel()
	_ = ctx

	hub.Register(conn1, "user1", "room1")
	hub.Register(conn2, "user2", "room1")
	time.Sleep(10 * time.Millisecond)

	envelope := Envelope{
		Type:    TypeMessageCreated,
		Payload: []byte(`{"message_id":"123"}`),
	}
	hub.BroadcastToRoom("room1", envelope)
	time.Sleep(10 * time.Millisecond)

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
	go func() {
		hub.Run()
	}()
	defer cancel()
	_ = ctx

	envelope := Envelope{
		Type:    TypeMessageCreated,
		Payload: []byte(`{"message_id":"123"}`),
	}
	hub.BroadcastToRoom("nonexistent", envelope)
	time.Sleep(10 * time.Millisecond)
}

func TestHub_MultipleRooms(t *testing.T) {
	hub := NewHub()
	conn1 := &mockConn{}
	conn2 := &mockConn{}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		hub.Run()
	}()
	defer cancel()
	_ = ctx

	hub.Register(conn1, "user1", "room1")
	hub.Register(conn2, "user2", "room2")
	time.Sleep(10 * time.Millisecond)

	envelope := Envelope{
		Type:    TypeMessageCreated,
		Payload: []byte(`{"message_id":"123"}`),
	}
	hub.BroadcastToRoom("room1", envelope)
	time.Sleep(10 * time.Millisecond)

	writes1 := conn1.getWrites()
	writes2 := conn2.getWrites()

	assert.Len(t, writes1, 1)
	assert.Len(t, writes2, 0)
}

func TestHub_MultipleConnsSameUser(t *testing.T) {
	hub := NewHub()
	conn1 := &mockConn{}
	conn2 := &mockConn{}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		hub.Run()
	}()
	defer cancel()
	_ = ctx

	hub.Register(conn1, "user1", "room1")
	hub.Register(conn2, "user1", "room1")
	time.Sleep(10 * time.Millisecond)

	envelope := Envelope{
		Type:    TypeMessageCreated,
		Payload: []byte(`{"message_id":"123"}`),
	}
	hub.BroadcastToRoom("room1", envelope)
	time.Sleep(10 * time.Millisecond)

	writes1 := conn1.getWrites()
	writes2 := conn2.getWrites()

	assert.Len(t, writes1, 1)
	assert.Len(t, writes2, 1)
}

func TestHub_GetRoomUserCount(t *testing.T) {
	hub := NewHub()
	conn := &mockConn{}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		hub.Run()
	}()
	defer cancel()
	_ = ctx

	count := hub.GetRoomUserCount("nonexistent")
	assert.Equal(t, 0, count)

	hub.Register(conn, "user1", "room1")
	time.Sleep(10 * time.Millisecond)

	count = hub.GetRoomUserCount("room1")
	assert.Equal(t, 1, count)
}
