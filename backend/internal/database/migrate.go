package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"fixphone/backend/migrations"

	"github.com/golang-migrate/migrate/v4"
	pgxv5 "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// ExpectedVersion is the migration version this build expects. Bump it whenever
// a new migration pair is added.
const ExpectedVersion uint = 17

// Migrator wraps golang-migrate with the embedded migration files.
type Migrator struct {
	m *migrate.Migrate
}

// NewMigrator opens a migration runner for dsn.
func NewMigrator(dsn string) (*Migrator, error) {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("membaca file migrasi: %w", err)
	}

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("membuka koneksi migrasi: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping database migrasi: %w", err)
	}

	driver, err := pgxv5.WithInstance(sqlDB, &pgxv5.Config{})
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("menyiapkan driver migrasi: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "pgx5", driver)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("menyiapkan migrasi: %w", err)
	}
	return &Migrator{m: m}, nil
}

// Up applies all pending migrations.
func (m *Migrator) Up() error {
	if err := m.m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// Down rolls back the last steps migrations. steps <= 0 rolls back everything.
func (m *Migrator) Down(steps int) error {
	var err error
	if steps <= 0 {
		err = m.m.Down()
	} else {
		err = m.m.Steps(-steps)
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) && !errors.Is(err, migrate.ErrNilVersion) {
		return err
	}
	return nil
}

// UpSteps applies n pending migrations.
func (m *Migrator) UpSteps(n int) error {
	if n <= 0 {
		return m.Up()
	}
	if err := m.m.Steps(n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// Status returns the current version and dirty flag. version 0 means no
// migration has been applied yet.
func (m *Migrator) Status() (uint, bool, error) {
	version, dirty, err := m.m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return version, dirty, nil
}

// Force clears a dirty state and pins the schema version.
func (m *Migrator) Force(version int) error {
	return m.m.Force(version)
}

// Close releases the migration connection.
func (m *Migrator) Close() error {
	sourceErr, dbErr := m.m.Close()
	if sourceErr != nil {
		return sourceErr
	}
	return dbErr
}

// ReadMigrationState reads schema_migrations using an existing pooled connection.
// It is safe to call before the first migration.
func ReadMigrationState(ctx context.Context, sqlDB *sql.DB) (uint, bool, error) {
	var exists bool
	if err := sqlDB.QueryRowContext(ctx, `SELECT to_regclass('public.schema_migrations') IS NOT NULL`).Scan(&exists); err != nil {
		return 0, false, err
	}
	if !exists {
		return 0, false, nil
	}

	var version int64
	var dirty bool
	err := sqlDB.QueryRowContext(ctx, `SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return uint(version), dirty, nil
}
