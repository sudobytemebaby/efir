package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         Environment `env:"ENV"             env-default:"development"`
	LogLevel    string      `env:"LOG_LEVEL"       env-default:"info"`
	Port        string      `env:"GATEWAY_PORT"    env-default:"8080"`
	ValkeyAddr  string      `env:"VALKEY_ADDR"     env-default:"valkey:6379"`
	ValkeyPass  string      `env:"VALKEY_PASSWORD"`
	WSTicketTTL string      `env:"WS_TICKET_TTL"   env-default:"30s"`

	AuthServiceAddr    string `env:"AUTH_SERVICE_ADDR"    env-default:"auth:50051"`
	UserServiceAddr    string `env:"USER_SERVICE_ADDR"     env-default:"user:50052"`
	RoomServiceAddr    string `env:"ROOM_SERVICE_ADDR"     env-default:"room:50053"`
	MessageServiceAddr string `env:"MESSAGE_SERVICE_ADDR" env-default:"message:50054"`

	JWTSecret string `env:"JWT_SECRET" env-required:"true"`

	GRPCTimeout       string `env:"GRPC_TIMEOUT"    env-default:"5s"`
	RateLimitRequests int    `env:"RATE_LIMIT_REQUESTS" env-default:"100"`
	RateLimitWindow   string `env:"RATE_LIMIT_WINDOW"   env-default:"1m"`
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

func (c *Config) ParseGRPCTimeout() (time.Duration, error) {
	return time.ParseDuration(c.GRPCTimeout)
}

func (c *Config) ParseRateLimitWindow() (time.Duration, error) {
	return time.ParseDuration(c.RateLimitWindow)
}
