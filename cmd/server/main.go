// Package main is the entry point of the Ticket Management application.
//
// LEARNING GO: PostgreSQL Database Wiring, Environment Config, and Resource Teardown
// 1. Decoupled Architecture in Action:
//    - Look at how easily we swapped `InMemoryRepository` with `PostgresRepository`!
//    - `Service`, `Handler`, and `Router` require ZERO changes because they depend on the `Repository` interface.
// 2. Sensitive Credentials Loaded Safely:
//    - Notice we load configuration via `config.Load()`, which securely reads `.env` and environment variables.
//    - No passwords, usernames, or database names are committed into git or hardcoded in source files.
// 3. `defer db.Close()`:
//    - Ensures the PostgreSQL connection pool is cleanly closed and drained when `main()` exits.
// 4. Conditional Seeding:
//    - In-memory data disappeared on shutdown. With PostgreSQL, data persists across restarts!
//    - We check `repo.GetMetrics()` first and only seed if the table is completely empty.
package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"ticket-management/internal/platform/config"
	"ticket-management/internal/platform/database"
	"ticket-management/internal/server"
	"ticket-management/internal/ticket"
)

func main() {
	log.Println("==================================================")
	log.Println(" Starting Ticket Management Application (Go)     ")
	log.Println("==================================================")

	// 1. Load application and database configuration (.env -> environment variables)
	appCfg := config.Load()

	// 2. Read HTTP server configuration
	srvCfg := server.DefaultConfig()
	srvCfg.Port = appCfg.Port

	// 3. Configure and Initialize Data Storage Layer (PostgreSQL vs In-Memory)
	var repo ticket.Repository
	var db *sql.DB

	if appCfg.DBType == "memory" {
		log.Println("[DATABASE] Using In-Memory Repository (DB_TYPE=memory)")
		inMemRepo := ticket.NewInMemoryRepository()
		seedInitialData(inMemRepo)
		repo = inMemRepo
	} else {
		// Map DB settings from config
		dbCfg := database.Config{
			URL:             appCfg.DB.URL,
			Host:            appCfg.DB.Host,
			Port:            appCfg.DB.Port,
			User:            appCfg.DB.User,
			Password:        appCfg.DB.Password,
			DBName:          appCfg.DB.DBName,
			SSLMode:         appCfg.DB.SSLMode,
			MaxOpenConns:    appCfg.DB.MaxOpenConns,
			MaxIdleConns:    appCfg.DB.MaxIdleConns,
			ConnMaxLifetime: appCfg.DB.ConnMaxLifetime,
		}

		// Connect to PostgreSQL connection pool
		var err error
		db, err = database.Connect(dbCfg)
		if err != nil {
			log.Fatalf("[DATABASE ERROR] Could not connect to PostgreSQL: %v\n"+
				"Check that PostgreSQL is running locally and credentials match in .env.\n"+
				"Hint: To run in-memory without Postgres, start with: DB_TYPE=memory go run ./cmd/server", err)
		}
		defer func() {
			log.Println("[DATABASE] Closing PostgreSQL connection pool...")
			_ = db.Close()
		}()

		// Verify or auto-create required tables and indexes
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := database.EnsureSchema(ctx, db); err != nil {
			cancel()
			log.Fatalf("[DATABASE ERROR] Failed to initialize database schema: %v", err)
		}
		cancel()

		pgRepo := ticket.NewPostgresRepository(db)

		// Seed initial sample data only if table is currently empty
		metrics, err := pgRepo.GetMetrics()
		if err == nil && metrics.Total == 0 {
			seedInitialData(pgRepo)
		}

		repo = pgRepo
	}

	// 4. Initialize Business Service Layer (Accepts Repository interface)
	ticketService := ticket.NewService(repo)

	// 5. Initialize HTTP Presentation Layer (Handler)
	ticketHandler := ticket.NewHandler(ticketService)

	// 6. Build HTTP Router with middleware stack
	router := server.NewRouter(ticketHandler)

	// 7. Initialize and start HTTP server with graceful shutdown
	srv := server.NewServer(srvCfg, router)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server stopped with error: %v", err)
	}
}

// seedInitialData populates the repository with sample tickets.
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
	log.Printf("[SEED] Loaded %d initial sample tickets into repository", len(samples))
}
