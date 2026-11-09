package version

import "runtime"

var (
	// Version is set with -ldflags "-X fixphone/backend/internal/pkg/version.Version=..." at build time.
	Version   = "0.9.0"
	Commit    = "dev"
	BuildDate = "unknown"
)

// Info returns metadata exposed by GET /version.
func Info() map[string]string {
	return map[string]string{
		"name":       "fixphone-api",
		"version":    Version,
		"commit":     Commit,
		"build_date": BuildDate,
		"go_version": runtime.Version(),
	}
}
