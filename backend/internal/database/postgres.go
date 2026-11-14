package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// OpenOptions controls the connection pool and SQL logging.
type OpenOptions struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	LogLevel        gormlogger.LogLevel
}

// DefaultOpenOptions is used by the CLI and tests.
func DefaultOpenOptions() OpenOptions {
	return OpenOptions{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
		LogLevel:        gormlogger.Warn,
	}
}

// Open connects to PostgreSQL through GORM and verifies the connection.
func Open(ctx context.Context, dsn string, opts OpenOptions) (*gorm.DB, error) {
	if dsn == "" {
		return nil, errors.New("DB_DSN kosong")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:  gormlogger.Default.LogMode(opts.LogLevel),
		NowFunc: func() time.Time { return time.Now().UTC() },
		// Writes run in the caller's transaction. Disabling GORM's implicit
		// per-statement transaction keeps request-scoped RLS transactions from
		// being committed halfway through a handler.
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("membuka koneksi database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("mengambil koneksi SQL: %w", err)
	}
	if opts.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
	}
	if opts.MaxIdleConns >= 0 {
		sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
	}
	if opts.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(opts.ConnMaxLifetime)
	}
	if opts.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(opts.ConnMaxIdleTime)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}

// Ping checks the database with a short timeout.
func Ping(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return sqlDB.PingContext(ctx)
}

// Close releases the underlying connection pool.
func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// WithTx runs fn inside a database transaction. The transaction is rolled back
// when fn returns an error or panics, and committed otherwise.
func WithTx(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	if db == nil {
		return errors.New("database tidak tersedia")
	}
	return WithContext(ctx, db).Transaction(fn)
}
