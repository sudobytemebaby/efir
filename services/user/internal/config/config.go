package config

import (
	"fmt"

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
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := cfg.Env.Validate(); err != nil {
		return nil, fmt.Errorf("invalid environment: %w", err)
	}
	return cfg, nil
}
