package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config is the complete runtime configuration loaded from environment
// variables. Secrets are never written to logs.
type Config struct {
	AppEnv     string
	AppPort    string
	AppBaseURL string
	LogLevel   slog.Level

	DBDSN             string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	DBConnMaxIdleTime time.Duration

	JWTSecret              string
	AccessTokenTTL         time.Duration
	RefreshTokenTTL        time.Duration
	LoginMaxAttempts       int
	LoginLockDuration      time.Duration
	LoginRateLimit         int
	LoginRateWindow        time.Duration
	AppEncKey              []byte
	CORSOrigins            []string
	UploadDir              string
	UploadMaxMB            int
	BackupDir              string
	BackupDSN              string
	SeedSuperadminEmail    string
	SeedSuperadminPassword string
	SeedDemo               bool

	FonnteToken  string
	WablasToken  string
	WablasSecret string
	WablasURL    string
}

// Load reads .env (when present) and then validates the environment.
func Load() (*Config, error) {
	loadDotEnv()

	cfg := &Config{
		AppEnv:     getEnv("APP_ENV", "development"),
		AppPort:    getEnv("APP_PORT", "8080"),
		AppBaseURL: getEnv("APP_BASE_URL", "http://localhost:5173"),
		DBDSN:      strings.TrimSpace(os.Getenv("DB_DSN")),
		JWTSecret:  strings.TrimSpace(os.Getenv("JWT_SECRET")),
		UploadDir:  getEnv("UPLOAD_DIR", "uploads"),
		BackupDir:  getEnv("BACKUP_DIR", "backups"),
		BackupDSN:  strings.TrimSpace(os.Getenv("BACKUP_DSN")),
	}

	var err error
	if cfg.LogLevel, err = ParseLogLevel(getEnv("LOG_LEVEL", "info")); err != nil {
		return nil, err
	}
	if cfg.DBMaxOpenConns, err = getEnvInt("DB_MAX_OPEN_CONNS", 25); err != nil {
		return nil, err
	}
	if cfg.DBMaxIdleConns, err = getEnvInt("DB_MAX_IDLE_CONNS", 5); err != nil {
		return nil, err
	}
	if cfg.DBConnMaxLifetime, err = getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute); err != nil {
		return nil, err
	}
	if cfg.DBConnMaxIdleTime, err = getEnvDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute); err != nil {
		return nil, err
	}
	if cfg.UploadMaxMB, err = getEnvInt("UPLOAD_MAX_MB", 5); err != nil {
		return nil, err
	}
	if cfg.AccessTokenTTL, err = getEnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute); err != nil {
		return nil, err
	}
	if cfg.RefreshTokenTTL, err = getEnvDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour); err != nil {
		return nil, err
	}
	if cfg.LoginMaxAttempts, err = getEnvInt("LOGIN_MAX_ATTEMPTS", 5); err != nil {
		return nil, err
	}
	if cfg.LoginLockDuration, err = getEnvDuration("LOGIN_LOCK_DURATION", 15*time.Minute); err != nil {
		return nil, err
	}
	if cfg.LoginRateLimit, err = getEnvInt("LOGIN_RATE_LIMIT", 5); err != nil {
		return nil, err
	}
	if cfg.LoginRateWindow, err = getEnvDuration("LOGIN_RATE_WINDOW", time.Minute); err != nil {
		return nil, err
	}
	if cfg.SeedDemo, err = getEnvBool("SEED_DEMO", false); err != nil {
		return nil, err
	}

	cfg.CORSOrigins = splitCSV(os.Getenv("CORS_ORIGINS"))
	cfg.SeedSuperadminEmail = strings.ToLower(strings.TrimSpace(os.Getenv("SEED_SUPERADMIN_EMAIL")))
	cfg.SeedSuperadminPassword = os.Getenv("SEED_SUPERADMIN_PASSWORD")
	cfg.FonnteToken = strings.TrimSpace(os.Getenv("WA_FONNTE_TOKEN"))
	cfg.WablasToken = strings.TrimSpace(os.Getenv("WA_WABLAS_TOKEN"))
	cfg.WablasSecret = strings.TrimSpace(os.Getenv("WA_WABLAS_SECRET"))
	cfg.WablasURL = strings.TrimSpace(os.Getenv("WA_WABLAS_URL"))

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	switch c.AppEnv {
	case "development", "test", "production":
	default:
		return fmt.Errorf("APP_ENV harus salah satu dari development, test, production")
	}
	if c.DBDSN == "" {
		return errors.New("DB_DSN wajib diisi")
	}
	if err := validateDSN(c.DBDSN); err != nil {
		return err
	}
	if len(c.JWTSecret) < 32 {
		return errors.New("JWT_SECRET minimal 32 karakter")
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(os.Getenv("APP_ENC_KEY")))
	if err != nil {
		return errors.New("APP_ENC_KEY harus base64 valid")
	}
	if len(key) != 32 {
		return errors.New("APP_ENC_KEY harus berisi tepat 32 byte setelah base64 decode")
	}
	c.AppEncKey = key
	if c.DBMaxOpenConns < 1 {
		return errors.New("DB_MAX_OPEN_CONNS minimal 1")
	}
	if c.DBMaxIdleConns < 0 || c.DBMaxIdleConns > c.DBMaxOpenConns {
		return errors.New("DB_MAX_IDLE_CONNS harus antara 0 dan DB_MAX_OPEN_CONNS")
	}
	if c.UploadMaxMB < 1 {
		return errors.New("UPLOAD_MAX_MB minimal 1")
	}
	if _, err := strconv.Atoi(c.AppPort); err != nil {
		return errors.New("APP_PORT harus berupa angka")
	}
	if c.AccessTokenTTL < time.Minute || c.AccessTokenTTL > 24*time.Hour {
		return errors.New("ACCESS_TOKEN_TTL harus antara 1m dan 24h")
	}
	if c.RefreshTokenTTL < time.Hour || c.RefreshTokenTTL > 365*24*time.Hour {
		return errors.New("REFRESH_TOKEN_TTL harus antara 1h dan 8760h")
	}
	if c.LoginMaxAttempts < 1 || c.LoginMaxAttempts > 100 {
		return errors.New("LOGIN_MAX_ATTEMPTS harus antara 1 dan 100")
	}
	if c.LoginLockDuration < time.Second || c.LoginLockDuration > 24*time.Hour {
		return errors.New("LOGIN_LOCK_DURATION harus antara 1s dan 24h")
	}
	if c.LoginRateLimit < 1 || c.LoginRateLimit > 10000 {
		return errors.New("LOGIN_RATE_LIMIT harus antara 1 dan 10000")
	}
	if c.LoginRateWindow < time.Second || c.LoginRateWindow > time.Hour {
		return errors.New("LOGIN_RATE_WINDOW harus antara 1s dan 1h")
	}
	return nil
}

