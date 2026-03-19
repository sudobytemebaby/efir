package nats

import (
	"github.com/nats-io/nats.go/jetstream"
	sharedjs "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
)

const (
	StreamMessage         = "MESSAGE"
	SubjectMessageCreated = "message.created"
)

func Streams() []sharedjs.StreamConfig {
	return []sharedjs.StreamConfig{
		{
			Name:      StreamMessage,
			Subjects:  []string{"message.>"},
			Retention: jetstream.LimitsPolicy,
			Storage:   jetstream.FileStorage,
			Replicas:  1,
		},
	}
}
