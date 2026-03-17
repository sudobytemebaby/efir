package nats

import (
	"github.com/nats-io/nats.go/jetstream"
	sharedjs "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
)

const (
	StreamRoom              = "ROOM"
	SubjectMembershipChange = "room.membership.changed"
)

func Streams() []sharedjs.StreamConfig {
	return []sharedjs.StreamConfig{
		{
			Name:      StreamRoom,
			Subjects:  []string{"room.>"},
			Retention: jetstream.LimitsPolicy,
			Storage:   jetstream.FileStorage,
			Replicas:  1,
		},
	}
}
