// Package config provides configuration loading from environment variables.
package config

import (
	"errors"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	GRPCPort    string `env:"GRPC_PORT" env-default:"50051"`
	PostgresDSN string `env:"POSTGRES_DSN" env-required:"true"`
	ValkeyAddr  string `env:"VALKEY_ADDR" env-default:"valkey:6379"`
	ValkeyPass  string `env:"VALKEY_PASSWORD"`
	NATSURL     string `env:"NATS_URL" env-default:"nats://nats:4222"`
	JWTSecret   string `env:"JWT_SECRET" env-required:"true"`
	AccessTTL   string `env:"JWT_ACCESS_TTL" env-default:"15m"`
	RefreshTTL  string `env:"JWT_REFRESH_TTL" env-default:"168h"`
}

func Load(cfg *Config) error {
	err := cleanenv.ReadEnv(cfg)
	if err != nil {
		return errors.New("failed to read config: " + err.Error())
	}

	return nil
}
