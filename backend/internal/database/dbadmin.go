package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var databaseNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// ParseDatabaseName returns the database name from a PostgreSQL URL.
func ParseDatabaseName(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		return "", errors.New("DSN harus berupa URL postgres:// untuk operasi database")
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		return "", errors.New("DSN tidak memuat nama database")
	}
	return name, nil
}

func withDatabase(dsn, database string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	u.Path = "/" + database
	return u.String(), nil
}

// DSNWithDatabase returns dsn pointed at another database on the same server.
func DSNWithDatabase(dsn, database string) (string, error) {
	return withDatabase(dsn, database)
}

func validDatabaseName(name string) error {
	if !databaseNamePattern.MatchString(name) {
		return errors.New("nama database harus huruf kecil/angka/underscore dan diawali huruf")
	}
	if name == "postgres" || name == "template0" || name == "template1" {
		return errors.New("nama database sistem tidak boleh diubah")
	}
	return nil
}

func openAdmin(ctx context.Context, dsn string) (*sql.DB, error) {
	adminDSN, err := withDatabase(dsn, "postgres")
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("pgx", adminDSN)
	if err != nil {
		return nil, fmt.Errorf("membuka koneksi admin: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database admin: %w", err)
	}
	return db, nil
}

// EnsureDatabase creates name when it does not exist. It returns true when the
// database was created by this call.
func EnsureDatabase(ctx context.Context, dsn, name string) (bool, error) {
	if err := validDatabaseName(name); err != nil {
		return false, err
	}
	db, err := openAdmin(ctx, dsn)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var exists bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, name).Scan(&exists); err != nil {
		return false, fmt.Errorf("memeriksa database: %w", err)
	}
	if exists {
		return false, nil
	}
	if _, err := db.ExecContext(ctx, `CREATE DATABASE "`+name+`" ENCODING 'UTF8' TEMPLATE template0`); err != nil {
		return false, fmt.Errorf("membuat database %s: %w", name, err)
	}
	return true, nil
}

// DropDatabase removes a database after terminating its connections. Names that
// do not end in _test or _dev require force=true.
func DropDatabase(ctx context.Context, dsn, name string, force bool) error {
	if err := validDatabaseName(name); err != nil {
		return err
	}
	if !force && !strings.HasSuffix(name, "_test") && !strings.HasSuffix(name, "_dev") {
		return errors.New("hanya database berakhiran _test atau _dev yang boleh dihapus tanpa --force")
	}
	db, err := openAdmin(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx,
		`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`,
		name); err != nil {
		return fmt.Errorf("memutus koneksi database: %w", err)
	}
	if _, err := db.ExecContext(ctx, `DROP DATABASE IF EXISTS "`+name+`"`); err != nil {
		return fmt.Errorf("menghapus database %s: %w", name, err)
	}
	return nil
}
