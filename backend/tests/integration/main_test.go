package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"fixphone/backend/tests/testutil"

	"gorm.io/gorm"
)

// TestMain prepares the disposable test database once per package run.
func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := testutil.EnsureTestDatabase(ctx); err != nil {
		os.Stderr.WriteString("cannot prepare test database: " + err.Error() + "\n")
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// freshDB resets the public schema and applies every migration.
func freshDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutil.Open(t)
	testutil.ResetSchema(t, db)
	testutil.MigrateUp(t)
	return db
}
