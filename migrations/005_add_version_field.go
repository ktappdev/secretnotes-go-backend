package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		notes, err := app.FindCollectionByNameOrId("notes")
		if err != nil {
			return err
		}

		notes.Fields.Add(&core.NumberField{
			Name:    "version",
			Min:     floatPtr(1),
			OnlyInt: true,
		})

		return app.Save(notes)
	}, func(app core.App) error {
		notes, err := app.FindCollectionByNameOrId("notes")
		if err != nil {
			return err
		}

		notes.Fields.RemoveByName("version")
		return app.Save(notes)
	})
}

func floatPtr(f float64) *float64 {
	return &f
}
