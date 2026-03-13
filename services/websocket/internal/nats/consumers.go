package nats

import sharedjs "github.com/sudobytemebaby/efir/services/shared/pkg/nats"

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

func MessageCreatedConsumer() sharedjs.ConsumerConfig {
	return sharedjs.DefaultConsumerConfig(ConsumerMessageCreated, SubjectMessageCreated)
}

func RoomMembershipChangedConsumer() sharedjs.ConsumerConfig {
	return sharedjs.DefaultConsumerConfig(ConsumerRoomMembershipChanged, SubjectRoomMembershipChanged)
}

func RoomUpdatedConsumer() sharedjs.ConsumerConfig {
	return sharedjs.DefaultConsumerConfig(ConsumerRoomUpdated, SubjectRoomUpdated)
}
