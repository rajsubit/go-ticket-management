// Package ticket defines domain entities, business logic, repository contracts, and HTTP handlers.
//
// LEARNING GO: SQL Repositories, Parameterized Queries, and sql.ErrNoRows
// 1. Interface Polymorphism:
//    - Notice how `PostgresRepository` implements the exact same `Repository` interface as `InMemoryRepository`.
//    - The `Service` layer does not care whether data is stored in RAM or PostgreSQL.
// 2. SQL Injection Prevention:
//    - NEVER concatenate variables into SQL strings (e.g. `SELECT ... WHERE id = '` + id + `'`).
//    - In PostgreSQL, use positional placeholders: `$1`, `$2`, `$3`.
//    - The database driver sanitizes arguments safely.
// 3. QueryRow vs Query vs Exec:
//    - `QueryRowContext`: Returns at most 1 row. Call `.Scan(&a, &b)`.
//    - `QueryContext`: Returns multiple rows. Iterate with `for rows.Next()`. Always `defer rows.Close()`.
//    - `ExecContext`: Executes INSERT/UPDATE/DELETE where you do not need row data back.
// 4. Checking `sql.ErrNoRows`:
//    - In Go, a missing row in `QueryRow` returns `sql.ErrNoRows`.
//    - We translate this into our domain `errors.ErrNotFound` so callers remain database-agnostic.
package ticket

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	platformErrors "ticket-management/internal/platform/errors"
)

// PostgresRepository persists tickets to a PostgreSQL database table.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository initializes a PostgresRepository with the given connection pool.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

// Create inserts a new ticket into PostgreSQL.
func (r *PostgresRepository) Create(ticket *Ticket) error {
	if ticket.ID == "" {
		ticket.ID = generateID()
	}

	now := time.Now().UTC()
	ticket.CreatedAt = now
	ticket.UpdatedAt = now

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		INSERT INTO tickets (id, title, description, status, priority, assignee, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		ticket.ID,
		ticket.Title,
		ticket.Description,
		string(ticket.Status),
		string(ticket.Priority),
		ticket.Assignee,
		ticket.CreatedAt,
		ticket.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting ticket into postgres: %w", err)
	}

	return nil
}

// GetByID queries a single ticket by its primary key.
func (r *PostgresRepository) GetByID(id string) (Ticket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT id, title, description, status, priority, assignee, created_at, updated_at
		FROM tickets
		WHERE id = $1
	`

	var t Ticket
	var statusStr, priorityStr string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID,
		&t.Title,
		&t.Description,
		&statusStr,
		&priorityStr,
		&t.Assignee,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Ticket{}, platformErrors.ErrNotFound
		}
		return Ticket{}, fmt.Errorf("querying ticket %s: %w", id, err)
	}

	t.Status = Status(statusStr)
	t.Priority = Priority(priorityStr)
	return t, nil
}

// List returns tickets matching dynamic filter options.
func (r *PostgresRepository) List(filter FilterOptions) ([]Ticket, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Build dynamic WHERE clause using numbered placeholders ($1, $2)
	query := "SELECT id, title, description, status, priority, assignee, created_at, updated_at FROM tickets"
	var conditions []string
	var args []any
	argIdx := 1

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(filter.Status))
		argIdx++
	}

	if filter.Priority != "" {
		conditions = append(conditions, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, string(filter.Priority))
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing tickets from postgres: %w", err)
	}
	defer rows.Close()

	var results []Ticket
	for rows.Next() {
		var t Ticket
		var statusStr, priorityStr string

		if err := rows.Scan(
			&t.ID,
			&t.Title,
			&t.Description,
			&statusStr,
			&priorityStr,
			&t.Assignee,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning ticket row: %w", err)
		}

		t.Status = Status(statusStr)
		t.Priority = Priority(priorityStr)
		results = append(results, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating ticket rows: %w", err)
	}

	if results == nil {
		results = []Ticket{} // Return empty slice instead of nil for clean JSON `[]` serialization
	}

	return results, nil
}

// Update updates an existing ticket record in PostgreSQL.
func (r *PostgresRepository) Update(ticket Ticket) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	now := time.Now().UTC()
	ticket.UpdatedAt = now

	query := `
		UPDATE tickets
		SET title = $1, description = $2, status = $3, priority = $4, assignee = $5, updated_at = $6
		WHERE id = $7
	`

	res, err := r.db.ExecContext(ctx, query,
		ticket.Title,
		ticket.Description,
		string(ticket.Status),
		string(ticket.Priority),
		ticket.Assignee,
		ticket.UpdatedAt,
		ticket.ID,
	)
	if err != nil {
		return fmt.Errorf("updating ticket %s: %w", ticket.ID, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for update %s: %w", ticket.ID, err)
	}

	if rowsAffected == 0 {
		return platformErrors.ErrNotFound
	}

	return nil
}

// Delete removes a ticket record by ID.
func (r *PostgresRepository) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := "DELETE FROM tickets WHERE id = $1"

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting ticket %s: %w", id, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for delete %s: %w", id, err)
	}

	if rowsAffected == 0 {
		return platformErrors.ErrNotFound
	}

	return nil
}

// GetMetrics computes aggregated counts using PostgreSQL's FILTER clause.
func (r *PostgresRepository) GetMetrics() (MetricsSummary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT
			count(*)::int AS total,
			count(*) FILTER (WHERE status = 'OPEN')::int AS open_count,
			count(*) FILTER (WHERE status = 'IN_PROGRESS')::int AS in_progress_count,
			count(*) FILTER (WHERE status = 'RESOLVED')::int AS resolved_count,
			count(*) FILTER (WHERE status = 'CLOSED')::int AS closed_count
		FROM tickets
	`

	var m MetricsSummary
	err := r.db.QueryRowContext(ctx, query).Scan(
		&m.Total,
		&m.Open,
		&m.InProgress,
		&m.Resolved,
		&m.Closed,
	)
	if err != nil {
		return MetricsSummary{}, fmt.Errorf("querying ticket metrics from postgres: %w", err)
	}

	return m, nil
}
