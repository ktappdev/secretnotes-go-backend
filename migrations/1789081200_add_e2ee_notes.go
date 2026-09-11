package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		legacyFiles, err := app.FindCollectionByNameOrId("encrypted_files")
		if err != nil {
			return err
		}
		legacyFileField, ok := legacyFiles.Fields.GetByName("file_data").(*core.FileField)
		if !ok {
			return fmt.Errorf("encrypted_files.file_data is not a file field")
		}
		legacyFileField.Protected = true
		if err := app.Save(legacyFiles); err != nil {
			return err
		}

		notes := core.NewBaseCollection("notes_v2")
		notes.ListRule = nil
		notes.ViewRule = nil
		notes.CreateRule = nil
		notes.UpdateRule = nil
		notes.DeleteRule = nil
		notes.Fields.Add(&core.TextField{
			Name:     "lookup_hash",
			Required: true,
			Min:      64,
			Max:      64,
			Pattern:  "^[a-f0-9]{64}$",
		})
		notes.Fields.Add(&core.TextField{
			Name:     "ciphertext",
			Required: true,
			Max:      300_000,
		})
		notes.Fields.Add(&core.FileField{
			Name:      "image_file",
			MaxSize:   15 << 20,
			MaxSelect: 1,
			MimeTypes: []string{"application/octet-stream", "application/json"},
			Protected: true,
		})
		notes.Fields.Add(&core.TextField{
			Name: "image_metadata",
			Max:  20_000,
		})
		notes.Fields.Add(&core.NumberField{
			Name:    "version",
			Min:     floatPtr(1),
			OnlyInt: true,
		})
		notes.Fields.Add(&core.AutodateField{
			Name:     "created",
			OnCreate: true,
		})
		notes.Fields.Add(&core.AutodateField{
			Name:     "updated",
			OnCreate: true,
			OnUpdate: true,
		})
		notes.Indexes = append(notes.Indexes,
			"CREATE UNIQUE INDEX idx_notes_v2_lookup_hash ON notes_v2 (lookup_hash)",
		)

		return app.Save(notes)
	}, func(app core.App) error {
		if notes, err := app.FindCollectionByNameOrId("notes_v2"); err == nil {
			if err := app.Delete(notes); err != nil {
				return err
			}
		}

		if legacyFiles, err := app.FindCollectionByNameOrId("encrypted_files"); err == nil {
			if fileField, ok := legacyFiles.Fields.GetByName("file_data").(*core.FileField); ok {
				fileField.Protected = false
				return app.Save(legacyFiles)
			}
		}

		return nil
	})
}
