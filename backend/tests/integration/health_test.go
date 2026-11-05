package integration_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"fixphone/backend/internal/config"
	"fixphone/backend/internal/database"
	"fixphone/backend/internal/pkg/logger"
	"fixphone/backend/internal/router"
	"fixphone/backend/tests/testutil"

	"github.com/gin-gonic/gin"
)

func TestHealthAndMetaRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.Open(t)
	testutil.ResetSchema(t, db)

	cfg := &config.Config{
		AppEnv:      "test",
		AppPort:     "8080",
		DBDSN:       testutil.DSN(),
		JWTSecret:   "test-secret-that-is-long-enough-for-validation",
		CORSOrigins: []string{"http://localhost:5173"},
	}
	log := logger.New(slog.LevelError, "test")

	t.Run("readyz is 503 before migrations", func(t *testing.T) {
		engine := router.New(cfg, db, log)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503; body=%s", rec.Code, rec.Body.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode readyz: %v", err)
		}
		if payload["status"] != "not_ready" {
			t.Fatalf("status payload = %v, want not_ready", payload["status"])
		}
	})

	testutil.MigrateUp(t)

	t.Run("readyz is 200 after migrations", func(t *testing.T) {
		engine := router.New(cfg, db, log)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
		}
		var payload struct {
			Status           string `json:"status"`
			Database         string `json:"database"`
			MigrationVersion uint   `json:"migration_version"`
			Dirty            bool   `json:"dirty"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode readyz: %v", err)
		}
		if payload.Status != "ready" || payload.Database != "up" {
			t.Fatalf("unexpected readyz payload: %+v", payload)
		}
		if payload.MigrationVersion != database.ExpectedVersion || payload.Dirty {
			t.Fatalf("migration state = %d dirty=%t", payload.MigrationVersion, payload.Dirty)
		}
	})

	t.Run("healthz and version respond", func(t *testing.T) {
		engine := router.New(cfg, db, log)

		health := httptest.NewRecorder()
		engine.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		if health.Code != http.StatusOK {
			t.Fatalf("healthz status = %d", health.Code)
		}
		if health.Header().Get("X-Request-ID") == "" {
			t.Fatal("healthz is missing X-Request-ID")
		}

		version := httptest.NewRecorder()
		engine.ServeHTTP(version, httptest.NewRequest(http.MethodGet, "/api/v1/version", nil))
		if version.Code != http.StatusOK {
			t.Fatalf("version status = %d", version.Code)
		}
		var payload map[string]string
		if err := json.Unmarshal(version.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode version: %v", err)
		}
		if payload["name"] != "fixphone-api" || payload["version"] == "" {
			t.Fatalf("unexpected version payload: %+v", payload)
		}
	})

	t.Run("unknown route returns json error", func(t *testing.T) {
		engine := router.New(cfg, db, log)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/nope", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
		var payload struct {
			Success bool `json:"success"`
			Error   struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode error payload: %v", err)
		}
		if payload.Success || payload.Error.Code != "NOT_FOUND" {
			t.Fatalf("unexpected error payload: %+v", payload)
		}
	})

	t.Run("cors allows configured origin only", func(t *testing.T) {
		engine := router.New(cfg, db, log)

		allowed := httptest.NewRequest(http.MethodOptions, "/api/v1/healthz", nil)
		allowed.Header.Set("Origin", "http://localhost:5173")
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, allowed)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("preflight status = %d, want 204", rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
			t.Fatalf("missing allow-origin header: %v", rec.Header())
		}

		denied := httptest.NewRequest(http.MethodOptions, "/api/v1/healthz", nil)
		denied.Header.Set("Origin", "https://evil.example")
		rec = httptest.NewRecorder()
		engine.ServeHTTP(rec, denied)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("disallowed preflight status = %d, want 403", rec.Code)
		}
	})
}
