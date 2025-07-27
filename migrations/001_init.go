package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Create notes collection
		notes := core.NewBaseCollection("notes")
		notes.Fields.Add(&core.TextField{
			Name:     "phrase_hash",
			Required: true,
		})
		notes.Fields.Add(&core.TextField{
			Name:     "message",
			Required: true,
		})
		notes.Fields.Add(&core.TextField{
			Name: "image_hash",
		})

		if err := app.Save(notes); err != nil {
			return err
		}

		// Create encrypted_files collection
		files := core.NewBaseCollection("encrypted_files")
		files.Fields.Add(&core.TextField{
			Name:     "phrase_hash",
			Required: true,
		})
		files.Fields.Add(&core.TextField{
			Name:     "file_name",
			Required: true,
		})
		files.Fields.Add(&core.TextField{
			Name:     "content_type",
			Required: true,
		})
		files.Fields.Add(&core.FileField{
			Name: "file_data",
		})
		files.Fields.Add(&core.TextField{
			Name: "encrypted_content",
			Required: false,
		})

		return app.Save(files)
	}, func(app core.App) error {
		// Drop collections on rollback
		files, err := app.FindCollectionByNameOrId("encrypted_files")
		if err == nil {
			if err := app.Delete(files); err != nil {
				return err
			}
		}

		notes, err := app.FindCollectionByNameOrId("notes")
		if err == nil {
			return app.Delete(notes)
		}
		return nil
	})
}
