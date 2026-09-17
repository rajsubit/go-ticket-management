// Package ticket defines domain entities, business logic, repository contracts, and HTTP handlers.
//
// LEARNING GO: Business Logic, Dependency Injection, and Error Wrapping
// 1. Dependency Injection:
//    - Notice how `NewService(repo Repository)` accepts an interface rather than a concrete struct.
//    - This decouples business rules from how or where data is saved (in memory, PostgreSQL, etc.).
// 2. Error Wrapping (`%w`):
//    - `fmt.Errorf("failed to ...: %w", err)` creates a wrapped error chain.
//    - This gives descriptive context in log messages while preserving the ability for
//      callers to check `errors.Is(err, errors.ErrNotFound)`.
// 3. Pointer Fields for Partial Updates:
//    - In Go, a string defaults to `""` (zero-value).
//    - To distinguish between "user didn't specify a field" vs "user sent an empty string",
//      we use pointers (`*string`). If the pointer is `nil`, the field was omitted in the request.
package ticket

import (
	"fmt"
	"strings"

	"ticket-management/internal/platform/errors"
)

// Service encapsulates core ticket business logic.
type Service struct {
	repo Repository
}

// NewService creates a new ticket Service with the provided Repository implementation.
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// Create creates a new ticket after validating inputs and setting initial state.
func (s *Service) Create(req CreateTicketRequest) (Ticket, error) {
	if err := req.Validate(); err != nil {
		return Ticket{}, fmt.Errorf("validating create request: %w", err)
	}

	t := Ticket{
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Status:      StatusOpen, // All tickets start in OPEN status
		Priority:    req.Priority,
		Assignee:    strings.TrimSpace(req.Assignee),
	}

	if err := s.repo.Create(&t); err != nil {
		return Ticket{}, fmt.Errorf("creating ticket in repository: %w", err)
	}

	return t, nil
}

// GetByID fetches a ticket by its ID.
func (s *Service) GetByID(id string) (Ticket, error) {
	if strings.TrimSpace(id) == "" {
		return Ticket{}, errors.ErrInvalidInput
	}

	t, err := s.repo.GetByID(id)
	if err != nil {
		return Ticket{}, fmt.Errorf("retrieving ticket %s: %w", id, err)
	}

	return t, nil
}

// List returns tickets matching any provided filter options.
func (s *Service) List(filter FilterOptions) ([]Ticket, error) {
	if filter.Status != "" && !filter.Status.IsValid() {
		return nil, fmt.Errorf("invalid status filter '%s': %w", filter.Status, errors.ErrInvalidInput)
	}
	if filter.Priority != "" && !filter.Priority.IsValid() {
		return nil, fmt.Errorf("invalid priority filter '%s': %w", filter.Priority, errors.ErrInvalidInput)
	}

	tickets, err := s.repo.List(filter)
	if err != nil {
		return nil, fmt.Errorf("listing tickets: %w", err)
	}

	return tickets, nil
}

// Update modifies allowed fields on an existing ticket.
func (s *Service) Update(id string, req UpdateTicketRequest) (Ticket, error) {
	if strings.TrimSpace(id) == "" {
		return Ticket{}, errors.ErrInvalidInput
	}

	current, err := s.repo.GetByID(id)
	if err != nil {
		return Ticket{}, fmt.Errorf("fetching ticket %s for update: %w", id, err)
	}

	// Apply partial updates if non-nil
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if len(trimmed) < 3 || len(trimmed) > 100 {
			return Ticket{}, fmt.Errorf("title must be between 3 and 100 characters: %w", errors.ErrInvalidInput)
		}
		current.Title = trimmed
	}

	if req.Description != nil {
		current.Description = strings.TrimSpace(*req.Description)
	}

	if req.Priority != nil {
		if !req.Priority.IsValid() {
			return Ticket{}, fmt.Errorf("invalid priority: %w", errors.ErrInvalidInput)
		}
		current.Priority = *req.Priority
	}

	if req.Assignee != nil {
		current.Assignee = strings.TrimSpace(*req.Assignee)
	}

	if err := s.repo.Update(current); err != nil {
		return Ticket{}, fmt.Errorf("persisting updated ticket %s: %w", id, err)
	}

	return current, nil
}

// UpdateStatus changes the ticket status while enforcing domain state machine rules.
func (s *Service) UpdateStatus(id string, target Status) (Ticket, error) {
	if !target.IsValid() {
		return Ticket{}, fmt.Errorf("invalid target status '%s': %w", target, errors.ErrInvalidInput)
	}

	current, err := s.repo.GetByID(id)
	if err != nil {
		return Ticket{}, fmt.Errorf("fetching ticket %s for status update: %w", id, err)
	}

	if !current.CanTransitionTo(target) {
		return Ticket{}, fmt.Errorf("cannot transition from %s to %s: %w", current.Status, target, errors.ErrInvalidStatusTransition)
	}

	current.Status = target
	if err := s.repo.Update(current); err != nil {
		return Ticket{}, fmt.Errorf("persisting status update for %s: %w", id, err)
	}

	return current, nil
}

// Delete removes a ticket by ID.
func (s *Service) Delete(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.ErrInvalidInput
	}

	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("deleting ticket %s: %w", id, err)
	}

	return nil
}

// GetMetrics returns statistics on tickets.
func (s *Service) GetMetrics() (MetricsSummary, error) {
	metrics, err := s.repo.GetMetrics()
	if err != nil {
		return MetricsSummary{}, fmt.Errorf("calculating metrics: %w", err)
	}
	return metrics, nil
}
