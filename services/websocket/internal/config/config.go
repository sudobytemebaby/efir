package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env        Environment `env:"ENV"               env-default:"development"`
	LogLevel   string      `env:"LOG_LEVEL"         env-default:"info"`
	Port       string      `env:"WEBSOCKET_PORT"     env-default:"8081"`
	GatewayURL string      `env:"GATEWAY_URL"        env-default:"http://gateway:8080"`
	ValkeyAddr string      `env:"VALKEY_ADDR"        env-default:"valkey:6379"`
	ValkeyPass string      `env:"VALKEY_PASSWORD"`
	NATSURL    string      `env:"NATS_URL"           env-default:"nats://nats:4222"`
	NATSUser   string      `env:"NATS_USER"`
	NATSPass   string      `env:"NATS_PASSWORD"`
	WriteLimit int64       `env:"WRITE_LIMIT"        env-default:"4096"`
	ReadLimit  int64       `env:"READ_LIMIT"         env-default:"4096"`
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

func (c *Config) WriteLimitBytes() int64 {
	if c.WriteLimit <= 0 {
		return 4096
	}
	return c.WriteLimit
}

func (c *Config) ReadLimitBytes() int64 {
	if c.ReadLimit <= 0 {
		return 4096
	}
	return c.ReadLimit
}

func (c *Config) PingInterval() time.Duration {
	return 30 * time.Second
}

func (c *Config) ReadDeadline() time.Duration {
	return 60 * time.Second
}
