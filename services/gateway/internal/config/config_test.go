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
		"ENV":                  "development",
		"LOG_LEVEL":            "debug",
		"GATEWAY_PORT":         "9090",
		"VALKEY_ADDR":          "localhost:6379",
		"VALKEY_PASSWORD":      "testpass",
		"WS_TICKET_TTL":        "60s",
		"AUTH_SERVICE_ADDR":    "auth:50051",
		"USER_SERVICE_ADDR":    "user:50052",
		"ROOM_SERVICE_ADDR":    "room:50053",
		"MESSAGE_SERVICE_ADDR": "message:50054",
		"JWT_SECRET":           "test-secret-key",
		"GRPC_TIMEOUT":         "10s",
		"RATE_LIMIT_REQUESTS":  "50",
		"RATE_LIMIT_WINDOW":    "2m",
	}

	for k, v := range env {
		t.Setenv(k, v)
	}

	tmp, err := os.CreateTemp("", "gateway_config_*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmp.Name()) }()
	_, err = tmp.WriteString("env: development\n")
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	cfg, err := Load(tmp.Name())
	require.NoError(t, err)

	assert.Equal(t, sharedcfg.Environment("development"), cfg.Env)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "9090", cfg.Server.Port)
	assert.Equal(t, "localhost:6379", cfg.Cache.Addr)
	assert.Equal(t, "testpass", cfg.Cache.Pass)
	assert.Equal(t, 60*time.Second, cfg.Auth.WSTicketTTL)
	assert.Equal(t, "auth:50051", cfg.Services.Auth)
	assert.Equal(t, "user:50052", cfg.Services.User)
	assert.Equal(t, "room:50053", cfg.Services.Room)
	assert.Equal(t, "message:50054", cfg.Services.Message)
	assert.Equal(t, "test-secret-key", cfg.Auth.Secret)
	assert.Equal(t, 10*time.Second, cfg.Timeouts.GRPC)
	assert.Equal(t, 50, cfg.RateLimit.Requests)
	assert.Equal(t, 2*time.Minute, cfg.RateLimit.Window)
}

func TestLoad_Defaults(t *testing.T) {
	env := map[string]string{
		"LOG_LEVEL":  "info",
		"JWT_SECRET": "test-secret",
	}
	for k, v := range env {
		t.Setenv(k, v)
	}

	tmp, err := os.CreateTemp("", "gateway_defaults_*.yaml")
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

func TestLoad_MissingJWTSecret(t *testing.T) {
	_ = os.Unsetenv("JWT_SECRET")
	tmp, err := os.CreateTemp("", "gateway_missing_*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmp.Name()) }()
	_, err = tmp.WriteString("env: development\n")
	require.NoError(t, err)
	require.NoError(t, tmp.Close())
	cfg, err := Load(tmp.Name())
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "", cfg.Auth.Secret)
}

func TestLoad_InvalidDuration(t *testing.T) {
	env := map[string]string{
		"JWT_SECRET":    "test-secret",
		"WS_TICKET_TTL": "invalid",
	}

	for k, v := range env {
		t.Setenv(k, v)
	}

	tmp, err := os.CreateTemp("", "gateway_invalid_*.yaml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmp.Name()) }()
	_, err = tmp.WriteString("invalid yaml\n")
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
