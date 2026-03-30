// Package testutil provides shared test infrastructure for integration tests.
// It is intended to be imported only from _test.go files.
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // register "pgx" driver for database/sql
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer manages a single testcontainers Postgres instance shared
// across an entire test binary. Each call to Pool creates a fresh schema with
// migrations applied, providing per-test isolation without the cost of
// restarting the container.
//
// Typical usage in a test package:
//
//	var pgContainer *testutil.PostgresContainer
//
//	func TestMain(m *testing.M) {
//	    ctx := context.Background()
//	    pgContainer = testutil.NewPostgresContainer(ctx, "../../migrations")
//	    defer pgContainer.Terminate(ctx)
//	    os.Exit(m.Run())
//	}
//
//	func TestFoo(t *testing.T) {
//	    pool := pgContainer.Pool(t)
//	    // pool is connected to a schema with all migrations applied.
//	    // The schema is dropped automatically when the test finishes.
//	}
type PostgresContainer struct {
	container     *tcpostgres.PostgresContainer
	connStr       string
	migrationsDir string
}

// NewPostgresContainer starts a Postgres container and returns a helper that
// creates per-test schemas. migrationsDir is the path to the goose SQL
// migration files for the service under test (e.g. "../../migrations").
// Panics on error — this is called from TestMain where panic is appropriate.
func NewPostgresContainer(ctx context.Context, migrationsDir string) *PostgresContainer {
	ctr, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("testutil: start postgres container: %v", err))
	}

	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(fmt.Sprintf("testutil: get postgres connection string: %v", err))
	}

	return &PostgresContainer{
		container:     ctr,
		connStr:       connStr,
		migrationsDir: migrationsDir,
	}
}

// Pool creates a pgxpool.Pool connected to a fresh, migration-applied schema.
// The schema is automatically dropped when t finishes.
//
// Tests using Pool are safe to run in parallel — each gets its own schema.
func (c *PostgresContainer) Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	// Unique schema per test prevents parallel interference.
	schema := "test_" + strings.ReplaceAll(RandomUUID().String()[:8], "-", "")

	// Create schema via a short-lived admin connection.
	adminPool, err := pgxpool.New(ctx, c.connStr)
	require.NoError(t, err)
	defer adminPool.Close()

	_, err = adminPool.Exec(ctx, "CREATE SCHEMA "+schema)
	require.NoError(t, err)

	// Run goose migrations inside the new schema.
	schemaConnStr := c.connStr + "&search_path=" + schema
	db, err := sql.Open("pgx", schemaConnStr)
	require.NoError(t, err)

	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS(c.migrationsDir))
	require.NoError(t, err)
	_, err = provider.Up(ctx)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	// Create the pool the test will use, scoped to the schema.
	pool, err := pgxpool.New(ctx, schemaConnStr)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
		// Best-effort schema cleanup — ignore errors on teardown.
		adminPool2, cleanErr := pgxpool.New(ctx, c.connStr)
		if cleanErr == nil {
			adminPool2.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE") //nolint:errcheck,gosec
			adminPool2.Close()
		}
	})

	return pool
}

// Terminate stops and removes the container. Call from TestMain's defer.
func (c *PostgresContainer) Terminate(ctx context.Context) error {
	return c.container.Terminate(ctx)
}
