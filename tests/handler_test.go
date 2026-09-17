// Package tests contains all test suites organized in a single folder for easy access.
package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ticket-management/internal/platform/response"
	"ticket-management/internal/ticket"
)

// TestHTTPCreateAndGetTicket tests the HTTP API lifecycle: POST create then GET by ID.
func TestHTTPCreateAndGetTicket(t *testing.T) {
	repo := ticket.NewInMemoryRepository()
	svc := ticket.NewService(repo)
	handler := ticket.NewHandler(svc)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// 1. Test POST /api/v1/tickets
	createBody := `{"title": "Implement JWT Authentication", "priority": "HIGH", "assignee": "Bob"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected HTTP 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var createdEnvelope struct {
		Success bool          `json:"success"`
		Data    ticket.Ticket `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&createdEnvelope); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	created := createdEnvelope.Data
	if created.ID == "" {
		t.Fatalf("expected ticket to have an ID")
	}
	if created.Status != ticket.StatusOpen {
		t.Errorf("expected status %s, got %s", ticket.StatusOpen, created.Status)
	}

	// 2. Test GET /api/v1/tickets/{id}
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+created.ID, nil)
	getRec := httptest.NewRecorder()

	mux.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got %d: %s", getRec.Code, getRec.Body.String())
	}

	// 3. Test POST with invalid payload
	invalidBody := `{"title": "x", "priority": "INVALID"}`
	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/tickets", bytes.NewBufferString(invalidBody))
	badRec := httptest.NewRecorder()

	mux.ServeHTTP(badRec, badReq)

	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("expected HTTP 400 Bad Request, got %d", badRec.Code)
	}

	var errorEnvelope response.Envelope
	_ = json.NewDecoder(badRec.Body).Decode(&errorEnvelope)
	if errorEnvelope.Success {
		t.Errorf("expected success to be false on error response")
	}
}
