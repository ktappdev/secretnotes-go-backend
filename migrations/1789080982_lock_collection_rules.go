package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Locks down the `notes` and `encrypted_files` collections.
//
// The live server had all five access rules set to the empty string "" on both
// collections. In PocketBase, "" means PUBLIC — anyone on the internet could
// list, create, update and delete every record without any authentication.
// `null` means superuser-only, which is what we want here: the mobile client
// never touches these collections directly, it goes through the custom
// /api/secretnotes routes, and those use internal app.* calls that bypass
// collection rules entirely.
//
// Applied after 1789065331_collections_snapshot.go, which captured the
// permissive state.
func init() {
	m.Register(func(app core.App) error {
		for _, name := range []string{"notes", "encrypted_files"} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}

			collection.ListRule = nil
			collection.ViewRule = nil
			collection.CreateRule = nil
			collection.UpdateRule = nil
			collection.DeleteRule = nil

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Restore the previous (public) state captured by the snapshot migration.
		public := ""

		for _, name := range []string{"notes", "encrypted_files"} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}

			collection.ListRule = &public
			collection.ViewRule = &public
			collection.CreateRule = &public
			collection.UpdateRule = &public
			collection.DeleteRule = &public

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	})
}
