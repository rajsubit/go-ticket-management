// Package ticket defines domain entities, business logic, repository contracts, and HTTP handlers.
//
// LEARNING GO: Structs, Enums, Tags, and Methods
// 1. In Go, there is no `class` or `enum` keyword.
//    - We use `struct` for grouping data fields.
//    - We define custom types (`type Status string`) and `const` blocks to create type-safe enums.
// 2. Exported vs Unexported (Visibility):
//    - Identifiers starting with a Capital letter (e.g. `Ticket`, `Title`) are Exported (public).
//    - Identifiers starting with a lowercase letter (e.g. `validTransitions`) are unexported (private to this package).
// 3. Struct Tags:
//    - The backtick strings like `json:"title"` are struct field tags. They tell Go's JSON parser
//      how to map JSON keys when serializing (marshalling) and deserializing (unmarshalling).
// 4. Value vs Pointer Receivers:
//    - `func (s Status) IsValid() bool` -> Value receiver: receives a copy, cannot mutate caller.
//    - `func (t *Ticket) ...` -> Pointer receiver: operates on the original struct in memory.
package ticket

import (
	"strings"
	"time"

	"ticket-management/internal/platform/errors"
)

// Status represents the lifecycle state of a ticket.
type Status string

const (
	StatusOpen       Status = "OPEN"
	StatusInProgress Status = "IN_PROGRESS"
	StatusResolved   Status = "RESOLVED"
	StatusClosed     Status = "CLOSED"
)

// IsValid checks if the status is one of the supported domain states.
func (s Status) IsValid() bool {
	switch s {
	case StatusOpen, StatusInProgress, StatusResolved, StatusClosed:
		return true
	default:
		return false
	}
}

// Priority defines the urgency level of a ticket.
type Priority string

const (
	PriorityLow      Priority = "LOW"
	PriorityMedium   Priority = "MEDIUM"
	PriorityHigh     Priority = "HIGH"
	PriorityCritical Priority = "CRITICAL"
)

// IsValid checks if the priority is valid.
func (p Priority) IsValid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
		return true
	default:
		return false
	}
}

// Ticket represents a single issue, bug, or task in the system.
type Ticket struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	Priority    Priority  `json:"priority"`
	Assignee    string    `json:"assignee,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateTicketRequest represents the payload required to create a new ticket.
type CreateTicketRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    Priority `json:"priority"`
	Assignee    string   `json:"assignee,omitempty"`
}

// Validate checks whether the create request meets all business rules.
func (r *CreateTicketRequest) Validate() error {
	trimmedTitle := strings.TrimSpace(r.Title)
	if trimmedTitle == "" {
		return errors.ErrInvalidInput
	}
	if len(trimmedTitle) < 3 || len(trimmedTitle) > 100 {
		return errors.ErrInvalidInput
	}
	if !r.Priority.IsValid() {
		return errors.ErrInvalidInput
	}
	return nil
}

// UpdateTicketRequest represents allowed fields when updating a ticket.
type UpdateTicketRequest struct {
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	Priority    *Priority `json:"priority,omitempty"`
	Assignee    *string   `json:"assignee,omitempty"`
}

// UpdateStatusRequest represents payload to change a ticket's status.
type UpdateStatusRequest struct {
	Status Status `json:"status"`
}

// FilterOptions allows filtering tickets when listing.
type FilterOptions struct {
	Status   Status
	Priority Priority
}

// MetricsSummary contains aggregated ticket statistics.
type MetricsSummary struct {
	Total      int `json:"total"`
	Open       int `json:"open"`
	InProgress int `json:"in_progress"`
	Resolved   int `json:"resolved"`
	Closed     int `json:"closed"`
}

// validTransitions defines allowable state machine transitions.
// E.g., An OPEN ticket can move to IN_PROGRESS or CLOSED (cancelled).
// A RESOLVED ticket can move to CLOSED or reopened to IN_PROGRESS.
var validTransitions = map[Status][]Status{
	StatusOpen:       {StatusInProgress, StatusClosed},
	StatusInProgress: {StatusResolved, StatusOpen, StatusClosed},
	StatusResolved:   {StatusClosed, StatusInProgress},
	StatusClosed:     {StatusOpen}, // Reopening a ticket
}

// CanTransitionTo verifies if a ticket can transition from current to target status.
func (t *Ticket) CanTransitionTo(target Status) bool {
	if t.Status == target {
		return true // No-op transition
	}

	allowed, exists := validTransitions[t.Status]
	if !exists {
		return false
	}

	for _, s := range allowed {
		if s == target {
			return true
		}
	}
	return false
}
