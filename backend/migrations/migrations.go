// Package migrations embeds the SQL migration files so the API and CLI can
// migrate without relying on the working directory or a separate file tree.
package migrations

import "embed"

// FS contains all *.sql migration pairs.
//
//go:embed *.sql
var FS embed.FS
