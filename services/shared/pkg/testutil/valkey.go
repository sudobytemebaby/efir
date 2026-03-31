package testutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcvalkey "github.com/testcontainers/testcontainers-go/modules/valkey"
	"github.com/testcontainers/testcontainers-go/wait"
	vk "github.com/valkey-io/valkey-go"
)

// ValkeyContainer manages a single testcontainers Valkey instance.
// Each call to Client returns a client connected to the shared instance.
//
// IMPORTANT: Do not call t.Parallel() in tests that share a ValkeyContainer
// unless those tests operate on disjoint key spaces. The cleanup strategy is
// FLUSHDB, which clears all keys in the database after each test. Parallel
// tests would interfere with each other's cleanup.
//
// Typical usage:
//
//	var valkeyContainer *testutil.ValkeyContainer
//
//	func TestMain(m *testing.M) {
//	    ctx := context.Background()
//	    valkeyContainer = testutil.NewValkeyContainer(ctx)
//	    defer valkeyContainer.Terminate(ctx)
//	    os.Exit(m.Run())
//	}
//
//	func TestFoo(t *testing.T) {
//	    client := valkeyContainer.Client(t)
//	    // client is connected; all keys are flushed when the test finishes.
//	}
type ValkeyContainer struct {
	container *tcvalkey.ValkeyContainer
	addr      string
}

// NewValkeyContainer starts a Valkey container. Panics on error.
func NewValkeyContainer(ctx context.Context) *ValkeyContainer {
	ctr, err := tcvalkey.Run(ctx, "valkey/valkey:9-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections"),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("testutil: start valkey container: %v", err))
	}

	connStr, err := ctr.ConnectionString(ctx)
	if err != nil {
		panic(fmt.Sprintf("testutil: get valkey connection string: %v", err))
	}

	opts, err := vk.ParseURL(connStr)
	if err != nil {
		panic(fmt.Sprintf("testutil: parse valkey URL %q: %v", connStr, err))
	}

	// Extract host:port from the parsed options for later use.
	addr := opts.InitAddress[0]

	return &ValkeyContainer{
		container: ctr,
		addr:      addr,
	}
}

// Client creates a valkey-go client connected to the container.
// All keys in the database are flushed when t finishes.
func (c *ValkeyContainer) Client(t *testing.T) vk.Client {
	t.Helper()

	client, err := vk.NewClient(vk.ClientOption{
		InitAddress: []string{c.addr},
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		client.Do(context.Background(), client.B().Flushdb().Build()) //nolint:errcheck
		client.Close()
	})

	return client
}

// Terminate stops and removes the container. Call from TestMain's defer.
func (c *ValkeyContainer) Terminate(ctx context.Context) error {
	return c.container.Terminate(ctx)
}
