package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"

	sharedcfg "github.com/sudobytemebaby/efir/services/shared/pkg/config"
)

type Config struct {
	Env      sharedcfg.Environment `yaml:"env"      env:"ENV"`
	LogLevel string                `yaml:"log_level" env:"LOG_LEVEL"`
	NATS     sharedcfg.NATSConfig  `yaml:"nats"`

	Server struct {
		Port          string `yaml:"port"            env:"WEBSOCKET_PORT"`
		HubBufferSize int    `yaml:"hub_buffer_size" env:"HUB_BUFFER_SIZE"`
	} `yaml:"server"`

	Timeouts struct {
		Shutdown   time.Duration `yaml:"shutdown"    env:"SHUTDOWN_TIMEOUT"`
		ReadHeader time.Duration `yaml:"read_header" env:"READ_HEADER_TIMEOUT"`
	} `yaml:"timeouts"`

	Cache struct {
		Addr string `yaml:"addr" env:"VALKEY_ADDR"`
		Pass string `yaml:"-"   env:"VALKEY_PASSWORD"`
	} `yaml:"cache"`

	Services struct {
		GatewayURL string `yaml:"gateway_url" env:"GATEWAY_URL"`
	} `yaml:"services"`

	WebSocket struct {
		WriteLimit    int64         `yaml:"write_limit"    env:"WRITE_LIMIT"`
		ReadLimit     int64         `yaml:"read_limit"     env:"READ_LIMIT"`
		PingInterval  time.Duration `yaml:"ping_interval"  env:"PING_INTERVAL"`
		ReadDeadline  time.Duration `yaml:"read_deadline"  env:"READ_DEADLINE"`
		WriteDeadline time.Duration `yaml:"write_deadline" env:"WRITE_DEADLINE"`
		WriteBuffer   int           `yaml:"write_buffer"   env:"WRITE_BUFFER"`
	} `yaml:"websocket"`
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
