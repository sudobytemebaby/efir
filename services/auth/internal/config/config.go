package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string `env:"ENV"       env-default:"development"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
	GRPCPort    string `env:"GRPC_PORT" env-default:"50051"`
	PostgresDSN string `env:"POSTGRES_DSN" env-required:"true"`
	ValkeyAddr  string `env:"VALKEY_ADDR" env-default:"valkey:6379"`
	ValkeyPass  string `env:"VALKEY_PASSWORD"`
	NATSURL     string `env:"NATS_URL" env-default:"nats://nats:4222"`
	JWTSecret   string `env:"JWT_SECRET" env-required:"true"`
	AccessTTL   string `env:"JWT_ACCESS_TTL" env-default:"15m"`
	RefreshTTL  string `env:"JWT_REFRESH_TTL" env-default:"168h"`
	NATSUser    string `env:"NATS_USER"`
	NATSPass    string `env:"NATS_PASSWORD"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	err := cleanenv.ReadEnv(cfg)
	if err != nil {
		return nil, fmt.Errorf("read env: %w", err)
	}

	return cfg, nil
}

func (c *Config) ParseAccessTTL() (time.Duration, error) {
	return time.ParseDuration(c.AccessTTL)
}

func (c *Config) ParseRefreshTTL() (time.Duration, error) {
	return time.ParseDuration(c.RefreshTTL)
}
