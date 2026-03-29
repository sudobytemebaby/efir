package nats_test

import (
	"context"
	"testing"
	"time"

	natsclient "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudobytemebaby/efir/services/shared/pkg/nats"
	"github.com/sudobytemebaby/efir/services/shared/pkg/testutil"
)

func TestConnect(t *testing.T) {
	t.Parallel()

	t.Run("success with no-auth embedded server", func(t *testing.T) {
		t.Parallel()
		ns := testutil.NewNATSServer(t)
		nc, err := nats.Connect(ns.URL(), "", "", nats.ConnectOptions{
			ReconnectWait: time.Second,
			MaxReconnects: 3,
		})
		require.NoError(t, err)
		defer nc.Close()
		assert.True(t, nc.IsConnected())
	})

	t.Run("invalid URL returns error", func(t *testing.T) {
		t.Parallel()
		_, err := nats.Connect("nats://127.0.0.1:1", "", "", nats.ConnectOptions{
			ReconnectWait: 0,
			MaxReconnects: 0,
		})
		assert.Error(t, err)
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	ns := testutil.NewNATSServer(t)
	nc, err := natsclient.Connect(ns.URL())
	require.NoError(t, err)
	t.Cleanup(nc.Close)

	js, err := nats.New(nc)
	require.NoError(t, err)
	assert.NotNil(t, js)
}

func TestProvisionStreams(t *testing.T) {
	t.Parallel()

	t.Run("creates new stream", func(t *testing.T) {
		t.Parallel()
		ns := testutil.NewNATSServer(t)
		js := ns.JetStream(t)
		ctx := context.Background()

		cfg := nats.StreamConfig{
			Name:     "TEST_STREAM",
			Subjects: []string{"test.>"},
		}

		err := nats.ProvisionStreams(ctx, js, []nats.StreamConfig{cfg})
		require.NoError(t, err)

		info, err := js.Stream(ctx, "TEST_STREAM")
		require.NoError(t, err)
		assert.Equal(t, "TEST_STREAM", info.CachedInfo().Config.Name)
	})

	t.Run("updates existing stream without error", func(t *testing.T) {
		t.Parallel()
		ns := testutil.NewNATSServer(t)
		js := ns.JetStream(t)
		ctx := context.Background()

		cfg := nats.StreamConfig{
			Name:     "TEST_UPDATE",
			Subjects: []string{"update.>"},
		}

		require.NoError(t, nats.ProvisionStreams(ctx, js, []nats.StreamConfig{cfg}))
		// Calling again is idempotent.
		require.NoError(t, nats.ProvisionStreams(ctx, js, []nats.StreamConfig{cfg}))
	})

	t.Run("creates multiple streams", func(t *testing.T) {
		t.Parallel()
		ns := testutil.NewNATSServer(t)
		js := ns.JetStream(t)
		ctx := context.Background()

		streams := []nats.StreamConfig{
			{Name: "MULTI_A", Subjects: []string{"multi.a.>"}},
			{Name: "MULTI_B", Subjects: []string{"multi.b.>"}},
		}

		require.NoError(t, nats.ProvisionStreams(ctx, js, streams))

		for _, s := range streams {
			info, err := js.Stream(ctx, s.Name)
			require.NoError(t, err)
			assert.Equal(t, s.Name, info.CachedInfo().Config.Name)
		}
	})
}

func TestProvisionConsumer(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ns := testutil.NewNATSServer(t)
		js := ns.JetStream(t)
		ctx := context.Background()

		_, err := js.CreateStream(ctx, jetstream.StreamConfig{
			Name:     "CONS_STREAM",
			Subjects: []string{"cons.>"},
		})
		require.NoError(t, err)

		cfg := nats.DefaultConsumerConfig("test-consumer", "cons.events", 3, 30*time.Second)
		consumer, err := nats.ProvisionConsumer(ctx, js, "CONS_STREAM", cfg)
		require.NoError(t, err)
		assert.Equal(t, "test-consumer", consumer.CachedInfo().Config.Durable)
	})

	t.Run("stream not found returns error", func(t *testing.T) {
		t.Parallel()
		ns := testutil.NewNATSServer(t)
		js := ns.JetStream(t)
		ctx := context.Background()

		cfg := nats.DefaultConsumerConfig("c", "x.y", 1, time.Second)
		_, err := nats.ProvisionConsumer(ctx, js, "NONEXISTENT", cfg)
		require.Error(t, err)
	})
}

func TestProvisionConsumerWithRetry(t *testing.T) {
	t.Parallel()

	t.Run("stream appears after retry", func(t *testing.T) {
		t.Parallel()
		ns := testutil.NewNATSServer(t)
		js := ns.JetStream(t)
		ctx := context.Background()

		// Create the stream after a short delay.
		go func() {
			time.Sleep(100 * time.Millisecond)
			js.CreateStream(ctx, jetstream.StreamConfig{ //nolint:errcheck
				Name:     "RETRY_STREAM",
				Subjects: []string{"retry.>"},
			})
		}()

		cfg := nats.DefaultConsumerConfig("retry-consumer", "retry.events", 3, 30*time.Second)
		consumer, err := nats.ProvisionConsumerWithRetry(ctx, js, "RETRY_STREAM", cfg, 20*time.Millisecond)
		require.NoError(t, err)
		assert.Equal(t, "retry-consumer", consumer.CachedInfo().Config.Durable)
	})

	t.Run("context cancelled returns error", func(t *testing.T) {
		t.Parallel()
		ns := testutil.NewNATSServer(t)
		js := ns.JetStream(t)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		cfg := nats.DefaultConsumerConfig("c", "x.y", 1, time.Second)
		_, err := nats.ProvisionConsumerWithRetry(ctx, js, "NEVER_CREATED", cfg, 10*time.Millisecond)
		require.Error(t, err)
	})
}
