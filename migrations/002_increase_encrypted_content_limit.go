package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Note: PocketBase uses SQLite TEXT for TextField, which already supports large payloads.
// This migration is a no-op to maintain compatibility with v0.29.x where LongTextField is not exposed.
func init() {
	m.Register(func(app core.App) error {
		return nil
	}, func(app core.App) error {
		return nil
	})
}
