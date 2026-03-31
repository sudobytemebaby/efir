//go:build integration

package valkey_test

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vk "github.com/valkey-io/valkey-go"

	"github.com/sudobytemebaby/efir/services/shared/pkg/testutil"
	"github.com/sudobytemebaby/efir/services/shared/pkg/valkey"
)

var valkeyContainer *testutil.ValkeyContainer

func TestMain(m *testing.M) {
	ctx := context.Background()
	valkeyContainer = testutil.NewValkeyContainer(ctx)
	defer valkeyContainer.Terminate(ctx) //nolint:errcheck
	os.Exit(m.Run())
}

func TestIncrWithExpiryScript(t *testing.T) {
	client := valkeyContainer.Client(t)
	ctx := context.Background()

	key := "test:incr-with-expiry:" + strconv.FormatInt(time.Now().UnixNano(), 36)
	ttlSeconds := "2"

	eval := func(k, ttl string) int64 {
		t.Helper()
		result, err := client.Do(ctx,
			client.B().Eval().Script(valkey.IncrWithExpiryScript).Numkeys(1).Key(k).Arg(ttl).Build(),
		).AsInt64()
		require.NoError(t, err)
		return result
	}

	t.Run("first call returns 1 and sets TTL", func(t *testing.T) {
		v := eval(key, ttlSeconds)
		assert.EqualValues(t, 1, v, "first increment should return 1")

		ttl, err := client.Do(ctx, client.B().Ttl().Key(key).Build()).AsInt64()
		require.NoError(t, err)
		assert.Positive(t, ttl, "TTL should be set after first increment")
	})

	t.Run("subsequent calls increment without resetting TTL", func(t *testing.T) {
		v2 := eval(key, ttlSeconds)
		assert.EqualValues(t, 2, v2, "second increment should return 2")

		v3 := eval(key, ttlSeconds)
		assert.EqualValues(t, 3, v3, "third increment should return 3")

		ttl, err := client.Do(ctx, client.B().Ttl().Key(key).Build()).AsInt64()
		require.NoError(t, err)
		// TTL was set on first call only — it should still be positive (not reset to 2).
		assert.Positive(t, ttl)
	})

	t.Run("key expires after TTL", func(t *testing.T) {
		expKey := "test:incr-expire:" + strconv.FormatInt(time.Now().UnixNano(), 36)
		eval(expKey, "1") // TTL = 1 second

		time.Sleep(1500 * time.Millisecond)

		err := client.Do(ctx, client.B().Get().Key(expKey).Build()).Error()
		assert.True(t, vk.IsValkeyNil(err), "key should have expired")
	})
}

func TestGetAndDeleteScript(t *testing.T) {
	client := valkeyContainer.Client(t)
	ctx := context.Background()

	t.Run("returns value and deletes key atomically", func(t *testing.T) {
		key := "test:get-del:" + strconv.FormatInt(time.Now().UnixNano(), 36)
		expectedValue := "user-123"

		err := client.Do(ctx, client.B().Set().Key(key).Value(expectedValue).Build()).Error()
		require.NoError(t, err)

		result, err := client.Do(ctx,
			client.B().Eval().Script(valkey.GetAndDeleteScript).Numkeys(1).Key(key).Build(),
		).ToString()
		require.NoError(t, err)
		assert.Equal(t, expectedValue, result)

		err = client.Do(ctx, client.B().Get().Key(key).Build()).Error()
		assert.True(t, vk.IsValkeyNil(err), "key should be deleted after get-and-delete")
	})

	t.Run("returns nil for non-existent key", func(t *testing.T) {
		key := "test:get-del:nonexistent:" + strconv.FormatInt(time.Now().UnixNano(), 36)

		resp := client.Do(ctx,
			client.B().Eval().Script(valkey.GetAndDeleteScript).Numkeys(1).Key(key).Build(),
		)
		err := resp.Error()
		assert.True(t, vk.IsValkeyNil(err), "error should be valkey nil for non-existent key")
	})
}
