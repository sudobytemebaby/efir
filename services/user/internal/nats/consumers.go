package nats

import (
	"time"

	sharedjs "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
)

const (
	StreamAuth                = "AUTH"
	SubjectAuthUserRegistered = "auth.user.registered"
	ConsumerUserRegistered    = "user-svc-auth-registered"
)

func UserRegisteredConsumer(maxDeliver int, ackWait time.Duration) sharedjs.ConsumerConfig {
	return sharedjs.DefaultConsumerConfig(ConsumerUserRegistered, SubjectAuthUserRegistered, maxDeliver, ackWait)
}
