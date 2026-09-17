// Package ticket_test contains unit tests for the ticket package.
//
// LEARNING GO: Idiomatic Testing & Table-Driven Tests
// 1. Package Naming:
//    - Using `package ticket_test` (black-box testing) tests the package only through its public API,
//      just like an external caller would.
// 2. The `testing.T` type:
//    - `t.Fatalf(...)`: Logs the failure and STOPS this test immediately.
//    - `t.Errorf(...)`: Logs the failure but continues executing remaining assertions.
//    - `t.Run(name, func(t *testing.T))`: Creates an isolated subtest with its own pass/fail status.
// 3. Table-Driven Tests (The Go Standard):
//    - Slices of anonymous structs holding test cases (`testCases := []struct{...}{...}`).
//    - Eliminates copy-paste code and makes adding new edge cases trivial.
package ticket_test

import (
	"errors"
	"testing"

	platformErrors "ticket-management/internal/platform/errors"
	"ticket-management/internal/ticket"
)

// TestCreateTicket demonstrates table-driven testing for validating input and creating tickets.
func TestCreateTicket(t *testing.T) {
	repo := ticket.NewInMemoryRepository()
	svc := ticket.NewService(repo)

	testCases := []struct {
		name        string
		req         ticket.CreateTicketRequest
		expectError bool
		expectedErr error
	}{
		{
			name: "Valid ticket creation with high priority",
			req: ticket.CreateTicketRequest{
				Title:       "Fix payment webhook bug",
				Description: "Stripe webhook times out during charge.failed",
				Priority:    ticket.PriorityHigh,
				Assignee:    "Alice",
			},
			expectError: false,
		},
		{
			name: "Valid ticket without assignee",
			req: ticket.CreateTicketRequest{
				Title:       "Update landing page copyright year",
				Description: "Change 2025 to 2026",
				Priority:    ticket.PriorityLow,
			},
			expectError: false,
		},
		{
			name: "Fails with empty title",
			req: ticket.CreateTicketRequest{
				Title:    "   ",
				Priority: ticket.PriorityMedium,
			},
			expectError: true,
			expectedErr: platformErrors.ErrInvalidInput,
		},
		{
			name: "Fails with title shorter than 3 characters",
			req: ticket.CreateTicketRequest{
				Title:    "ab",
				Priority: ticket.PriorityMedium,
			},
			expectError: true,
			expectedErr: platformErrors.ErrInvalidInput,
		},
		{
			name: "Fails with invalid priority",
			req: ticket.CreateTicketRequest{
				Title:    "Legitimate title",
				Priority: ticket.Priority("SUPER_URGENT"), // Not a valid domain priority
			},
			expectError: true,
			expectedErr: platformErrors.ErrInvalidInput,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			created, err := svc.Create(tc.req)

			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.expectedErr != nil && !errors.Is(err, tc.expectedErr) {
					t.Fatalf("expected error matching %v, got %v", tc.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error creating ticket: %v", err)
			}

			if created.ID == "" {
				t.Errorf("expected generated ticket ID, got empty string")
			}
			if created.Status != ticket.StatusOpen {
				t.Errorf("expected initial status %s, got %s", ticket.StatusOpen, created.Status)
			}
			if created.Title != tc.req.Title {
				t.Errorf("expected title '%s', got '%s'", tc.req.Title, created.Title)
			}
		})
	}
}

