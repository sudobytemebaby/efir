package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         Environment `env:"ENV"           env-default:"development"`
	LogLevel    string      `env:"LOG_LEVEL"     env-default:"info"`
	Port        string      `env:"GATEWAY_PORT"  env-default:"8080"`
	ValkeyAddr  string      `env:"VALKEY_ADDR"   env-default:"valkey:6379"`
	ValkeyPass  string      `env:"VALKEY_PASSWORD"`
	WSTicketTTL string      `env:"WS_TICKET_TTL" env-default:"30s"`
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

func (c *Config) ParseWSTicketTTL() (time.Duration, error) {
	return time.ParseDuration(c.WSTicketTTL)
}
