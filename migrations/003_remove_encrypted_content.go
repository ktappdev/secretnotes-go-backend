package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Removes the deprecated encrypted_content field from encrypted_files.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("encrypted_files")
		if err != nil {
			return err
		}

		// Remove the encrypted_content field if present
		collection.Fields.RemoveById("encrypted_content")

		return app.Save(collection)
	}, func(app core.App) error {
		// Down: re-add the encrypted_content Text field (not strictly needed for fresh setups)
		collection, err := app.FindCollectionByNameOrId("encrypted_files")
		if err != nil {
			return err
		}

		collection.Fields.Add(&core.TextField{
			Name:     "encrypted_content",
			Required: false,
		})

		return app.Save(collection)
	})
}