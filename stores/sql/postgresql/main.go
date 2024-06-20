package sqlstore

import (
    "context"
    "database/sql"
    "fmt"
    "log"

    _ "github.com/lib/pq"
)

type PostgresStore struct {
    db *sql.DB
}

func NewPostgresStore(dsn string) (*PostgresStore, error) {
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    err = db.Ping()
    if err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Close() error {
    return s.db.Close()
}

func (s *PostgresStore) CreateRenter(ctx context.Context, renter Renter) error {
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO renters (id, name, contract_id, wallet_address)
        VALUES ($1, $2, $3, $4)
        RETURNING id;
    `, renter.ID, renter.Name, renter.ContractID, renter.WalletAddress)
    return err
}

func (s *PostgresStore) GetRenter(ctx context.Context, id string) (*Renter, error) {
    row := s.db.QueryRowContext(ctx, `
        SELECT id, name, contract_id, wallet_address
        FROM renters
        WHERE id = $1;
    `, id)

    var renter Renter
    err := row.Scan(&renter.ID, &renter.Name, &renter.ContractID, &renter.WalletAddress)
    if err == sql.ErrNoRows {
        return nil, nil
    } else if err != nil {
        return nil, fmt.Errorf("failed to scan row: %w", err)
    }

    return &renter, nil
}

func (s *PostgresStore) UpdateRenter(ctx context.Context, renter Renter) error {
    _, err := s.db.ExecContext(ctx, `
        UPDATE renters
        SET name = $1, contract_id = $2, wallet_address = $3
        WHERE id = $4;
    `, renter.Name, renter.ContractID, renter.WalletAddress, renter.ID)
    return err
}

func (s *PostgresStore) DeleteRenter(ctx context.Context, id string) error {
    _, err := s.db.ExecContext(ctx, `
        DELETE FROM renters
        WHERE id = $1;
    `, id)
    return err
}
