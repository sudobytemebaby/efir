//go:build integration

package repository_test

import (
	"context"
	"os"
	"testing"

	"github.com/sudobytemebaby/efir/services/shared/pkg/testutil"
)

var (
	pgContainer     *testutil.PostgresContainer
	valkeyContainer *testutil.ValkeyContainer
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer = testutil.NewPostgresContainer(ctx, "../../migrations")
	valkeyContainer = testutil.NewValkeyContainer(ctx)

	exitCode := m.Run()

	_ = pgContainer.Terminate(ctx)
	_ = valkeyContainer.Terminate(ctx)

	os.Exit(exitCode)
}
