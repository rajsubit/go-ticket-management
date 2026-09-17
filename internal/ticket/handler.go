// Package ticket defines domain entities, business logic, repository contracts, and HTTP handlers.
//
// LEARNING GO: HTTP Handlers, JSON Decoding, and Path Variables
// 1. `http.HandlerFunc`: Any function with the signature `func(w http.ResponseWriter, r *http.Request)`
//    can act as an HTTP endpoint in Go.
// 2. Modern Routing in Go (1.22+):
//    - The standard `http.ServeMux` supports HTTP method prefixes (e.g., "GET /tickets/{id}")
//    - Path parameters are accessed cleanly using `r.PathValue("id")`.
// 3. Decoding Request Bodies:
//    - `json.NewDecoder(r.Body).Decode(&target)` streams and parses JSON payload directly into a Go struct.
// 4. Error Mapping:
//    - Notice how we use `errors.Is` to map internal domain errors to appropriate HTTP status codes (400, 404, 500).
package ticket

import (
	"encoding/json"
	"errors"
	"net/http"

	platformErrors "ticket-management/internal/platform/errors"
	"ticket-management/internal/platform/response"
)

// Handler handles HTTP requests for tickets.
type Handler struct {
	service *Service
}

// NewHandler constructs a new ticket Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// RegisterRoutes registers all ticket endpoints onto the provided http.ServeMux.
// In Go 1.22+, pattern strings can include HTTP methods (e.g. "POST /path") and path wildcards ("{id}").
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/tickets", h.CreateTicket)
	mux.HandleFunc("GET /api/v1/tickets", h.ListTickets)
	mux.HandleFunc("GET /api/v1/tickets/metrics", h.GetMetrics)
	mux.HandleFunc("GET /api/v1/tickets/{id}", h.GetTicketByID)
	mux.HandleFunc("PUT /api/v1/tickets/{id}", h.UpdateTicket)
	mux.HandleFunc("PATCH /api/v1/tickets/{id}/status", h.UpdateStatus)
	mux.HandleFunc("DELETE /api/v1/tickets/{id}", h.DeleteTicket)
}

// CreateTicket handles POST /api/v1/tickets
func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	created, err := h.service.Create(req)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, created)
}

// ListTickets handles GET /api/v1/tickets?status=OPEN&priority=HIGH
func (h *Handler) ListTickets(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := FilterOptions{
		Status:   Status(query.Get("status")),
		Priority: Priority(query.Get("priority")),
	}

	tickets, err := h.service.List(filter)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, tickets)
}

// GetTicketByID handles GET /api/v1/tickets/{id}
func (h *Handler) GetTicketByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Ticket ID is required")
		return
	}

	t, err := h.service.GetByID(id)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, t)
}

// UpdateTicket handles PUT /api/v1/tickets/{id}
func (h *Handler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Ticket ID is required")
		return
	}

	var req UpdateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	updated, err := h.service.Update(id, req)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, updated)
}

// UpdateStatus handles PATCH /api/v1/tickets/{id}/status
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Ticket ID is required")
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	updated, err := h.service.UpdateStatus(id, req.Status)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, updated)
}

// DeleteTicket handles DELETE /api/v1/tickets/{id}
func (h *Handler) DeleteTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Ticket ID is required")
		return
	}

	if err := h.service.Delete(id); err != nil {
		h.handleDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Ticket deleted successfully"})
}

// GetMetrics handles GET /api/v1/tickets/metrics
func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.service.GetMetrics()
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, metrics)
}

// handleDomainError maps internal domain errors to appropriate HTTP responses.
func (h *Handler) handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, platformErrors.ErrNotFound):
		response.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, platformErrors.ErrInvalidInput):
		response.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, platformErrors.ErrInvalidStatusTransition):
		response.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, platformErrors.ErrConflict):
		response.Error(w, http.StatusConflict, err.Error())
	default:
		response.Error(w, http.StatusInternalServerError, "An unexpected internal server error occurred")
	}
}
