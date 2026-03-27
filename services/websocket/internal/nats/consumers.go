package nats

import (
	"time"

	sharedjs "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
)

const (
	StreamMessage                 = "MESSAGE"
	StreamRoom                    = "ROOM"
	SubjectMessageCreated         = "message.created"
	SubjectRoomMembershipChanged  = "room.membership.changed"
	SubjectRoomUpdated            = "room.updated"
	ConsumerMessageCreated        = "ws-svc-message-created"
	ConsumerRoomMembershipChanged = "ws-svc-room-membership"
	ConsumerRoomUpdated           = "ws-svc-room-updated"
)

func MessageCreatedConsumer(maxDeliver int, ackWait time.Duration) sharedjs.ConsumerConfig {
	return sharedjs.DefaultConsumerConfig(ConsumerMessageCreated, SubjectMessageCreated, maxDeliver, ackWait)
}

func RoomMembershipChangedConsumer(maxDeliver int, ackWait time.Duration) sharedjs.ConsumerConfig {
	return sharedjs.DefaultConsumerConfig(ConsumerRoomMembershipChanged, SubjectRoomMembershipChanged, maxDeliver, ackWait)
}

func RoomUpdatedConsumer(maxDeliver int, ackWait time.Duration) sharedjs.ConsumerConfig {
	return sharedjs.DefaultConsumerConfig(ConsumerRoomUpdated, SubjectRoomUpdated, maxDeliver, ackWait)
}
