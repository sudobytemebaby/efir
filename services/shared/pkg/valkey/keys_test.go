//go:build integration

package valkey_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sudobytemebaby/efir/services/shared/pkg/valkey"
)

func TestKeyFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		got      string
		wantPfx  string // key must start with this prefix
		wantSfx  string // key must end with this suffix
	}{
		{
			name:    "AuthRefreshKey has correct format",
			got:     valkey.AuthRefreshKey("tok123"),
			wantPfx: "auth:refresh:",
			wantSfx: "tok123",
		},
		{
			name:    "AuthRateLimitKey has correct format",
			got:     valkey.AuthRateLimitKey("login", "user@test.com"),
			wantPfx: "auth:ratelimit:login:",
			wantSfx: "user@test.com",
		},
		{
			name:    "GatewayRateLimitKey has correct format",
			got:     valkey.GatewayRateLimitKey("ip", "1.2.3.4", "60"),
			wantPfx: "gateway:ratelimit:ip:",
			wantSfx: "60",
		},
		{
			name:    "WSPubsubChannelKey has correct format",
			got:     valkey.WSPubsubChannelKey("room-abc"),
			wantPfx: "ws:pubsub:",
			wantSfx: "room-abc",
		},
		{
			name:    "PresenceOnlineKey has correct format",
			got:     valkey.PresenceOnlineKey("user-xyz"),
			wantPfx: "presence:online:",
			wantSfx: "user-xyz",
		},
		{
			name:    "GatewayWSTicketKey has correct format",
			got:     valkey.GatewayWSTicketKey("ticket-789"),
			wantPfx: "gateway:ws:ticket:",
			wantSfx: "ticket-789",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.True(t, strings.HasPrefix(tc.got, tc.wantPfx),
				"key %q should start with %q", tc.got, tc.wantPfx)
			assert.True(t, strings.HasSuffix(tc.got, tc.wantSfx),
				"key %q should end with %q", tc.got, tc.wantSfx)
		})
	}

	t.Run("no key collisions between different types", func(t *testing.T) {
		t.Parallel()
		seen := map[string]string{}
		keys := map[string]string{
			"AuthRefreshKey":      valkey.AuthRefreshKey("x"),
			"AuthRateLimitKey":    valkey.AuthRateLimitKey("login", "x"),
			"GatewayRateLimitKey": valkey.GatewayRateLimitKey("ip", "x", "x"),
			"WSPubsubChannelKey":  valkey.WSPubsubChannelKey("x"),
			"PresenceOnlineKey":   valkey.PresenceOnlineKey("x"),
			"GatewayWSTicketKey":  valkey.GatewayWSTicketKey("x"),
		}
		for fn, key := range keys {
			if existing, ok := seen[key]; ok {
				t.Errorf("key collision: %s and %s produce the same key %q", fn, existing, key)
			}
			seen[key] = fn
		}
	})
}
