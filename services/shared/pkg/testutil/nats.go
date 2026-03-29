package testutil

import (
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	natsd "github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/require"
)

// NATSServer wraps an embedded, in-process NATS server with JetStream enabled.
// It starts in <100ms and requires no Docker, making it suitable for both
// unit and integration tests.
//
// Typical usage:
//
//	func TestFoo(t *testing.T) {
//	    ns := testutil.NewNATSServer(t)
//	    js := ns.JetStream(t)
//	    // use js ...
//	}
type NATSServer struct {
	server *natsd.Server
	url    string
}

// NewNATSServer starts an embedded in-process NATS server with JetStream using
// in-memory storage. The server is shut down when t finishes.
func NewNATSServer(t *testing.T) *NATSServer {
	t.Helper()

	opts := &natsd.Options{
		Port:      -1, // auto-select an available port
		JetStream: true,
		NoLog:     true,
		NoSigs:    true,
		// Unique store dir prevents JetStream data from leaking between test runs.
		StoreDir:           t.TempDir(),
		JetStreamMaxMemory: 128 * 1024 * 1024, // 128 MB
	}

	s, err := natsd.NewServer(opts)
	if err != nil {
		require.NoError(t, fmt.Errorf("testutil: create embedded NATS server: %w", err))
	}

	s.Start()

	if !s.ReadyForConnections(10 * time.Second) {
		t.Fatal("testutil: embedded NATS server did not become ready in time")
	}

	ns := &NATSServer{
		server: s,
		url:    s.ClientURL(),
	}

	t.Cleanup(func() {
		s.Shutdown()
		s.WaitForShutdown()
	})

	return ns
}

// URL returns the client URL for the embedded NATS server.
func (s *NATSServer) URL() string {
	return s.url
}

// JetStream returns a connected JetStream context. The underlying NATS
// connection is closed when t finishes.
func (s *NATSServer) JetStream(t *testing.T) jetstream.JetStream {
	t.Helper()

	nc, err := nats.Connect(s.url)
	require.NoError(t, err)

	js, err := jetstream.New(nc)
	require.NoError(t, err)

	t.Cleanup(nc.Close)

	return js
}
