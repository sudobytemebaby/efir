package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"

	sharedcfg "github.com/sudobytemebaby/efir/services/shared/pkg/config"
)

type Config struct {
	Env      sharedcfg.Environment `yaml:"env"      env:"ENV"`
	LogLevel string                `yaml:"log_level" env:"LOG_LEVEL"`

	Server struct {
		Port        string `yaml:"port"          env:"GATEWAY_PORT"`
		MaxBodySize int64  `yaml:"max_body_size" env:"MAX_BODY_SIZE"`
	} `yaml:"server"`

	Timeouts struct {
		Shutdown   time.Duration `yaml:"shutdown"    env:"SHUTDOWN_TIMEOUT"`
		ReadHeader time.Duration `yaml:"read_header" env:"READ_HEADER_TIMEOUT"`
		GRPC       time.Duration `yaml:"grpc"        env:"GRPC_TIMEOUT"`
	} `yaml:"timeouts"`

	Cache struct {
		Addr string `yaml:"addr" env:"VALKEY_ADDR"`
		Pass string `yaml:"-"   env:"VALKEY_PASSWORD"`
	} `yaml:"cache"`

	Auth struct {
		Secret      string        `yaml:"-"            env:"JWT_SECRET"`
		WSTicketTTL time.Duration `yaml:"ws_ticket_ttl" env:"WS_TICKET_TTL"`
	} `yaml:"auth"`

	RateLimit struct {
		Requests int           `yaml:"requests" env:"RATE_LIMIT_REQUESTS"`
		Window   time.Duration `yaml:"window"   env:"RATE_LIMIT_WINDOW"`
	} `yaml:"rate_limit"`

	Services struct {
		Auth    string `yaml:"auth"    env:"AUTH_SERVICE_ADDR"`
		User    string `yaml:"user"    env:"USER_SERVICE_ADDR"`
		Room    string `yaml:"room"    env:"ROOM_SERVICE_ADDR"`
		Message string `yaml:"message" env:"MESSAGE_SERVICE_ADDR"`
	} `yaml:"services"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if cfg.Timeouts.Shutdown == 0 {
		cfg.Timeouts.Shutdown = 15 * time.Second
	}
	if cfg.Timeouts.ReadHeader == 0 {
		cfg.Timeouts.ReadHeader = 5 * time.Second
	}
	if cfg.Timeouts.GRPC == 0 {
		cfg.Timeouts.GRPC = 10 * time.Second
	}
	if cfg.RateLimit.Requests == 0 {
		cfg.RateLimit.Requests = 100
	}
	if cfg.RateLimit.Window == 0 {
		cfg.RateLimit.Window = 60 * time.Second
	}
	if err := cfg.Env.Validate(); err != nil {
		return nil, fmt.Errorf("invalid environment: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.RateLimit.Requests <= 0 {
		return errors.New("RATE_LIMIT_REQUESTS must be positive")
	}
	if c.RateLimit.Window < time.Second {
		return errors.New("RATE_LIMIT_WINDOW must be at least 1 second")
	}
	return nil
}
