package logger

import (
	"log/slog"
	"os"
)

// New builds a JSON structured logger. Health/request logs intentionally never
// include DSNs, tokens, or device credentials.
func New(level slog.Level, service string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: false,
	})
	return slog.New(handler).With("service", service)
}
