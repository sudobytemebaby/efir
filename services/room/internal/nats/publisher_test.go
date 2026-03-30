package nats_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	roomnats "github.com/sudobytemebaby/efir/services/room/internal/nats"
	"github.com/sudobytemebaby/efir/services/room/internal/service"
	sharedjs "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
	"github.com/sudobytemebaby/efir/services/shared/pkg/testutil"
)

func setup(t *testing.T) (service.Publisher, jetstream.JetStream) {
	t.Helper()
	ns := testutil.NewNATSServer(t)
	js := ns.JetStream(t)
	ctx := context.Background()
	err := sharedjs.ProvisionStreams(ctx, js, roomnats.Streams())
	require.NoError(t, err)
	return roomnats.NewPublisher(js), js
}

// capture creates a consumer for new messages on subject and returns a
// function that blocks until one message arrives or 3 s elapse.
// The consumer must be created BEFORE publishing to avoid missing the message.
func capture(t *testing.T, js jetstream.JetStream, streamName, subject string) func() []byte {
	t.Helper()
	ctx := context.Background()
	received := make(chan []byte, 1)

	cons, err := js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		FilterSubject: subject,
		DeliverPolicy: jetstream.DeliverNewPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	require.NoError(t, err)

	sub, err := cons.Consume(func(msg jetstream.Msg) {
		_ = msg.Ack()
		select {
		case received <- append([]byte(nil), msg.Data()...):
		default:
		}
	})
	require.NoError(t, err)
	t.Cleanup(sub.Stop)

	return func() []byte {
		select {
		case data := <-received:
			return data
		case <-time.After(3 * time.Second):
			t.Fatal("timeout: no message received on subject " + subject)
			return nil
		}
	}
}

func TestPublishMembershipChanged(t *testing.T) {
	pub, js := setup(t)
	ctx := context.Background()

	roomID := uuid.New()
	userID := uuid.New()
	recipientID := uuid.New()

	get := capture(t, js, roomnats.StreamRoom, roomnats.SubjectMembershipChange)

	err := pub.PublishMembershipChanged(ctx, roomID, userID, "added", []uuid.UUID{recipientID})
	require.NoError(t, err)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(get(), &payload))

	assert.Equal(t, roomID.String(), payload["room_id"])
	assert.Equal(t, userID.String(), payload["user_id"])
	assert.Equal(t, "added", payload["action"])
	recipients, ok := payload["recipient_ids"].([]interface{})
	require.True(t, ok)
	require.Len(t, recipients, 1)
	assert.Equal(t, recipientID.String(), recipients[0])
}

func TestPublishRoomUpdated(t *testing.T) {
	pub, js := setup(t)
	ctx := context.Background()

	roomID := uuid.New()
	recipientID := uuid.New()
	name := "New Room Name"

	get := capture(t, js, roomnats.StreamRoom, roomnats.SubjectRoomUpdated)

	err := pub.PublishRoomUpdated(ctx, roomID, name, []uuid.UUID{recipientID})
	require.NoError(t, err)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(get(), &payload))

	assert.Equal(t, roomID.String(), payload["room_id"])
	assert.Equal(t, name, payload["name"])
	recipients, ok := payload["recipient_ids"].([]interface{})
	require.True(t, ok)
	require.Len(t, recipients, 1)
	assert.Equal(t, recipientID.String(), recipients[0])
}

func TestPublishMembershipChanged_EmptyRecipients(t *testing.T) {
	pub, js := setup(t)
	ctx := context.Background()

	get := capture(t, js, roomnats.StreamRoom, roomnats.SubjectMembershipChange)

	err := pub.PublishMembershipChanged(ctx, uuid.New(), uuid.New(), "removed", []uuid.UUID{})
	require.NoError(t, err)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(get(), &payload))

	recipients, ok := payload["recipient_ids"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, recipients)
}
