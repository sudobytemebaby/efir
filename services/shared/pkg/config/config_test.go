package config

import (
	"os"
	"testing"
)

func TestLoadWithDefaults(t *testing.T) {
	_ = os.Setenv("POSTGRES_DSN", "postgres://localhost:5432/db")
	_ = os.Setenv("JWT_SECRET", "secret123")
	defer func() {
		_ = os.Unsetenv("POSTGRES_DSN")
		_ = os.Unsetenv("JWT_SECRET")
	}()

	cfg := &Config{}
	err := Load(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GRPCPort != "50051" {
		t.Errorf("expected default GRPC_PORT 50051, got %s", cfg.GRPCPort)
	}

	if cfg.ValkeyAddr != "valkey:6379" {
		t.Errorf("expected default ValkeyAddr valkey:6379, got %s", cfg.ValkeyAddr)
	}

	if cfg.PostgresDSN != "postgres://localhost:5432/db" {
		t.Errorf("expected PostgresDSN from env, got %s", cfg.PostgresDSN)
	}
}

func TestLoadMissingRequired(t *testing.T) {
	_ = os.Unsetenv("POSTGRES_DSN")
	_ = os.Unsetenv("JWT_SECRET")

	cfg := &Config{}
	err := Load(cfg)

	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
}

func TestLoadWithAllFields(t *testing.T) {
	_ = os.Setenv("GRPC_PORT", "50052")
	_ = os.Setenv("POSTGRES_DSN", "postgres://localhost:5432/mydb")
	_ = os.Setenv("VALKEY_ADDR", "valkey:6380")
	_ = os.Setenv("VALKEY_PASSWORD", "mypassword")
	_ = os.Setenv("NATS_URL", "nats://localhost:4222")
	_ = os.Setenv("JWT_SECRET", "myjwtsecret")
	_ = os.Setenv("JWT_ACCESS_TTL", "30m")
	_ = os.Setenv("JWT_REFRESH_TTL", "336h")
	defer func() {
		_ = os.Unsetenv("GRPC_PORT")
		_ = os.Unsetenv("POSTGRES_DSN")
		_ = os.Unsetenv("VALKEY_ADDR")
		_ = os.Unsetenv("VALKEY_PASSWORD")
		_ = os.Unsetenv("NATS_URL")
		_ = os.Unsetenv("JWT_SECRET")
		_ = os.Unsetenv("JWT_ACCESS_TTL")
		_ = os.Unsetenv("JWT_REFRESH_TTL")
	}()

	cfg := &Config{}
	err := Load(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GRPCPort != "50052" {
		t.Errorf("expected GRPC_PORT 50052, got %s", cfg.GRPCPort)
	}
	if cfg.PostgresDSN != "postgres://localhost:5432/mydb" {
		t.Errorf("expected PostgresDSN postgres://localhost:5432/mydb, got %s", cfg.PostgresDSN)
	}
	if cfg.ValkeyAddr != "valkey:6380" {
		t.Errorf("expected ValkeyAddr valkey:6380, got %s", cfg.ValkeyAddr)
	}
	if cfg.ValkeyPass != "mypassword" {
		t.Errorf("expected ValkeyPass mypassword, got %s", cfg.ValkeyPass)
	}
	if cfg.NATSURL != "nats://localhost:4222" {
		t.Errorf("expected NATSURL nats://localhost:4222, got %s", cfg.NATSURL)
	}
	if cfg.JWTSecret != "myjwtsecret" {
		t.Errorf("expected JWTSecret myjwtsecret, got %s", cfg.JWTSecret)
	}
	if cfg.AccessTTL != "30m" {
		t.Errorf("expected AccessTTL 30m, got %s", cfg.AccessTTL)
	}
	if cfg.RefreshTTL != "336h" {
		t.Errorf("expected RefreshTTL 336h, got %s", cfg.RefreshTTL)
	}
}
