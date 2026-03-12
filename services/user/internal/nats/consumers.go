package nats

import sharedjs "github.com/sudobytemebaby/efir/services/shared/pkg/nats"

const (
	StreamAuth                = "AUTH"
	SubjectAuthUserRegistered = "auth.user.registered"
	ConsumerUserRegistered    = "user-svc-auth-registered"
)

func UserRegisteredConsumer() sharedjs.ConsumerConfig {
	return sharedjs.DefaultConsumerConfig(ConsumerUserRegistered, SubjectAuthUserRegistered)
}
