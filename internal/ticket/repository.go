// Package ticket defines domain entities, business logic, repository contracts, and HTTP handlers.
//
// LEARNING GO: Interfaces, Concurrency, and Mutexes
// 1. Interfaces:
//    - Go interfaces are satisfied IMPLICITLY. There is no `implements` keyword.
//    - If a struct implements all methods declared in `Repository`, it satisfies `Repository`.
//    - "Accept interfaces, return structs" is a classic Go design idiom.
// 2. Concurrency Safety (`sync.RWMutex`):
//    - Go maps (`map[string]Ticket`) are NOT thread-safe by default.
//    - An HTTP server in Go processes each incoming request in its own lightweight thread (goroutine).
//    - Without locking, simultaneous read/write operations will crash the program with `concurrent map writes`.
//    - `mu.RLock()` allows multiple simultaneous readers.
//    - `mu.Lock()` guarantees exclusive access for writers.
// 3. The `defer` statement:
//    - `defer r.mu.Unlock()` ensures the lock is released whenever the surrounding function exits,
//      even if an early `return` or error occurs.
package ticket

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"ticket-management/internal/platform/errors"
)

// Repository defines storage operations for tickets.
// By defining this interface, the business layer (Service) is decoupled from the storage layer (Memory, SQL, etc.).
type Repository interface {
	Create(ticket *Ticket) error
	GetByID(id string) (Ticket, error)
	List(filter FilterOptions) ([]Ticket, error)
	Update(ticket Ticket) error
	Delete(id string) error
	GetMetrics() (MetricsSummary, error)
}

// InMemoryRepository is a thread-safe in-memory implementation of Repository.
type InMemoryRepository struct {
	mu      sync.RWMutex
	tickets map[string]Ticket
}

// NewInMemoryRepository creates a new, ready-to-use InMemoryRepository.
// In Go, constructor functions typically follow the naming convention `New<Type>`.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		tickets: make(map[string]Ticket),
	}
}

// generateID generates a random 8-byte hexadecimal ID using standard crypto/rand.
func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("tkt-%x", b)
}

// Create stores a new ticket in the repository.
func (r *InMemoryRepository) Create(ticket *Ticket) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ticket.ID == "" {
		ticket.ID = generateID()
	}

	now := time.Now().UTC()
	ticket.CreatedAt = now
	ticket.UpdatedAt = now

	// Check for collision
	if _, exists := r.tickets[ticket.ID]; exists {
		return errors.ErrConflict
	}

	r.tickets[ticket.ID] = *ticket
	return nil
}

// GetByID retrieves a ticket by its unique identifier.
func (r *InMemoryRepository) GetByID(id string) (Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, exists := r.tickets[id]
	if !exists {
		return Ticket{}, errors.ErrNotFound
	}
	return t, nil
}

// List returns tickets matching the provided filter options.
func (r *InMemoryRepository) List(filter FilterOptions) ([]Ticket, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Ticket, 0, len(r.tickets))
	for _, t := range r.tickets {
		if filter.Status != "" && t.Status != filter.Status {
			continue
		}
		if filter.Priority != "" && t.Priority != filter.Priority {
			continue
		}
		result = append(result, t)
	}

	return result, nil
}

// Update updates an existing ticket.
func (r *InMemoryRepository) Update(ticket Ticket) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tickets[ticket.ID]; !exists {
		return errors.ErrNotFound
	}

	ticket.UpdatedAt = time.Now().UTC()
	r.tickets[ticket.ID] = ticket
	return nil
}

// Delete removes a ticket by ID.
func (r *InMemoryRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tickets[id]; !exists {
		return errors.ErrNotFound
	}

	delete(r.tickets, id)
	return nil
}

// GetMetrics returns counts of tickets grouped by status.
func (r *InMemoryRepository) GetMetrics() (MetricsSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var summary MetricsSummary
	summary.Total = len(r.tickets)

	for _, t := range r.tickets {
		switch t.Status {
		case StatusOpen:
			summary.Open++
		case StatusInProgress:
			summary.InProgress++
		case StatusResolved:
			summary.Resolved++
		case StatusClosed:
			summary.Closed++
		}
	}

	return summary, nil
}
