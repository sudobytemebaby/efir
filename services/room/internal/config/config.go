package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         Environment `env:"ENV"              env-default:"development"`
	LogLevel    string      `env:"LOG_LEVEL"        env-default:"info"`
	GRPCPort    string      `env:"GRPC_PORT"        env-default:"50053"`
	PostgresDSN string      `env:"POSTGRES_DSN"     env-required:"true"`
	NATSURL     string      `env:"NATS_URL"         env-default:"nats://nats:4222"`
	NATSUser    string      `env:"NATS_USER"`
	NATSPass    string      `env:"NATS_PASSWORD"`
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

func (e Environment) Validate() error {
	switch e {
	case EnvDevelopment, EnvProduction:
		return nil
	default:
		return fmt.Errorf("invalid environment %q, allowed: development, production", e)
	}
}
