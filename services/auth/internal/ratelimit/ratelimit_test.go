//go:build integration

package ratelimit_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudobytemebaby/efir/services/auth/internal/ratelimit"
	"github.com/sudobytemebaby/efir/services/shared/pkg/testutil"
)

var valkeyContainer *testutil.ValkeyContainer

func TestMain(m *testing.M) {
	ctx := context.Background()
	valkeyContainer = testutil.NewValkeyContainer(ctx)
	exitCode := m.Run()
	_ = valkeyContainer.Terminate(ctx)
	os.Exit(exitCode)
}

func TestAllow(t *testing.T) {
	ctx := context.Background()

	t.Run("allows requests within limit", func(t *testing.T) {
		client := valkeyContainer.Client(t)
		limiter := ratelimit.NewValkeyLimiter(client, 3, 10*time.Second)
		email := testutil.RandomEmail()

		for i := 0; i < 3; i++ {
			err := limiter.Allow(ctx, ratelimit.ActionLogin, email)
			require.NoError(t, err, "request %d should be allowed", i+1)
		}
	})

	t.Run("blocks request exceeding limit", func(t *testing.T) {
		client := valkeyContainer.Client(t)
		limiter := ratelimit.NewValkeyLimiter(client, 2, 10*time.Second)
		email := testutil.RandomEmail()

		require.NoError(t, limiter.Allow(ctx, ratelimit.ActionLogin, email))
		require.NoError(t, limiter.Allow(ctx, ratelimit.ActionLogin, email))

		err := limiter.Allow(ctx, ratelimit.ActionLogin, email)
		require.Error(t, err)

		var rateLimitErr *ratelimit.ErrRateLimitExceeded
		assert.ErrorAs(t, err, &rateLimitErr)
		assert.Equal(t, ratelimit.ActionLogin, rateLimitErr.Action)
		assert.Equal(t, email, rateLimitErr.Email)
	})

	t.Run("register and login use independent counters", func(t *testing.T) {
		client := valkeyContainer.Client(t)
		limiter := ratelimit.NewValkeyLimiter(client, 1, 10*time.Second)
		email := testutil.RandomEmail()

		require.NoError(t, limiter.Allow(ctx, ratelimit.ActionLogin, email))
		// Login counter is now at the limit; register counter is independent.
		require.NoError(t, limiter.Allow(ctx, ratelimit.ActionRegister, email))
	})

	t.Run("counter resets after TTL", func(t *testing.T) {
		client := valkeyContainer.Client(t)
		// Use 1-second window so the test doesn't take long.
		limiter := ratelimit.NewValkeyLimiter(client, 1, time.Second)
		email := testutil.RandomEmail()

		require.NoError(t, limiter.Allow(ctx, ratelimit.ActionLogin, email))
		var rateLimitErr *ratelimit.ErrRateLimitExceeded
		err := limiter.Allow(ctx, ratelimit.ActionLogin, email)
		require.ErrorAs(t, err, &rateLimitErr)

		time.Sleep(1100 * time.Millisecond)

		// Counter should have expired; request is allowed again.
		require.NoError(t, limiter.Allow(ctx, ratelimit.ActionLogin, email))
	})
}
