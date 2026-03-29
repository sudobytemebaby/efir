package nats_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	natsclient "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	usernats "github.com/sudobytemebaby/efir/services/user/internal/nats"
	svcmocks "github.com/sudobytemebaby/efir/services/user/internal/service/mocks"
	sharednats "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
	"github.com/sudobytemebaby/efir/services/shared/pkg/testutil"
)

// provisionAuthStream creates the AUTH stream required by the subscriber.
func provisionAuthStream(t *testing.T, js jetstream.JetStream) {
	t.Helper()
	err := sharednats.ProvisionStreams(context.Background(), js, []sharednats.StreamConfig{
		{
			Name:     usernats.StreamAuth,
			Subjects: []string{"auth.>"},
			Storage:  jetstream.MemoryStorage,
		},
	})
	require.NoError(t, err)
}

// publishRegistered publishes a user_registered event directly via the raw NATS connection.
func publishRegistered(t *testing.T, url string, userID uuid.UUID, email string) {
	t.Helper()
	nc, err := natsclient.Connect(url)
	require.NoError(t, err)
	defer nc.Close()

	payload, err := json.Marshal(map[string]string{
		"user_id": userID.String(),
		"email":   email,
	})
	require.NoError(t, err)

	require.NoError(t, nc.Publish(usernats.SubjectAuthUserRegistered, payload))
}

func TestSubscriber_HandlesUserRegisteredEvent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ns := testutil.NewNATSServer(t)
	js := ns.JetStream(t)
	provisionAuthStream(t, js)

	userID := testutil.RandomUUID()
	email := testutil.RandomEmail()

	done := make(chan struct{})
	svcMock := svcmocks.NewUserService(t)
	svcMock.On("CreateUser", mock.Anything, userID, email).
		Return(nil, nil).
		Run(func(args mock.Arguments) { close(done) }).
		Once()

	cfg := usernats.SubscriberConfig{
		MaxDeliver: 3,
		AckWait:    5 * time.Second,
		RetryWait:  100 * time.Millisecond,
	}
	sub := usernats.NewSubscriber(js, svcMock, cfg)
	require.NoError(t, sub.Start(ctx))

	publishRegistered(t, ns.URL(), userID, email)

	select {
	case <-done:
		// CreateUser was called with correct args — test passes.
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for subscriber to process user_registered event")
	}
}

func TestSubscriber_NaksOnInvalidPayload(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ns := testutil.NewNATSServer(t)
	js := ns.JetStream(t)
	provisionAuthStream(t, js)

	svcMock := svcmocks.NewUserService(t)
	// CreateUser must NOT be called for an invalid payload.

	cfg := usernats.SubscriberConfig{
		MaxDeliver: 1,
		AckWait:    5 * time.Second,
		RetryWait:  100 * time.Millisecond,
	}
	sub := usernats.NewSubscriber(js, svcMock, cfg)
	require.NoError(t, sub.Start(ctx))

	// Publish garbage.
	nc, err := natsclient.Connect(ns.URL())
	require.NoError(t, err)
	defer nc.Close()
	require.NoError(t, nc.Publish(usernats.SubjectAuthUserRegistered, []byte("not json")))

	// Give the subscriber time to process; CreateUser should never be called.
	time.Sleep(200 * time.Millisecond)
	svcMock.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything, mock.Anything)
}
