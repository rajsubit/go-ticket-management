// Package errors defines domain-level sentinel errors and error helpers.
//
// LEARNING GO: Sentinel Errors vs Dynamic Errors
// In Go, errors are values (implementing the built-in `error` interface: `type error interface { Error() string }`).
// Sentinel errors (like ErrNotFound) are predefined static error values that allow callers to check
// what went wrong using `errors.Is(err, ErrNotFound)` rather than parsing error message strings.
package errors

import "errors"

var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrInvalidInput is returned when user input fails validation constraints.
	ErrInvalidInput = errors.New("invalid input parameters")

	// ErrInvalidStatusTransition is returned when moving a ticket to an incompatible state.
	ErrInvalidStatusTransition = errors.New("invalid status transition")

	// ErrConflict is returned when an operation conflicts with existing data.
	ErrConflict = errors.New("resource conflict")
)
