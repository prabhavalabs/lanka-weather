// Package dbtest opens the disposable integration-test database named by
// LANKAWEATHER_TEST_DATABASE_URL, migrated and with the given tables
// emptied. Tests skip when the variable is not set.
package dbtest

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/platform/database"
)

// Open returns a pool on the migrated test database after truncating
// tables.
func Open(t *testing.T, tables ...string) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("LANKAWEATHER_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("LANKAWEATHER_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	if err := database.Migrate(ctx, databaseURL); err != nil {
		t.Fatal(err)
	}
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if len(tables) > 0 {
		if _, err := pool.Exec(ctx, "TRUNCATE "+strings.Join(tables, ", ")+" CASCADE"); err != nil {
			t.Fatal(err)
		}
	}
	return pool
}
