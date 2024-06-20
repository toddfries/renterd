package postgres

import (
    "context"
    "embed"
    "fmt"

    "github.com/jackc/pgx/v5"
)

var deadlockMsgs = []string{
    "deadlock detected",
}

//go:embed all:migrations/*
var migrationsFs embed.FS

func applyMigration(ctx context.Context, db *pgx.Conn, fn func(tx pgx.Tx) (bool, error)) error {
    return db.Begin(ctx, func(tx pgx.Tx) error {
        _, err := fn(tx)
        return err
    })
}

func createMigrationTable(ctx context.Context, db *pgx.Conn) error {
    if _, err := db.Exec(ctx, `
            CREATE TABLE IF NOT EXISTS migrations (
                id varchar(255) NOT NULL,
                PRIMARY KEY (id)
            );`); err != nil {
        return fmt.Errorf("failed to create migrations table: %w", err)
    }
    return nil
}

func version(ctx context.Context, db *pgx.Conn) (string, string, error) {
    var version string
    if err := db.QueryRow(ctx, "select version()").Scan(&version); err != nil {
        return "", "", err
    }
    return "PostgreSQL", version, nil
}
