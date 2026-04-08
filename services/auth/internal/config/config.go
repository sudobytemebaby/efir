package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"

	sharedcfg "github.com/sudobytemebaby/efir/services/shared/pkg/config"
)

type Config struct {
	Env      sharedcfg.Environment    `yaml:"env"      env:"ENV"`
	LogLevel string                   `yaml:"log_level" env:"LOG_LEVEL"`
	Server   sharedcfg.ServerConfig   `yaml:"server"`
	Timeouts sharedcfg.TimeoutsConfig `yaml:"timeouts"`
	NATS     sharedcfg.NATSConfig     `yaml:"nats"`

	Database struct {
		DSN string `yaml:"-" env:"POSTGRES_DSN"`
	} `yaml:"database"`

	Cache struct {
		Addr string `yaml:"addr" env:"VALKEY_ADDR"`
		Pass string `yaml:"-"   env:"VALKEY_PASSWORD"`
	} `yaml:"cache"`

	Auth struct {
		Secret     string        `yaml:"-"          env:"JWT_SECRET"`
		AccessTTL  time.Duration `yaml:"access_ttl"  env:"JWT_ACCESS_TTL"`
		RefreshTTL time.Duration `yaml:"refresh_ttl" env:"JWT_REFRESH_TTL"`
	} `yaml:"auth"`

	RateLimit struct {
		Requests int           `yaml:"requests" env:"RATE_LIMIT_REQUESTS"`
		Window   time.Duration `yaml:"window"   env:"RATE_LIMIT_WINDOW"`
	} `yaml:"rate_limit"`
}

func (c *Config) Validate() error {
	if len(c.Auth.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 bytes, got %d", len(c.Auth.Secret))
	}
	if c.Auth.AccessTTL <= 0 {
		return errors.New("JWT_ACCESS_TTL must be positive")
	}
	if c.Auth.RefreshTTL <= 0 {
		return errors.New("JWT_REFRESH_TTL must be positive")
	}
	if c.RateLimit.Requests <= 0 {
		return errors.New("RATE_LIMIT_REQUESTS must be positive")
	}
	if c.RateLimit.Window < time.Second {
		return errors.New("RATE_LIMIT_WINDOW must be at least 1 second")
	}
	return nil
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	cfg.Timeouts.WithDefaults()
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
