// Package tests contains all test suites organized in a single folder for easy access.
package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"ticket-management/internal/platform/config"
	"ticket-management/internal/platform/database"
	platformErrors "ticket-management/internal/platform/errors"
	"ticket-management/internal/ticket"
)

// setupPostgresTest initializes a connection to PostgreSQL if available, or skips the test.
func setupPostgresTest(t *testing.T) *ticket.PostgresRepository {
	t.Helper()

	cfg := config.Load()
	dbCfg := database.Config{
		Host:            cfg.DB.Host,
		Port:            cfg.DB.Port,
		User:            cfg.DB.User,
		Password:        cfg.DB.Password,
		DBName:          cfg.DB.DBName,
		SSLMode:         cfg.DB.SSLMode,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 1 * time.Minute,
	}

	db, err := database.Connect(dbCfg)
	if err != nil {
		t.Skipf("PostgreSQL is not reachable (%v). Skipping database integration tests.", err)
		return nil
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	// Ensure schema exists
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := database.EnsureSchema(ctx, db); err != nil {
		t.Fatalf("failed to initialize schema for test: %v", err)
	}

	return ticket.NewPostgresRepository(db)
}

func TestPostgresRepositoryCRUD(t *testing.T) {
	repo := setupPostgresTest(t)
	if repo == nil {
		return
	}

	// Track test tickets for cleanup
	testTicketIDs := make([]string, 0)
	t.Cleanup(func() {
		for _, id := range testTicketIDs {
			_ = repo.Delete(id)
		}
	})

	// 1. Test Create
	testTicket := &ticket.Ticket{
		Title:       "Postgres Integration Test Ticket",
		Description: "Verifying insert, select, update, and delete in PostgreSQL",
		Status:      ticket.StatusOpen,
		Priority:    ticket.PriorityHigh,
		Assignee:    "Subit",
	}

	if err := repo.Create(testTicket); err != nil {
		t.Fatalf("failed to create ticket in postgres: %v", err)
	}
	testTicketIDs = append(testTicketIDs, testTicket.ID)

	if testTicket.ID == "" {
		t.Fatalf("expected generated ticket ID, got empty string")
	}

	// 2. Test GetByID
	fetched, err := repo.GetByID(testTicket.ID)
	if err != nil {
		t.Fatalf("failed to fetch created ticket %s: %v", testTicket.ID, err)
	}
	if fetched.Title != testTicket.Title {
		t.Errorf("expected title '%s', got '%s'", testTicket.Title, fetched.Title)
	}
	if fetched.Status != ticket.StatusOpen {
		t.Errorf("expected status %s, got %s", ticket.StatusOpen, fetched.Status)
	}

	// 3. Test List with filters
	list, err := repo.List(ticket.FilterOptions{Status: ticket.StatusOpen, Priority: ticket.PriorityHigh})
	if err != nil {
		t.Fatalf("failed to list tickets: %v", err)
	}
	found := false
	for _, item := range list {
		if item.ID == testTicket.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find test ticket %s in filtered list", testTicket.ID)
	}

	// 4. Test Update
	testTicket.Title = "Updated Postgres Ticket Title"
	testTicket.Status = ticket.StatusInProgress
	if err := repo.Update(*testTicket); err != nil {
		t.Fatalf("failed to update ticket in postgres: %v", err)
	}

	updated, err := repo.GetByID(testTicket.ID)
	if err != nil {
		t.Fatalf("failed to fetch updated ticket: %v", err)
	}
	if updated.Title != "Updated Postgres Ticket Title" {
		t.Errorf("expected updated title, got '%s'", updated.Title)
	}
	if updated.Status != ticket.StatusInProgress {
		t.Errorf("expected status %s, got %s", ticket.StatusInProgress, updated.Status)
	}

	// 5. Test GetMetrics
	metrics, err := repo.GetMetrics()
	if err != nil {
		t.Fatalf("failed to get metrics: %v", err)
	}
	if metrics.Total < 1 {
		t.Errorf("expected at least 1 total ticket, got %d", metrics.Total)
	}
	if metrics.InProgress < 1 {
		t.Errorf("expected at least 1 in_progress ticket, got %d", metrics.InProgress)
	}

	// 6. Test Delete
	if err := repo.Delete(testTicket.ID); err != nil {
		t.Fatalf("failed to delete ticket: %v", err)
	}

	// 7. Verify ErrNotFound on deleted ticket
	_, err = repo.GetByID(testTicket.ID)
	if err == nil {
		t.Fatalf("expected error retrieving deleted ticket, got nil")
	}
	if !errors.Is(err, platformErrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
