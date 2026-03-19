package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env             Environment `env:"ENV" env-default:"development"`
	LogLevel        string      `env:"LOG_LEVEL" env-default:"info"`
	GRPCPort        string      `env:"GRPC_PORT" env-default:"50054"`
	PostgresDSN     string      `env:"POSTGRES_DSN" env-required:"true"`
	NATSURL         string      `env:"NATS_URL" env-default:"nats://nats:4222"`
	NATSUser        string      `env:"NATS_USER"`
	NATSPass        string      `env:"NATS_PASSWORD"`
	RoomServiceAddr string      `env:"ROOM_SERVICE_ADDR" env-default:"room:50053"`
	RoomCallTimeout string      `env:"ROOM_CALL_TIMEOUT" env-default:"3s"`
}

type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvProduction  Environment = "production"
)

func Load() (*Config, error) {
	cfg := &Config{}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, fmt.Errorf("read env: %w", err)
	}
	return cfg, nil
}

func (c *Config) ParseRoomCallTimeout() (time.Duration, error) {
	return time.ParseDuration(c.RoomCallTimeout)
}
