// Package main is the entry point of the Ticket Management application.
//
// LEARNING GO: Package main, func main, and Dependency Injection
// 1. `package main`:
//    - Tells the Go compiler this package compiles into an executable binary (not a shared library).
// 2. `func main()`:
//    - The entrypoint where execution begins. It takes no arguments and returns nothing.
// 3. Composition & Explicit Wiring:
//    - Go avoids "magic" reflection-heavy frameworks or dependency injection containers.
//    - Notice how clean and obvious it is to trace: Repo -> Service -> Handler -> Router -> Server.
package main

import (
	"log"
	"os"

	"ticket-management/internal/server"
	"ticket-management/internal/ticket"
)

func main() {
	log.Println("==================================================")
	log.Println(" Starting Ticket Management Application (Go)     ")
	log.Println("==================================================")

	// 1. Read environment configuration (with defaults)
	cfg := server.DefaultConfig()
	if envPort := os.Getenv("PORT"); envPort != "" {
		cfg.Port = envPort
	}

	// 2. Initialize the Data Layer (In-Memory Repository)
	repo := ticket.NewInMemoryRepository()

	// 3. Seed initial demo data for immediate experimentation
	seedInitialData(repo)

	// 4. Initialize Business Service Layer
	ticketService := ticket.NewService(repo)

	// 5. Initialize HTTP Presentation Layer (Handler)
	ticketHandler := ticket.NewHandler(ticketService)

	// 6. Build HTTP Router with middleware
	router := server.NewRouter(ticketHandler)

	// 7. Initialize and start HTTP server with graceful shutdown
	srv := server.NewServer(cfg, router)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server stopped with error: %v", err)
	}
}

// seedInitialData populates the repository with a few sample tickets to explore immediately.
func seedInitialData(repo ticket.Repository) {
	samples := []*ticket.Ticket{
		{
			Title:       "Set up production database replica",
			Description: "Configure read-replica in AWS RDS for analytical reporting",
			Status:      ticket.StatusOpen,
			Priority:    ticket.PriorityHigh,
			Assignee:    "Sarah",
		},
		{
			Title:       "Fix navbar alignment on mobile screens",
			Description: "Hamburger icon overlaps user avatar on iOS Safari",
			Status:      ticket.StatusInProgress,
			Priority:    ticket.PriorityMedium,
			Assignee:    "Subit",
		},
		{
			Title:       "Update Go version to 1.27.1",
			Description: "Upgrade build pipelines to use the latest stable Go release",
			Status:      ticket.StatusResolved,
			Priority:    ticket.PriorityLow,
			Assignee:    "DevOps Team",
		},
	}

	for _, t := range samples {
		_ = repo.Create(t)
	}
	log.Printf("[SEED] Loaded %d sample tickets into in-memory store", len(samples))
}
