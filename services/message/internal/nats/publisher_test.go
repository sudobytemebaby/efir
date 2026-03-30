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

	msgnats "github.com/sudobytemebaby/efir/services/message/internal/nats"
	"github.com/sudobytemebaby/efir/services/message/internal/service"
	sharedjs "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
	"github.com/sudobytemebaby/efir/services/shared/pkg/testutil"
)

func setup(t *testing.T) (service.Publisher, jetstream.JetStream) {
	t.Helper()
	ns := testutil.NewNATSServer(t)
	js := ns.JetStream(t)
	ctx := context.Background()
	err := sharedjs.ProvisionStreams(ctx, js, msgnats.Streams())
	require.NoError(t, err)
	return msgnats.NewPublisher(js), js
}

// capture creates a consumer for new messages on subject and returns a
// function that blocks until one message arrives or 3 s elapse.
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

func TestPublishMessageCreated(t *testing.T) {
	pub, js := setup(t)
	ctx := context.Background()

	msgID := uuid.New()
	roomID := uuid.New()
	senderID := uuid.New()
	recipientID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	msg := &service.Message{
		ID:        msgID,
		RoomID:    roomID,
		SenderID:  senderID,
		Type:      service.MessageTypeText,
		Content:   service.TextContent{Text: "hello"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	get := capture(t, js, msgnats.StreamMessage, msgnats.SubjectMessageCreated)

	err := pub.PublishMessageCreated(ctx, msg, []uuid.UUID{recipientID})
	require.NoError(t, err)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(get(), &payload))

	assert.Equal(t, msgID.String(), payload["message_id"])
	assert.Equal(t, roomID.String(), payload["room_id"])
	assert.Equal(t, senderID.String(), payload["sender_id"])
	assert.Equal(t, string(service.MessageTypeText), payload["type"])
	recipients, ok := payload["recipient_ids"].([]interface{})
	require.True(t, ok)
	require.Len(t, recipients, 1)
	assert.Equal(t, recipientID.String(), recipients[0])
}

func TestPublishMessageCreated_MultipleRecipients(t *testing.T) {
	pub, js := setup(t)
	ctx := context.Background()

	msgID := uuid.New()
	roomID := uuid.New()
	now := time.Now().UTC()
	r1, r2, r3 := uuid.New(), uuid.New(), uuid.New()

	msg := &service.Message{
		ID:        msgID,
		RoomID:    roomID,
		SenderID:  uuid.New(),
		Type:      service.MessageTypeText,
		Content:   service.TextContent{Text: "broadcast"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	get := capture(t, js, msgnats.StreamMessage, msgnats.SubjectMessageCreated)

	err := pub.PublishMessageCreated(ctx, msg, []uuid.UUID{r1, r2, r3})
	require.NoError(t, err)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(get(), &payload))

	recipients, ok := payload["recipient_ids"].([]interface{})
	require.True(t, ok)
	assert.Len(t, recipients, 3)
}

func TestPublishMessageCreated_NoRecipients(t *testing.T) {
	pub, js := setup(t)
	ctx := context.Background()

	now := time.Now().UTC()
	msg := &service.Message{
		ID:        uuid.New(),
		RoomID:    uuid.New(),
		SenderID:  uuid.New(),
		Type:      service.MessageTypeText,
		Content:   service.TextContent{Text: "solo"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	get := capture(t, js, msgnats.StreamMessage, msgnats.SubjectMessageCreated)

	err := pub.PublishMessageCreated(ctx, msg, []uuid.UUID{})
	require.NoError(t, err)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(get(), &payload))

	recipients, ok := payload["recipient_ids"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, recipients)
}
