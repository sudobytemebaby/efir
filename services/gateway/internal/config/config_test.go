package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, Environment("development"), cfg.Env)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "localhost:6379", cfg.ValkeyAddr)
	assert.Equal(t, "testpass", cfg.ValkeyPass)
	assert.Equal(t, 60*time.Second, cfg.WSTicketTTL)
	assert.Equal(t, "auth:50051", cfg.AuthServiceAddr)
	assert.Equal(t, "user:50052", cfg.UserServiceAddr)
	assert.Equal(t, "room:50053", cfg.RoomServiceAddr)
	assert.Equal(t, "message:50054", cfg.MessageServiceAddr)
	assert.Equal(t, "test-secret-key", cfg.JWTSecret)
	assert.Equal(t, 10*time.Second, cfg.GRPCTimeout)
	assert.Equal(t, 50, cfg.RateLimitRequests)
	assert.Equal(t, 2*time.Minute, cfg.RateLimitWindow)
}

func TestLoad_Defaults(t *testing.T) {
	env := map[string]string{
		"JWT_SECRET": "test-secret",
	}

	for k, v := range env {
		t.Setenv(k, v)
	}

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, Environment("development"), cfg.Env)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "valkey:6379", cfg.ValkeyAddr)
	assert.Equal(t, 30*time.Second, cfg.WSTicketTTL)
	assert.Equal(t, "auth:50051", cfg.AuthServiceAddr)
	assert.Equal(t, 5*time.Second, cfg.GRPCTimeout)
	assert.Equal(t, 100, cfg.RateLimitRequests)
	assert.Equal(t, 1*time.Minute, cfg.RateLimitWindow)
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_InvalidDuration(t *testing.T) {
	env := map[string]string{
		"JWT_SECRET":    "test-secret",
		"WS_TICKET_TTL": "invalid",
	}

	for k, v := range env {
		t.Setenv(k, v)
	}

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestEnvironment_Validate(t *testing.T) {
	tests := []struct {
		name    string
		env     Environment
		wantErr bool
	}{
		{"development", EnvDevelopment, false},
		{"production", EnvProduction, false},
		{"invalid", Environment("invalid"), true},
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
