package router

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"fixphone/backend/internal/config"
	"fixphone/backend/internal/database"
	"fixphone/backend/internal/middleware"
	"fixphone/backend/internal/modules/auth"
	"fixphone/backend/internal/modules/dashboard"
	"fixphone/backend/internal/modules/expense"
	"fixphone/backend/internal/modules/inventory"
	"fixphone/backend/internal/modules/invoice"
	"fixphone/backend/internal/modules/outlet"
	"fixphone/backend/internal/modules/payment"
	servicemod "fixphone/backend/internal/modules/service"
	"fixphone/backend/internal/modules/user"
	"fixphone/backend/internal/pkg/jwtutil"
	"fixphone/backend/internal/pkg/version"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const requestIDKey = "request_id"

// New builds the HTTP engine, global middleware, health checks, and Stage 3
// auth/RBAC routes.
func New(cfg *config.Config, db *gorm.DB, log *slog.Logger) *gin.Engine {
	switch cfg.AppEnv {
	case "production":
		gin.SetMode(gin.ReleaseMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	r.Use(requestIDMiddleware())
	r.Use(accessLogMiddleware(log))
	r.Use(recoveryMiddleware(log))
	r.Use(middleware.SecurityHeaders())
	r.Use(corsMiddleware(cfg.CORSOrigins))
	r.Use(middleware.CheckOrigin(cfg.CORSOrigins))
	r.Use(bodyLimitMiddleware(1 << 20)) // 1 MB JSON

	r.GET("/healthz", healthz)
	r.GET("/readyz", readyz(db, log))
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, version.Info())
	})

	accessTTL := cfg.AccessTokenTTL
	if accessTTL <= 0 {
		accessTTL = 15 * time.Minute
	}
	refreshTTL := cfg.RefreshTokenTTL
	if refreshTTL <= 0 {
		refreshTTL = 30 * 24 * time.Hour
	}
	maxAttempts := cfg.LoginMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	lockDuration := cfg.LoginLockDuration
	if lockDuration <= 0 {
		lockDuration = 15 * time.Minute
	}
	rateLimit := cfg.LoginRateLimit
	if rateLimit <= 0 {
		rateLimit = 5
	}
	rateWindow := cfg.LoginRateWindow
	if rateWindow <= 0 {
		rateWindow = time.Minute
	}

	tokenManager, err := jwtutil.NewManager(cfg.JWTSecret, accessTTL)
	if err != nil {
		// Config validation guarantees a usable secret in production. Reaching
		// this branch means the process cannot issue safe tokens.
		panic("router: " + err.Error())
	}

	userRepo := user.NewRepository(db)
	userService := user.NewService(db, userRepo)
	authService := auth.NewService(db, userRepo, tokenManager, auth.ServiceConfig{
		MaxLoginAttempts: maxAttempts,
		LockDuration:     lockDuration,
		RefreshTTL:       refreshTTL,
	})
	outletRepo := outlet.NewRepository(db)
	outletService := outlet.NewService(db, outletRepo)
	dashboardService := dashboard.NewService(db)
	serviceModule := servicemod.NewService(db, servicemod.Options{
		UploadDir:      cfg.UploadDir,
		UploadMaxBytes: int64(cfg.UploadMaxMB) << 20,
		EncKey:         cfg.AppEncKey,
		AppBaseURL:     cfg.AppBaseURL,
	})
	inventoryModule := inventory.NewService(db)
	invoiceModule := invoice.NewService(db)
	paymentModule := payment.NewService(db)
	expenseModule := expense.NewService(db)

	authMiddleware := middleware.Auth(tokenManager, authService)
	tenantMiddleware := middleware.Tenant(db)
	can := middleware.RequirePermission
	loginLimiter := middleware.RateLimit(rateLimit, rateWindow)

	api := r.Group("/api/v1")
	{
		api.GET("/healthz", healthz)
		api.GET("/readyz", readyz(db, log))
		api.GET("/version", func(c *gin.Context) {
			c.JSON(http.StatusOK, version.Info())
		})

		auth.RegisterRoutes(api, auth.NewHandler(authService, cfg.IsProduction()), authMiddleware, loginLimiter)
		user.RegisterRoutes(api, user.NewHandler(userService), authMiddleware, tenantMiddleware, can)
		outlet.RegisterRoutes(api, outlet.NewHandler(outletService), authMiddleware, tenantMiddleware, can)
		dashboard.RegisterRoutes(api, dashboard.NewHandler(dashboardService), authMiddleware, tenantMiddleware, can)
		servicemod.RegisterRoutes(api, servicemod.NewHandler(serviceModule), authMiddleware, tenantMiddleware, can)
		inventory.RegisterRoutes(api, inventory.NewHandler(inventoryModule), authMiddleware, tenantMiddleware, can)
		invoice.RegisterRoutes(api, invoice.NewHandler(invoiceModule), authMiddleware, tenantMiddleware, can)
		payment.RegisterRoutes(api, payment.NewHandler(paymentModule), authMiddleware, tenantMiddleware, can)
		expense.RegisterRoutes(api, expense.NewHandler(expenseModule), authMiddleware, tenantMiddleware, can)
	}

	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Warn("upload_dir_unavailable", "dir", cfg.UploadDir, "error", err)
	} else {
		r.Static("/uploads", cfg.UploadDir)
	}

	r.NoRoute(func(c *gin.Context) {
		errorJSON(c, http.StatusNotFound, "NOT_FOUND", "Endpoint tidak ditemukan.")
	})
	return r
}

func healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": version.Version,
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

func readyz(db *gorm.DB, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		payload := gin.H{
			"status":             "ready",
			"database":           "up",
			"migration_version":  uint(0),
			"migration_expected": database.ExpectedVersion,
			"dirty":              false,
			"time":               time.Now().UTC().Format(time.RFC3339),
		}

		ready := true
		if err := database.Ping(ctx, db); err != nil {
			log.Error("readyz_database_down", "request_id", requestID(c), "error", err)
			payload["status"] = "not_ready"
			payload["database"] = "down"
			payload["error"] = "Database tidak dapat dihubungi."
			ready = false
		} else if sqlDB, err := db.DB(); err != nil {
			log.Error("readyz_sql_pool_error", "request_id", requestID(c), "error", err)
			payload["status"] = "not_ready"
			payload["error"] = "Koneksi database tidak siap."
			ready = false
		} else {
			migrationVersion, dirty, err := database.ReadMigrationState(ctx, sqlDB)
			if err != nil {
				log.Error("readyz_migration_state_error", "request_id", requestID(c), "error", err)
				payload["status"] = "not_ready"
				payload["error"] = "Status migrasi tidak dapat dibaca."
				ready = false
			} else {
				payload["migration_version"] = migrationVersion
				payload["dirty"] = dirty
				if dirty {
					payload["status"] = "not_ready"
					payload["error"] = "Migrasi database berstatus dirty."
					ready = false
				} else if migrationVersion != database.ExpectedVersion {
					payload["status"] = "not_ready"
					payload["error"] = "Versi migrasi database tidak sesuai dengan aplikasi."
					ready = false
				}
			}
		}

		if !ready {
			c.JSON(http.StatusServiceUnavailable, payload)
			return
		}
		c.JSON(http.StatusOK, payload)
	}
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.NewV7()
		if err != nil {
			id = uuid.New()
		}
		value := id.String()
		c.Set(requestIDKey, value)
		c.Header("X-Request-ID", value)
		c.Next()
	}
}

func accessLogMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info("http_request",
			"request_id", requestID(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", float64(time.Since(start).Microseconds())/1000,
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		)
	}
}

func recoveryMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error("http_panic", "request_id", requestID(c), "panic", recovered)
				errorJSON(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan pada server.")
				c.Abort()
			}
		}()
		c.Next()
	}
}

func corsMiddleware(origins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, X-Outlet-Id, Idempotency-Key")
				c.Header("Access-Control-Expose-Headers", "X-Request-ID")
				c.Header("Access-Control-Max-Age", "600")
			}
		}
		if c.Request.Method == http.MethodOptions {
			if origin != "" {
				if _, ok := allowed[origin]; !ok {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func bodyLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") &&
			c.Request.ContentLength > maxBytes {
			errorJSON(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Ukuran request terlalu besar.")
			c.Abort()
			return
		}
		c.Next()
	}
}

func requestID(c *gin.Context) string {
	if value, ok := c.Get(requestIDKey); ok {
		if id, ok := value.(string); ok {
			return id
		}
	}
	return ""
}

func errorJSON(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
		"request_id": requestID(c),
	})
}
