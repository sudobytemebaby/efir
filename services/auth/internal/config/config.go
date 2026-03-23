package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env               Environment   `env:"ENV"                  env-default:"development"`
	LogLevel          string        `env:"LOG_LEVEL"            env-default:"info"`
	GRPCPort          string        `env:"GRPC_PORT"            env-default:"50051"`
	PostgresDSN       string        `env:"POSTGRES_DSN"         env-required:"true"`
	ValkeyAddr        string        `env:"VALKEY_ADDR"          env-default:"valkey:6379"`
	ValkeyPass        string        `env:"VALKEY_PASSWORD"`
	NATSURL           string        `env:"NATS_URL"             env-default:"nats://nats:4222"`
	JWTSecret         string        `env:"JWT_SECRET"           env-required:"true"`
	AccessTTL         time.Duration `env:"JWT_ACCESS_TTL"     env-default:"15m"`
	RefreshTTL        time.Duration `env:"JWT_REFRESH_TTL"    env-default:"168h"`
	NATSUser          string        `env:"NATS_USER"`
	NATSPass          string        `env:"NATS_PASSWORD"`
	RateLimitRequests int64         `env:"RATE_LIMIT_REQUESTS" env-default:"10"`
	RateLimitWindow   time.Duration `env:"RATE_LIMIT_WINDOW"  env-default:"1m"`
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
	if err := cfg.Env.Validate(); err != nil {
		return nil, fmt.Errorf("invalid environment: %w", err)
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
