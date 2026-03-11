package valkey

import (
	"testing"
)

func TestAuthRefreshKey(t *testing.T) {
	key := AuthRefreshKey("token-123")
	expected := "auth:refresh:token-123"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestGatewayRateLimitKey(t *testing.T) {
	key := GatewayRateLimitKey("user", "user-123", "60s")
	expected := "gateway:ratelimit:user:user-123:60s"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestWSPubsubChannelKey(t *testing.T) {
	key := WSPubsubChannelKey("room-123")
	expected := "ws:pubsub:room-123"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestPresenceOnlineKey(t *testing.T) {
	key := PresenceOnlineKey("user-456")
	expected := "presence:online:user-456"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestConstantsFormat(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"AuthRefreshToken", AuthRefreshToken, "auth:refresh:%s"},
		{"GatewayRateLimit", GatewayRateLimit, "gateway:ratelimit:%s:%s:%s"},
		{"WSPubsubChannel", WSPubsubChannel, "ws:pubsub:%s"},
		{"PresenceOnline", PresenceOnline, "presence:online:%s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.constant)
			}
		})
	}
}
