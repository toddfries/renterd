package sql

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"unicode/utf8"

	"go.sia.tech/renterd/internal/utils"
	"go.uber.org/zap"
)

type (
	Migration struct {
		ID      string
		Migrate func(tx Tx) error
	}

	// Migrator is an interface for defining database-specific helper methods
	// required during migrations
	Migrator interface {
		ApplyMigration(ctx context.Context, fn func(tx Tx) (bool, error)) error
		CreateMigrationTable(ctx context.Context) error
		DB() *DB
	}

	MainMigrator interface {
		Migrator
		MakeDirsForPath(ctx context.Context, tx Tx, path string) (int64, error)
	}
)

var (
	MainMigrations = func(ctx context.Context, m MainMigrator, migrationsFs embed.FS, log *zap.SugaredLogger) []Migration {
		dbIdentifier := "main"
		return []Migration{
			{
				ID:      "00001_init",
				Migrate: func(tx Tx) error { return ErrRunV072 },
			},
			{
				ID: "00001_object_metadata",
				Migrate: func(tx Tx) error {
					return performMigration(ctx, tx, migrationsFs, dbIdentifier, "00001_object_metadata", log)
				},
			},
			{
				ID: "00002_prune_slabs_trigger",
				Migrate: func(tx Tx) error {
					err := performMigration(ctx, tx, migrationsFs, dbIdentifier, "00002_prune_slabs_trigger", log)
					if utils.IsErr(err, ErrMySQLNoSuperPrivilege) {
						log.Warn("migration 00002_prune_slabs_trigger requires the user to have the SUPER privilege to register triggers")
					}
					return err
				},
			},
			{
				ID: "00003_idx_objects_size",
				Migrate: func(tx Tx) error {
					return performMigration(ctx, tx, migrationsFs, dbIdentifier, "00003_idx_objects_size", log)
				},
			},
			{
				ID: "00004_prune_slabs_cascade",
				Migrate: func(tx Tx) error {
					return performMigration(ctx, tx, migrationsFs, dbIdentifier, "00004_prune_slabs_cascade", log)
				},
			},
			{
				ID: "00005_zero_size_object_health",
				Migrate: func(tx Tx) error {
					return performMigration(ctx, tx, migrationsFs, dbIdentifier, "00005_zero_size_object_health", log)
				},
			},
			{
				ID: "00006_idx_objects_created_at",
				Migrate: func(tx Tx) error {
					return performMigration(ctx, tx, migrationsFs, dbIdentifier, "00006_idx_objects_created_at", log)
				},
			},
			{
				ID: "00007_host_checks",
				Migrate: func(tx Tx) error {
					return performMigration(ctx, tx, migrationsFs, dbIdentifier, "00007_host_checks", log)
				},
			},
			{
				ID: "00008_directories",
				Migrate: func(tx Tx) error {
					if err := performMigration(ctx, tx, migrationsFs, dbIdentifier, "00008_directories_1", log); err != nil {
						return fmt.Errorf("failed to migrate: %v", err)
					}
					// helper type
					type obj struct {
						ID       uint
						ObjectID string
					}
					// loop over all objects and deduplicate dirs to create
					log.Info("beginning post-migration directory creation, this might take a while")
					batchSize := 10000
					processedDirs := make(map[string]struct{})
					for offset := 0; ; offset += batchSize {
						if offset > 0 && offset%batchSize == 0 {
							log.Infof("processed %v objects", offset)
						}
						var objBatch []obj
						rows, err := tx.Query(ctx, "SELECT id, object_id FROM objects ORDER BY id LIMIT ? OFFSET ?", batchSize, offset)
						if err != nil {
							return fmt.Errorf("failed to fetch objects: %v", err)
						}
						for rows.Next() {
							var o obj
							if err := rows.Scan(&o.ID, &o.ObjectID); err != nil {
								_ = rows.Close()
								return fmt.Errorf("failed to scan object: %v", err)
							}
							objBatch = append(objBatch, o)
						}
						if err := rows.Close(); err != nil {
							return fmt.Errorf("failed to close rows: %v", err)
						}
						if len(objBatch) == 0 {
							break // done
						}
						for _, obj := range objBatch {
							// check if dir was processed
							dir := "" // root
							if i := strings.LastIndex(obj.ObjectID, "/"); i > -1 {
								dir = obj.ObjectID[:i+1]
							}
							_, exists := processedDirs[dir]
							if exists {
								continue // already processed
							}
							processedDirs[dir] = struct{}{}

							// process
							dirID, err := m.MakeDirsForPath(ctx, tx, obj.ObjectID)
							if err != nil {
								return fmt.Errorf("failed to create directory %s: %w", obj.ObjectID, err)
							}

							if _, err := tx.Exec(ctx, `
							UPDATE objects
							SET db_directory_id = ?
							WHERE object_id LIKE ? AND
							SUBSTR(object_id, 1, ?) = ? AND
							INSTR(SUBSTR(object_id, ?), '/') = 0
						`,
								dirID,
								dir+"%",
								utf8.RuneCountInString(dir), dir,
								utf8.RuneCountInString(dir)+1); err != nil {
								return fmt.Errorf("failed to update object %s: %w", obj.ObjectID, err)
							}
						}
					}
					log.Info("post-migration directory creation complete")
					if err := performMigration(ctx, tx, migrationsFs, dbIdentifier, "00008_directories_2", log); err != nil {
						return fmt.Errorf("failed to migrate: %v", err)
					}
					return nil
				},
			},
			{
				ID: "00009_json_settings",
				Migrate: func(tx Tx) error {
					return performMigration(ctx, tx, migrationsFs, dbIdentifier, "00009_json_settings", log)
				},
			},
			{
				ID: "00010_webhook_headers",
				Migrate: func(tx Tx) error {
					return performMigration(ctx, tx, migrationsFs, dbIdentifier, "00010_webhook_headers", log)
				},
			},
		}
	}
	MetricsMigrations = func(ctx context.Context, migrationsFs embed.FS, log *zap.SugaredLogger) []Migration {
		dbIdentifier := "metrics"
		return []Migration{
			{
				ID:      "00001_init",
				Migrate: func(tx Tx) error { return ErrRunV072 },
			},
			{
				ID: "00001_idx_contracts_fcid_timestamp",
				Migrate: func(tx Tx) error {
					return performMigration(ctx, tx, migrationsFs, dbIdentifier, "00001_idx_contracts_fcid_timestamp", log)
				},
			},
		}
	}
)

