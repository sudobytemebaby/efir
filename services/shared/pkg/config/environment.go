package config

import (
	"fmt"
	"time"
)

type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvProduction  Environment = "production"
)

func (e Environment) Validate() error {
	switch e {
	case EnvDevelopment, EnvProduction:
		return nil
	default:
		return fmt.Errorf("invalid environment %q, allowed: development, production", e)
	}
}

type ServerConfig struct {
	GRPCPort         string `yaml:"grpc_port"          env:"GRPC_PORT"`
	HealthListenAddr string `yaml:"health_listen_addr" env:"HEALTH_LISTEN_ADDR"`
}

type TimeoutsConfig struct {
	Shutdown     time.Duration `yaml:"shutdown"       env:"SHUTDOWN_TIMEOUT"`
	ReadHeader   time.Duration `yaml:"read_header"   env:"READ_HEADER_TIMEOUT"`
	GRPCGraceful time.Duration `yaml:"grpc_graceful" env:"GRPC_GRACEFUL_TIMEOUT"`
}

type NATSConfig struct {
	URL               string        `yaml:"url"               env:"NATS_URL"`
	User              string        `yaml:"user"              env:"NATS_USER"`
	Pass              string        `yaml:"-"                 env:"NATS_PASSWORD"`
	ReconnectWait     time.Duration `yaml:"reconnect_wait"   env:"NATS_RECONNECT_WAIT"`
	MaxReconnects     int           `yaml:"max_reconnects"   env:"NATS_MAX_RECONNECTS"`
	AckWait           time.Duration `yaml:"ack_wait"         env:"NATS_ACK_WAIT"`
	MaxDeliver        int           `yaml:"max_deliver"      env:"NATS_MAX_DELIVER"`
	ConsumerRetryWait time.Duration `yaml:"consumer_retry_wait" env:"NATS_CONSUMER_RETRY_WAIT"`
}
