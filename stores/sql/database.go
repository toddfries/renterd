package sql

import (
	"context"
	"io"
)

// The database interfaces define all methods that a SQL database must implement
// to be used by the SQLStore.
type (
	Database interface {
		io.Closer

		// Transaction starts a new transaction.
		Transaction(ctx context.Context, fn func(DatabaseTx) error) error

		// Migrate runs all missing migrations on the database.
		Migrate(ctx context.Context) error

		// Version returns the database version and name.
		Version(ctx context.Context) (string, string, error)
	}

	DatabaseTx interface {
		// DeleteObject deletes an object from the database and returns true if
		// the requested object was actually deleted.
		DeleteObject(ctx context.Context, bucket, key string) (bool, error)

		// DeleteObjects deletes a batch of objects starting with the given
		// prefix and returns 'true' if any object was deleted.
		DeleteObjects(ctx context.Context, bucket, prefix string, limit int64) (bool, error)

		// MakeDirsForPath creates all directories for a given object's path.
		MakeDirsForPath(ctx context.Context, path string) (uint, error)

		// PruneDirs prunes any directories that are empty.
		PruneDirs(ctx context.Context) error

		// PruneSlabs deletes slabs that are no longer referenced by any slice
		// or slab buffer.
		PruneSlabs(ctx context.Context, limit int64) (int64, error)

		// RenameObject renames an object in the database from keyOld to keyNew
		// and the new directory dirID. returns api.ErrObjectExists if the an
		// object already exists at the target location or api.ErrObjectNotFound
		// if the object at keyOld doesn't exist. If force is true, the instead
		// of returning api.ErrObjectExists, the existing object will be
		// deleted.
		RenameObject(ctx context.Context, bucket, keyOld, keyNew string, dirID uint, force bool) error
	}

	MetricsDatabase interface {
		io.Closer
		Migrate(ctx context.Context) error
		Version(ctx context.Context) (string, string, error)
	}
)