func PerformMigrations(ctx context.Context, m Migrator, fs embed.FS, identifier string, migrations []Migration) error {
	// try to create migrations table
	err := m.CreateMigrationTable(ctx)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// check if the migrations table is empty
	var isEmpty bool
	if err := m.DB().QueryRow(ctx, "SELECT COUNT(*) = 0 FROM migrations").Scan(&isEmpty); err != nil {
		return fmt.Errorf("failed to count rows in migrations table: %w", err)
	} else if isEmpty {
		// table is empty, init schema
		return initSchema(ctx, m.DB(), fs, identifier, migrations)
	}

	// apply missing migrations
	for _, migration := range migrations {
		if err := m.ApplyMigration(ctx, func(tx Tx) (bool, error) {
			// check if migration was already applied
			var applied bool
			if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM migrations WHERE id = ?)", migration.ID).Scan(&applied); err != nil {
				return false, fmt.Errorf("failed to check if migration '%s' was already applied: %w", migration.ID, err)
			} else if applied {
				return false, nil
			}
			// run migration
			if err := migration.Migrate(tx); err != nil {
				return false, fmt.Errorf("migration '%s' failed: %w", migration.ID, err)
			}
			// insert migration
			if _, err := tx.Exec(ctx, "INSERT INTO migrations (id) VALUES (?)", migration.ID); err != nil {
				return false, fmt.Errorf("failed to insert migration '%s': %w", migration.ID, err)
			}
			return true, nil
		}); err != nil {
			return fmt.Errorf("migration '%s' failed: %w", migration.ID, err)
		}
	}
	return nil
}

func execSQLFile(ctx context.Context, tx Tx, fs embed.FS, folder, filename string) error {
	path := fmt.Sprintf("migrations/%s/%s.sql", folder, filename)

	// read file
	file, err := fs.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	// execute it
	if _, err := tx.Exec(ctx, string(file)); err != nil {
		return fmt.Errorf("failed to execute %s: %w", path, err)
	}
	return nil
}

func initSchema(ctx context.Context, db *DB, fs embed.FS, identifier string, migrations []Migration) error {
	return db.Transaction(ctx, func(tx Tx) error {
		// init schema
		if err := execSQLFile(ctx, tx, fs, identifier, "schema"); err != nil {
			return fmt.Errorf("failed to execute schema: %w", err)
		}
		// insert migration ids
		for _, migration := range migrations {
			if _, err := tx.Exec(ctx, "INSERT INTO migrations (id) VALUES (?)", migration.ID); err != nil {
				return fmt.Errorf("failed to insert migration '%s': %w", migration.ID, err)
			}
		}
		return nil
	})
}

func performMigration(ctx context.Context, tx Tx, fs embed.FS, kind, migration string, logger *zap.SugaredLogger) error {
	logger.Infof("performing %s migration '%s'", kind, migration)
	if err := execSQLFile(ctx, tx, fs, kind, fmt.Sprintf("migration_%s", migration)); err != nil {
		return err
	}
	logger.Infof("migration '%s' complete", migration)
	return nil
}
