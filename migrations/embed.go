// Package migrations embeds the goose SQL migrations so the binary can apply
// them without the source tree.
package migrations

import "embed"

// Files holds every migration in this directory.
//
//go:embed *.sql
var Files embed.FS
