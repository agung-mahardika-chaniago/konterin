package config_test

import (
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"

	"fixphone/backend/internal/config"
)

const validDSN = "postgres://postgres@127.0.0.1:55432/fixphone_test?sslmode=disable"

func isolateEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ENV_FILE", filepath.Join(t.TempDir(), "does-not-exist.env"))
	for _, key := range []string{
		"APP_ENV", "APP_PORT", "APP_BASE_URL", "DB_DSN", "JWT_SECRET",
		"APP_ENC_KEY", "CORS_ORIGINS", "UPLOAD_DIR", "UPLOAD_MAX_MB",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME",
		"DB_CONN_MAX_IDLE_TIME", "SEED_DEMO", "DB_ADMIN_DSN", "BACKUP_DSN",
		"TRUSTED_PROXIES", "METRICS_TOKEN",
	} {
		t.Setenv(key, "")
	}
}

func encKey() string {
	return base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
}

func TestParseLogLevel(t *testing.T) {
	cases := map[string]string{
		"debug":   "DEBUG",
		"info":    "INFO",
		"warn":    "WARN",
		"warning": "WARN",
		"error":   "ERROR",
	}
	for input, want := range cases {
		level, err := config.ParseLogLevel(input)
		if err != nil {
			t.Fatalf("ParseLogLevel(%q): %v", input, err)
		}
		if level.String() != want {
			t.Fatalf("ParseLogLevel(%q) = %s, want %s", input, level.String(), want)
		}
	}
	if _, err := config.ParseLogLevel("verbose"); err == nil {
		t.Fatal("expected invalid log level error")
	}
}

func TestLoadRejectsMissingDatabaseDSN(t *testing.T) {
	isolateEnv(t)
	if _, err := config.Load(); err == nil || !strings.Contains(err.Error(), "DB_DSN") {
		t.Fatalf("expected DB_DSN error, got %v", err)
	}
}

func TestLoadRejectsInvalidSecrets(t *testing.T) {
	t.Run("short jwt", func(t *testing.T) {
		isolateEnv(t)
		t.Setenv("DB_DSN", validDSN)
		t.Setenv("JWT_SECRET", "too-short")
		t.Setenv("APP_ENC_KEY", encKey())
		_, err := config.Load()
		if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
			t.Fatalf("expected JWT_SECRET error, got %v", err)
		}
	})

	t.Run("wrong encryption key length", func(t *testing.T) {
		isolateEnv(t)
		t.Setenv("DB_DSN", validDSN)
		t.Setenv("JWT_SECRET", strings.Repeat("j", 40))
		t.Setenv("APP_ENC_KEY", base64.StdEncoding.EncodeToString([]byte("short")))
		_, err := config.Load()
		if err == nil || !strings.Contains(err.Error(), "APP_ENC_KEY") {
			t.Fatalf("expected APP_ENC_KEY error, got %v", err)
		}
	})

	t.Run("invalid encryption key encoding", func(t *testing.T) {
		isolateEnv(t)
		t.Setenv("DB_DSN", validDSN)
		t.Setenv("JWT_SECRET", strings.Repeat("j", 40))
		t.Setenv("APP_ENC_KEY", "***not-base64***")
		_, err := config.Load()
		if err == nil || !strings.Contains(err.Error(), "base64") {
			t.Fatalf("expected base64 error, got %v", err)
		}
	})
}

func TestLoadValidConfiguration(t *testing.T) {
	isolateEnv(t)
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("DB_DSN", validDSN)
	t.Setenv("JWT_SECRET", strings.Repeat("j", 40))
	t.Setenv("APP_ENC_KEY", encKey())
	t.Setenv("CORS_ORIGINS", "http://localhost:5173, https://fixphone.example.com")
	t.Setenv("DB_MAX_OPEN_CONNS", "10")
	t.Setenv("DB_MAX_IDLE_CONNS", "3")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AppPort != "9090" || cfg.DBMaxOpenConns != 10 || cfg.DBMaxIdleConns != 3 {
		t.Fatalf("unexpected numeric config: %+v", cfg)
	}
	if len(cfg.CORSOrigins) != 2 || cfg.CORSOrigins[1] != "https://fixphone.example.com" {
		t.Fatalf("unexpected CORS origins: %v", cfg.CORSOrigins)
	}
	if len(cfg.AppEncKey) != 32 {
		t.Fatalf("encryption key length = %d, want 32", len(cfg.AppEncKey))
	}
	if cfg.LogLevel.String() != "DEBUG" {
		t.Fatalf("log level = %s", cfg.LogLevel)
	}
}

func TestLoadRejectsBadPoolLimits(t *testing.T) {
	isolateEnv(t)
	t.Setenv("DB_DSN", validDSN)
	t.Setenv("JWT_SECRET", strings.Repeat("j", 40))
	t.Setenv("APP_ENC_KEY", encKey())
	t.Setenv("DB_MAX_OPEN_CONNS", "2")
	t.Setenv("DB_MAX_IDLE_CONNS", "5")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected pool limit validation error")
	}
}

func TestLoadProductionRequiresAdminDSN(t *testing.T) {
	isolateEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_BASE_URL", "https://fixphone.example.com")
	t.Setenv("CORS_ORIGINS", "https://fixphone.example.com")
	t.Setenv("DB_DSN", validDSN)
	t.Setenv("JWT_SECRET", strings.Repeat("j", 40))
	t.Setenv("APP_ENC_KEY", encKey())

	if _, err := config.Load(); err == nil || !strings.Contains(err.Error(), "DB_ADMIN_DSN") {
		t.Fatalf("expected DB_ADMIN_DSN error, got %v", err)
	}
}

func TestLoadProductionBackupFallsBackToAdminDSN(t *testing.T) {
	isolateEnv(t)
	const adminDSN = "postgres://postgres:secret@127.0.0.1:55432/fixphone?sslmode=disable"
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_BASE_URL", "https://fixphone.example.com")
	t.Setenv("CORS_ORIGINS", "https://fixphone.example.com")
	t.Setenv("DB_DSN", "postgres://fixphone_app:secret@127.0.0.1:55432/fixphone?sslmode=disable")
	t.Setenv("DB_ADMIN_DSN", adminDSN)
	t.Setenv("JWT_SECRET", strings.Repeat("j", 40))
	t.Setenv("APP_ENC_KEY", encKey())

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BackupDSN != adminDSN {
		t.Fatalf("BackupDSN = %q, want admin DSN fallback", cfg.BackupDSN)
	}
}

func TestLoadDBOnlySkipsAPISecrets(t *testing.T) {
	isolateEnv(t)
	t.Setenv("DB_DSN", validDSN)
	cfg, err := config.LoadDBOnly()
	if err != nil {
		t.Fatalf("LoadDBOnly: %v", err)
	}
	if cfg.DBDSN != validDSN {
		t.Fatalf("DBDSN = %q", cfg.DBDSN)
	}
	if len(cfg.AppEncKey) != 0 {
		t.Fatal("LoadDBOnly must not read API secrets")
	}
}

func TestLoadDBOnlyRejectsBadDSN(t *testing.T) {
	isolateEnv(t)
	t.Setenv("DB_DSN", "mysql://root@localhost/app")
	if _, err := config.LoadDBOnly(); err == nil {
		t.Fatal("expected DSN validation error")
	}
}
