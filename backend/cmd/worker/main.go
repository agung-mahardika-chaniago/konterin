// Command worker runs the WhatsApp outbox, reminders, and credential purge.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fixphone/backend/internal/config"
	"fixphone/backend/internal/database"
	"fixphone/backend/internal/jobs"
	"fixphone/backend/internal/modules/notification"
	"fixphone/backend/internal/pkg/logger"
	"fixphone/backend/internal/pkg/version"

	gormlogger "gorm.io/gorm/logger"
)

func main() {
	once := flag.Bool("once", false, "jalankan semua job satu kali lalu keluar")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("konfigurasi tidak valid", "error", err)
		os.Exit(1)
	}
	log := logger.New(cfg.LogLevel, "fixphone-worker").With("version", version.Version)
	log.Info("starting_worker", "env", cfg.AppEnv)

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
		if closeErr := database.Close(db); closeErr != nil {
			log.Warn("database_close_failed", "error", closeErr)
		}
	}()

	notifier := notification.NewService(db, notification.ServiceDeps{
		AppBaseURL: cfg.AppBaseURL, FonnteToken: cfg.FonnteToken,
		WablasToken: cfg.WablasToken, WablasSecret: cfg.WablasSecret,
		WablasURL: cfg.WablasURL,
	})
	runner := jobs.NewRunner(db, notifier, cfg.AppBaseURL)

	runOutbox := func() {
		sent, failed, runErr := runner.ProcessOutbox(ctx)
		if runErr != nil {
			log.Error("wa_outbox_failed", "error", runErr)
			return
		}
		if sent+failed > 0 {
			log.Info("wa_outbox_processed", "sent", sent, "failed", failed)
		}
	}
	runReminders := func() {
		summary, runErr := runner.RunReminders(ctx)
		if runErr != nil {
			log.Error("reminder_job_failed", "error", runErr)
			return
		}
		log.Info("reminders_processed", "overdue", summary.OverdueServices,
			"uncollected", summary.UncollectedDevices, "expiring_warranties", summary.ExpiringWarranties)
	}
	runPurge := func() {
		purged, runErr := runner.PurgeCredentials(ctx)
		if runErr != nil {
			log.Error("credential_purge_failed", "error", runErr)
			return
		}
		if purged > 0 {
			log.Info("credentials_purged", "count", purged)
		}
	}

	if *once {
		runOutbox()
		runReminders()
		runPurge()
		log.Info("worker_once_done")
		return
	}

	outboxEvery := envDuration("WORKER_OUTBOX_INTERVAL", 30*time.Second)
	reminderEvery := envDuration("WORKER_REMINDER_INTERVAL", 15*time.Minute)
	purgeEvery := envDuration("WORKER_PURGE_INTERVAL", time.Hour)
	log.Info("worker_intervals", "outbox", outboxEvery.String(),
		"reminders", reminderEvery.String(), "purge", purgeEvery.String())

	outboxTicker := time.NewTicker(outboxEvery)
	reminderTicker := time.NewTicker(reminderEvery)
	purgeTicker := time.NewTicker(purgeEvery)
	defer outboxTicker.Stop()
	defer reminderTicker.Stop()
	defer purgeTicker.Stop()

	runOutbox()
	runReminders()
	runPurge()
	for {
		select {
		case <-ctx.Done():
			log.Info("worker_stopped")
			return
		case <-outboxTicker.C:
			runOutbox()
		case <-reminderTicker.C:
			runReminders()
		case <-purgeTicker.C:
			runPurge()
		}
	}
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value < time.Second {
		return fallback
	}
	return value
}
