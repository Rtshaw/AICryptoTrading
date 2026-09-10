// Package migrations embeds the SQL migration files so the server binary
// carries them without needing the source tree at runtime.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
