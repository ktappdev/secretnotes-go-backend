package migrations

import (
    "github.com/pocketbase/pocketbase/core"
    m "github.com/pocketbase/pocketbase/migrations"
)

// Adds explicit Autodate fields named "created" and "updated" to
// the "notes" and "encrypted_files" collections so they auto-populate
// on record create/update.
//
// Note: PocketBase has system timestamps, but this migration creates
// explicit Autodate schema fields with the exact names requested.
func init() {
    m.Register(func(app core.App) error {
        // notes collection
        notes, err := app.FindCollectionByNameOrId("notes")
        if err != nil {
            return err
        }
        // created: set on create only
        notes.Fields.Add(&core.AutodateField{
            Name:     "created",
            OnCreate: true,
        })
        // updated: set on create and on every update
        notes.Fields.Add(&core.AutodateField{
            Name:     "updated",
            OnCreate: true,
            OnUpdate: true,
        })
        if err := app.Save(notes); err != nil {
            return err
        }

        // encrypted_files collection
        files, err := app.FindCollectionByNameOrId("encrypted_files")
        if err != nil {
            return err
        }
        files.Fields.Add(&core.AutodateField{
            Name:     "created",
            OnCreate: true,
        })
        files.Fields.Add(&core.AutodateField{
            Name:     "updated",
            OnCreate: true,
            OnUpdate: true,
        })
        return app.Save(files)
    }, func(app core.App) error {
        // Down: remove the added fields if they exist
        if c, err := app.FindCollectionByNameOrId("notes"); err == nil {
            c.Fields.RemoveByName("created")
            c.Fields.RemoveByName("updated")
            if err := app.Save(c); err != nil {
                return err
            }
        }
        if c, err := app.FindCollectionByNameOrId("encrypted_files"); err == nil {
            c.Fields.RemoveByName("created")
            c.Fields.RemoveByName("updated")
            if err := app.Save(c); err != nil {
                return err
            }
        }
        return nil
    })
}
