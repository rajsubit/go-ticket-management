// Package response provides standard JSON response utilities for HTTP handlers.
//
// LEARNING GO: HTTP Responses and JSON Encoding
// 1. `http.ResponseWriter`: An interface passed to HTTP handlers to write response headers, status codes, and body bytes.
// 2. Order matters! Call `w.Header().Set(...)` before `w.WriteHeader(...)`, and `w.WriteHeader(...)` before `w.Write(...)`.
// 3. `json.NewEncoder(w).Encode(data)`: Streams the JSON output directly to the network socket writer,
//    which is more memory-efficient than `json.Marshal(data)` which allocates an in-memory byte slice.
package response

import (
	"encoding/json"
	"net/http"
)

// Envelope wraps API responses in a consistent JSON structure.
type Envelope struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// JSON sends a JSON response with the given HTTP status code and data payload.
func JSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := Envelope{
		Success: statusCode >= 200 && statusCode < 300,
		Data:    data,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// Error sends a JSON error response with the given HTTP status code and error message.
func Error(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := Envelope{
		Success: false,
		Error:   message,
	}

	_ = json.NewEncoder(w).Encode(resp)
}
