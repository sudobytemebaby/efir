package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sharedcfg "github.com/sudobytemebaby/efir/services/shared/pkg/config"
)

func TestLoad(t *testing.T) {
	env := map[string]string{
		"ENV":                   "development",
		"LOG_LEVEL":             "debug",
		"GRPC_PORT":             "50053",
		"HEALTH_LISTEN_ADDR":    ":8080",
		"SHUTDOWN_TIMEOUT":      "30s",
		"READ_HEADER_TIMEOUT":   "5s",
		"GRPC_GRACEFUL_TIMEOUT": "5s",
		"NATS_URL":              "nats://localhost:4222",
		"NATS_RECONNECT_WAIT":   "2s",
		"NATS_MAX_RECONNECTS":   "10",
	}

	for k, v := range env {
		t.Setenv(k, v)
	}

	tmp, err := os.CreateTemp("", "room_config_*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmp.Name()) }()
	_, err = tmp.WriteString("env: development\n")
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	cfg, err := Load(tmp.Name())
	require.NoError(t, err)

	assert.Equal(t, sharedcfg.Environment("development"), cfg.Env)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "50053", cfg.Server.GRPCPort)
	assert.Equal(t, ":8080", cfg.Server.HealthListenAddr)
	assert.Equal(t, 30*time.Second, cfg.Timeouts.Shutdown)
	assert.Equal(t, 5*time.Second, cfg.Timeouts.ReadHeader)
	assert.Equal(t, 5*time.Second, cfg.Timeouts.GRPCGraceful)
	assert.Equal(t, "nats://localhost:4222", cfg.NATS.URL)
	assert.Equal(t, 2*time.Second, cfg.NATS.ReconnectWait)
	assert.Equal(t, 10, cfg.NATS.MaxReconnects)
}

func TestLoad_Defaults(t *testing.T) {
	env := map[string]string{
		"LOG_LEVEL": "info",
	}
	for k, v := range env {
		t.Setenv(k, v)
	}

	tmp, err := os.CreateTemp("", "room_defaults_*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmp.Name()) }()
	_, err = tmp.WriteString("env: development\n")
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	cfg, err := Load(tmp.Name())
	require.NoError(t, err)

	assert.Equal(t, sharedcfg.Environment("development"), cfg.Env)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmp, err := os.CreateTemp("", "room_invalid_*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmp.Name()) }()
	_, err = tmp.WriteString("invalid yaml content\n")
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	cfg, err := Load(tmp.Name())
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_InvalidEnvironment(t *testing.T) {
	env := map[string]string{
		"LOG_LEVEL": "info",
	}
	for k, v := range env {
		t.Setenv(k, v)
	}

	tmp, err := os.CreateTemp("", "room_env_*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmp.Name()) }()
	_, err = tmp.WriteString("env: invalid\n")
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	cfg, err := Load(tmp.Name())
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestEnvironment_Validate(t *testing.T) {
	tests := []struct {
		name    string
		env     sharedcfg.Environment
		wantErr bool
	}{
		{"development", sharedcfg.EnvDevelopment, false},
		{"production", sharedcfg.EnvProduction, false},
		{"invalid", sharedcfg.Environment("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.env.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
