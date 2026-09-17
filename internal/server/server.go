// Package server configures the HTTP server, middleware, routes, and graceful shutdown.
//
// LEARNING GO: Concurrency, Channels, and Graceful Shutdown
// 1. Goroutines (`go func()`):
//    - Lightweight concurrent threads managed by the Go runtime (costing only a few kilobytes of stack memory).
// 2. Channels (`make(chan os.Signal, 1)`):
//    - Go's signature communication primitive ("Do not communicate by sharing memory; instead, share memory by communicating").
//    - The channel blocks until the operating system sends an interrupt signal (SIGINT / SIGTERM).
// 3. `context.Context`:
//    - Used across API boundaries to signal cancellation, deadlines, and timeouts.
//    - `context.WithTimeout` allows active HTTP requests up to 10 seconds to finish before force-killing the server.
package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Config holds configuration parameters for the HTTP server.
type Config struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// DefaultConfig returns reasonable default server settings.
func DefaultConfig() Config {
	return Config{
		Port:            "8080",
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
}

// Server wraps the standard http.Server with graceful shutdown capability.
type Server struct {
	httpServer *http.Server
	config     Config
}

// NewServer initializes a new Server with the given configuration and handler.
func NewServer(cfg Config, handler http.Handler) *Server {
	return &Server{
		config: cfg,
		httpServer: &http.Server{
			Addr:         ":" + cfg.Port,
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
	}
}

// Start begins listening on the configured port and blocks until an interrupt signal is received,
// then cleanly drains connections with graceful shutdown.
func (s *Server) Start() error {
	shutdownError := make(chan error)

	// Listen for OS interrupt signals in background
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		sig := <-quit

		log.Printf("[SERVER] Received signal %s, initiating graceful shutdown...", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
		defer cancel()

		shutdownError <- s.httpServer.Shutdown(ctx)
	}()

	log.Printf("[SERVER] Ticket Management API running on http://localhost:%s", s.config.Port)
	log.Printf("[SERVER] Press Ctrl+C to stop")

	err := s.httpServer.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server failed: %w", err)
	}

	err = <-shutdownError
	if err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	log.Printf("[SERVER] Server gracefully stopped")
	return nil
}