func validateDSN(dsn string) error {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil || u.Host == "" {
			return errors.New("DB_DSN bukan URL PostgreSQL yang valid")
		}
		return nil
	}
	if strings.Contains(dsn, "host=") && strings.Contains(dsn, "dbname=") {
		return nil
	}
	return errors.New("DB_DSN harus berupa URL postgres:// atau keyword DSN dengan host dan dbname")
}

// LoadDBOnly loads only the database-related settings. It is used by CLI
// commands that must work before API secrets are configured.
func LoadDBOnly() (*Config, error) {
	loadDotEnv()
	cfg := &Config{
		AppEnv: getEnv("APP_ENV", "development"),
		DBDSN:  strings.TrimSpace(os.Getenv("DB_DSN")),
	}
	var err error
	if cfg.DBMaxOpenConns, err = getEnvInt("DB_MAX_OPEN_CONNS", 25); err != nil {
		return nil, err
	}
	if cfg.DBMaxIdleConns, err = getEnvInt("DB_MAX_IDLE_CONNS", 5); err != nil {
		return nil, err
	}
	if cfg.DBConnMaxLifetime, err = getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute); err != nil {
		return nil, err
	}
	if cfg.DBConnMaxIdleTime, err = getEnvDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute); err != nil {
		return nil, err
	}
	if cfg.DBDSN == "" {
		return nil, errors.New("DB_DSN wajib diisi")
	}
	if err := validateDSN(cfg.DBDSN); err != nil {
		return nil, err
	}
	return cfg, nil
}

// IsProduction reports whether production-only rules should be enforced.
func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

// ParseLogLevel converts a textual level to slog.Level.
func ParseLogLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("LOG_LEVEL tidak valid: %s", value)
	}
}

// loadDotEnv looks for .env in the current directory and up to three parent
// directories. Existing process variables always win.
func loadDotEnv() {
	if path := os.Getenv("ENV_FILE"); path != "" {
		_ = godotenv.Load(path)
		return
	}
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for i := 0; i < 4; i++ {
		candidate := filepath.Join(dir, ".env")
		if _, statErr := os.Stat(candidate); statErr == nil {
			_ = godotenv.Load(candidate)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s harus berupa angka", key)
	}
	return v, nil
}

func getEnvBool(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s harus true atau false", key)
	}
	return v, nil
}

func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s harus durasi valid (contoh 5m, 30s)", key)
	}
	return v, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
