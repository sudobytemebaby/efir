package nats

import (
	"github.com/nats-io/nats.go/jetstream"

	sharedjs "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
)

func Streams() []sharedjs.StreamConfig {
	return []sharedjs.StreamConfig{
		{
			Name:      "MESSAGE",
			Subjects:  []string{"message.>"},
			Retention: jetstream.LimitsPolicy,
			Storage:   jetstream.FileStorage,
			Replicas:  1,
		},
	}
}
