package metrics

import (
    "database/sql"
    "fmt"
    "log"
    "time"

    _ "github.com/lib/pq" // PostgreSQL driver
)

type MetricsDB struct {
    db *sql.DB
}

func NewMetricsDB(dbURI string) (*MetricsDB, error) {
    db, err := sql.Open("postgres", dbURI)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %v", err)
    }

    err = db.Ping()
    if err != nil {
        return nil, fmt.Errorf("failed to ping database: %v", err)
    }

    return &MetricsDB{db: db}, nil
}

func (m *MetricsDB) Close() {
    m.db.Close()
}

func (m *MetricsDB) AddMetric(metric string, value float64) error {
    _, err := m.db.Exec("INSERT INTO metrics (metric, value, timestamp) VALUES ($1, $2, $3)", metric, value, time.Now())
    return err
}

func (m *MetricsDB) GetMetrics(metric string) ([]Metric, error) {
    rows, err := m.db.Query("SELECT value, timestamp FROM metrics WHERE metric = $1 ORDER BY timestamp DESC", metric)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var metrics []Metric
    for rows.Next() {
        var m Metric
        err := rows.Scan(&m.Value, &m.Timestamp)
        if err != nil {
            return nil, err
        }
        metrics = append(metrics, m)
    }

    if err := rows.Err(); err != nil {
        return nil, err
    }

    return metrics, nil
}

type Metric struct {
    Value     float64
    Timestamp time.Time
}

func (m *MetricsDB) Migrate() error {
    _, err := m.db.Exec(`
        CREATE TABLE IF NOT EXISTS metrics (
            id SERIAL PRIMARY KEY,
            metric VARCHAR(255) NOT NULL,
            value FLOAT NOT NULL,
            timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        );`)
    return err
}
