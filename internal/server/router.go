// Package server configures the HTTP server, middleware, routes, and graceful shutdown.
//
// LEARNING GO: HTTP Middleware and Interface Embedding
// 1. Middleware Pattern:
//    - Middleware in Go is a higher-order function: `func(next http.Handler) http.Handler`.
//    - It intercepts incoming requests, performs pre-processing (logging, CORS), invokes `next.ServeHTTP(w, r)`,
//      and performs post-processing (duration tracking).
// 2. Struct Embedding (Composition):
//    - Notice `type statusRecorder struct { http.ResponseWriter; statusCode int }`.
//    - By embedding `http.ResponseWriter`, `statusRecorder` automatically inherits all methods of `http.ResponseWriter`.
//      We only override `WriteHeader(...)` to record the status code!
// 3. Panics and `recover()`:
//    - Go avoids exceptions, but unhandled bugs (like nil pointer dereference) trigger a `panic`.
//    - Panic recovery middleware uses `defer` and `recover()` to prevent a crash from taking down the whole server.
package server

import (
	"log"
	"net/http"
	"time"

	"ticket-management/internal/platform/response"
	"ticket-management/internal/ticket"
)

// statusRecorder wraps http.ResponseWriter to capture the HTTP status code sent to the client.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

// LoggerMiddleware logs every incoming HTTP request with its method, path, status, and processing duration.
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default if WriteHeader is not explicitly called
		}

		next.ServeHTTP(recorder, r)

		duration := time.Since(start)
		log.Printf("[HTTP] %s %s -> %d (%s)", r.Method, r.URL.Path, recorder.statusCode, duration)
	})
}

// RecoveryMiddleware catches any unexpected runtime panic and responds with a clean HTTP 500.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[PANIC RECOVERED] %v", rec)
				response.Error(w, http.StatusInternalServerError, "Internal server error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware enables Cross-Origin Resource Sharing for web browsers and frontend clients.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// NewRouter constructs the HTTP router, registers all routes, and wraps them in middleware.
func NewRouter(ticketHandler *ticket.Handler) http.Handler {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{
			"status": "healthy",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Register ticket domain routes
	ticketHandler.RegisterRoutes(mux)

	// Apply middleware stack (innermost executes last):
	// Request -> Recovery -> CORS -> Logger -> ServeMux
	var handler http.Handler = mux
	handler = LoggerMiddleware(handler)
	handler = CORSMiddleware(handler)
	handler = RecoveryMiddleware(handler)

	return handler
}
