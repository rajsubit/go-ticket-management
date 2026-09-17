// Package database provides connection management, pooling, and schema initialization for PostgreSQL.
//
// LEARNING GO: database/sql, Drivers, and Connection Pooling
// 1. What is `*sql.DB`?
//    - In Go, `*sql.DB` is NOT an individual database connection. It is an abstract, thread-safe CONNECTION POOL.
//    - You do not open and close a connection for each query. You create one `*sql.DB` when your application
//      starts and share it across all goroutines.
// 2. Blank Import (`_ "github.com/jackc/pgx/v5/stdlib"`):
//    - The underscore `_` import executes the driver's `init()` function, registering the "pgx" driver
//      with Go's `database/sql` registry without explicitly importing any exported symbols from it.
// 3. `sql.Open` does NOT verify connection:
//    - `sql.Open` only validates the connection string format.
//    - Always call `db.PingContext(ctx)` immediately afterward to verify that the PostgreSQL server
//      is alive and accepting credentials.
// 4. Connection Pool Tuning:
//    - `SetMaxOpenConns`: Caps max connections to prevent overwhelming PostgreSQL.
//    - `SetMaxIdleConns`: Maintains idle connections to eliminate connection handshake latency.
//    - `SetConnMaxLifetime`: Closes connections older than a duration to prevent stale socket states.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Config encapsulates parameters required to establish a PostgreSQL connection pool.
type Config struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DefaultConfig returns base connection pool settings without credentials.
// Sensitive credentials must be provided via .env or environment variables.
func DefaultConfig() Config {
	return Config{
		Host:            "localhost",
		Port:            "5432",
		User:            "",
		Password:        "",
		DBName:          "",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 15 * time.Minute,
	}
}

// DSN (Data Source Name) constructs the standard PostgreSQL connection URI.
func (c Config) DSN() string {
	var userPass string
	if c.Password != "" {
		userPass = fmt.Sprintf("%s:%s@", url.QueryEscape(c.User), url.QueryEscape(c.Password))
	} else {
		userPass = fmt.Sprintf("%s@", url.QueryEscape(c.User))
	}

	return fmt.Sprintf("postgres://%s%s:%s/%s?sslmode=%s",
		userPass,
		c.Host,
		c.Port,
		c.DBName,
		c.SSLMode,
	)
}

// Connect initializes the PostgreSQL connection pool, validates connectivity, and applies pool limits.
func Connect(cfg Config) (*sql.DB, error) {
	dsn := cfg.DSN()

	// Open connection pool using the registered "pgx" driver
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Configure pool parameters
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify database is accessible with a 5-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pinging postgres at %s:%s (user: %s, db: %s): %w",
			cfg.Host, cfg.Port, cfg.User, cfg.DBName, err)
	}

	log.Printf("[DATABASE] Connected to PostgreSQL (%s:%s/%s as %s)",
		cfg.Host, cfg.Port, cfg.DBName, cfg.User)

	return db, nil
}

// EnsureSchema automatically applies required tables and indexes if they do not exist.
func EnsureSchema(ctx context.Context, db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS tickets (
		id VARCHAR(36) PRIMARY KEY,
		title VARCHAR(100) NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		status VARCHAR(20) NOT NULL,
		priority VARCHAR(20) NOT NULL,
		assignee VARCHAR(100) NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);
	CREATE INDEX IF NOT EXISTS idx_tickets_priority ON tickets(priority);
	CREATE INDEX IF NOT EXISTS idx_tickets_created_at ON tickets(created_at DESC);
	`

	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("executing schema initialization: %w", err)
	}

	log.Println("[DATABASE] Verified tickets table and indexes exist in PostgreSQL")
	return nil
}