// TestStatusTransitions tests legal and illegal lifecycle state changes.
func TestStatusTransitions(t *testing.T) {
	repo := ticket.NewInMemoryRepository()
	svc := ticket.NewService(repo)

	// Setup: Create a base ticket in OPEN status
	created, err := svc.Create(ticket.CreateTicketRequest{
		Title:    "Database performance tuning",
		Priority: ticket.PriorityHigh,
	})
	if err != nil {
		t.Fatalf("failed to setup test ticket: %v", err)
	}

	// 1. OPEN -> IN_PROGRESS (Allowed)
	updated, err := svc.UpdateStatus(created.ID, ticket.StatusInProgress)
	if err != nil {
		t.Fatalf("expected transition from OPEN to IN_PROGRESS to succeed, got: %v", err)
	}
	if updated.Status != ticket.StatusInProgress {
		t.Errorf("expected status %s, got %s", ticket.StatusInProgress, updated.Status)
	}

	// 2. IN_PROGRESS -> RESOLVED (Allowed)
	updated, err = svc.UpdateStatus(created.ID, ticket.StatusResolved)
	if err != nil {
		t.Fatalf("expected transition from IN_PROGRESS to RESOLVED to succeed, got: %v", err)
	}
	if updated.Status != ticket.StatusResolved {
		t.Errorf("expected status %s, got %s", ticket.StatusResolved, updated.Status)
	}

	// 3. RESOLVED -> OPEN (Disallowed: must go to IN_PROGRESS or CLOSED)
	_, err = svc.UpdateStatus(created.ID, ticket.StatusOpen)
	if err == nil {
		t.Fatalf("expected illegal transition from RESOLVED to OPEN to fail, but it succeeded")
	}
	if !errors.Is(err, platformErrors.ErrInvalidStatusTransition) {
		t.Errorf("expected ErrInvalidStatusTransition, got: %v", err)
	}

	// 4. RESOLVED -> CLOSED (Allowed)
	updated, err = svc.UpdateStatus(created.ID, ticket.StatusClosed)
	if err != nil {
		t.Fatalf("expected transition from RESOLVED to CLOSED to succeed, got: %v", err)
	}
	if updated.Status != ticket.StatusClosed {
		t.Errorf("expected status %s, got %s", ticket.StatusClosed, updated.Status)
	}

	// 5. CLOSED -> OPEN (Allowed: Reopening ticket)
	updated, err = svc.UpdateStatus(created.ID, ticket.StatusOpen)
	if err != nil {
		t.Fatalf("expected reopening ticket from CLOSED to OPEN to succeed, got: %v", err)
	}
	if updated.Status != ticket.StatusOpen {
		t.Errorf("expected status %s, got %s", ticket.StatusOpen, updated.Status)
	}
}

// TestGetByIDAndNotFound tests fetching an existing ticket vs unknown ID.
func TestGetByIDAndNotFound(t *testing.T) {
	repo := ticket.NewInMemoryRepository()
	svc := ticket.NewService(repo)

	created, err := svc.Create(ticket.CreateTicketRequest{
		Title:    "Sample bug report",
		Priority: ticket.PriorityLow,
	})
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	// Successful retrieval
	found, err := svc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("expected to find ticket %s, got error: %v", created.ID, err)
	}
	if found.ID != created.ID {
		t.Errorf("expected id %s, got %s", created.ID, found.ID)
	}

	// Non-existent ID retrieval
	_, err = svc.GetByID("non-existent-id-999")
	if err == nil {
		t.Fatalf("expected error for non-existent ticket, got nil")
	}
	if !errors.Is(err, platformErrors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

// TestMetricsCalculation tests ticket count aggregation.
func TestMetricsCalculation(t *testing.T) {
	repo := ticket.NewInMemoryRepository()
	svc := ticket.NewService(repo)

	// Create 3 tickets: 1 will stay OPEN, 1 will become IN_PROGRESS, 1 will become RESOLVED
	t1, _ := svc.Create(ticket.CreateTicketRequest{Title: "Task 1", Priority: ticket.PriorityLow})
	t2, _ := svc.Create(ticket.CreateTicketRequest{Title: "Task 2", Priority: ticket.PriorityMedium})
	_, _ = svc.Create(ticket.CreateTicketRequest{Title: "Task 3", Priority: ticket.PriorityHigh})

	_, _ = svc.UpdateStatus(t1.ID, ticket.StatusInProgress)
	_, _ = svc.UpdateStatus(t2.ID, ticket.StatusInProgress)
	_, _ = svc.UpdateStatus(t2.ID, ticket.StatusResolved)

	metrics, err := svc.GetMetrics()
	if err != nil {
		t.Fatalf("unexpected error fetching metrics: %v", err)
	}

	if metrics.Total != 3 {
		t.Errorf("expected total 3, got %d", metrics.Total)
	}
	if metrics.Open != 1 {
		t.Errorf("expected 1 open ticket, got %d", metrics.Open)
	}
	if metrics.InProgress != 1 {
		t.Errorf("expected 1 in_progress ticket, got %d", metrics.InProgress)
	}
	if metrics.Resolved != 1 {
		t.Errorf("expected 1 resolved ticket, got %d", metrics.Resolved)
	}
}
