package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fixphone/backend/internal/config"
	"fixphone/backend/internal/database"
	"fixphone/backend/internal/pkg/logger"
	"fixphone/backend/internal/pkg/version"
	"fixphone/backend/internal/router"

	gormlogger "gorm.io/gorm/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("konfigurasi tidak valid", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel, "fixphone-api").With("version", version.Version)
	log.Info("starting_api", "env", cfg.AppEnv, "port", cfg.AppPort)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	opts := database.DefaultOpenOptions()
	opts.MaxOpenConns = cfg.DBMaxOpenConns
	opts.MaxIdleConns = cfg.DBMaxIdleConns
	opts.ConnMaxLifetime = cfg.DBConnMaxLifetime
	opts.ConnMaxIdleTime = cfg.DBConnMaxIdleTime
	if cfg.IsProduction() {
		opts.LogLevel = gormlogger.Silent
	} else {
		opts.LogLevel = gormlogger.Warn
	}

	db, err := database.Open(ctx, cfg.DBDSN, opts)
	if err != nil {
		log.Error("database_connection_failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := database.Close(db); err != nil {
			log.Warn("database_close_failed", "error", err)
		}
	}()

	engine := router.New(cfg, db, log)
	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Info("http_listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown_signal_received")
	case err := <-serverErrors:
		log.Error("http_server_failed", "error", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("http_shutdown_failed", "error", err)
		_ = server.Close()
	}
	log.Info("api_stopped")
}
