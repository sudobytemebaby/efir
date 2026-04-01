package nats

import (
	"github.com/nats-io/nats.go/jetstream"
	sharedjs "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
)

const (
	StreamRoom              = "ROOM"
	SubjectRoomCreated      = "room.created"
	SubjectMembershipChange = "room.membership.changed"
	SubjectRoomUpdated      = "room.updated"
	SubjectRoomDeleted      = "room.deleted"
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
