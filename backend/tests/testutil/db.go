// Package testutil provides shared helpers for database-backed tests.
//
// Every integration test runs against a real PostgreSQL instance. Set
// TEST_DB_DSN to point at a disposable database; the default matches the
// portable development cluster started by scripts/dev-db.ps1.
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"fixphone/backend/internal/database"

	"gorm.io/gorm"
)

// DefaultDSN matches the portable development cluster on port 55432. The
// password is ignored by the portable cluster (trust auth) but matches the
// Docker Compose default so the same .env works for both.
const DefaultDSN = "postgres://postgres:postgres@127.0.0.1:55432/fixphone_test?sslmode=disable"

// DSN returns the test database DSN.
func DSN() string {
	if v := strings.TrimSpace(os.Getenv("TEST_DB_DSN")); v != "" {
		return v
	}
	return DefaultDSN
}

// AdminDSN returns the DSN used to create/drop the test database.
func AdminDSN() string {
	return DSN()
}

// EnsureTestDatabase creates the test database when it is missing.
func EnsureTestDatabase(ctx context.Context) error {
	name, err := database.ParseDatabaseName(DSN())
	if err != nil {
		return err
	}
	_, err = database.EnsureDatabase(ctx, DSN(), name)
	return err
}

// Open connects to the test database and registers cleanup.
func Open(t *testing.T) *gorm.DB {
	t.Helper()
	opts := database.DefaultOpenOptions()
	opts.MaxOpenConns = 5
	opts.MaxIdleConns = 2
	db, err := database.Open(context.Background(), DSN(), opts)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close(db)
	})
	return db
}

// ResetSchema drops the public schema so each test starts from an empty
// database. Extensions are recreated by migration 000001.
func ResetSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql pool: %v", err)
	}
	statements := []string{
		`DROP SCHEMA IF EXISTS public CASCADE`,
		`CREATE SCHEMA public`,
		`GRANT ALL ON SCHEMA public TO CURRENT_USER`,
	}
	for _, stmt := range statements {
		if _, err := sqlDB.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
}

// MigrateUp applies all embedded migrations to the test database.
func MigrateUp(t *testing.T) {
	t.Helper()
	m, err := database.NewMigrator(DSN())
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	version, dirty, err := m.Status()
	if err != nil {
		t.Fatalf("migration status: %v", err)
	}
	if dirty {
		t.Fatalf("migration is dirty at version %d", version)
	}
	if version != database.ExpectedVersion {
		t.Fatalf("migration version = %d, want %d", version, database.ExpectedVersion)
	}
}

// SQLDB returns the raw *sql.DB for the GORM connection.
func SQLDB(t *testing.T, db *gorm.DB) *sql.DB {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql pool: %v", err)
	}
	return sqlDB
}

// Count returns count(*) for a table. The table name is validated against a
// small allowlist to keep this helper safe when reused.
func Count(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var count int64
	if err := db.Raw(fmt.Sprintf(`SELECT count(*) FROM %s`, table)).Scan(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}
