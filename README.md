# 🎟️ Ticket Management System in Go (Beginner's Masterclass)

Welcome to your first Go (Golang) project! This project is an idiomatic, production-ready **Ticket Management REST API** written from scratch using Go's standard library. It is designed specifically to teach you Go fundamentals, clean architecture, concurrency safety, and testing patterns.

---

## 📑 Table of Contents
1. [What is Go & Why Learn It?](#1-what-is-go--why-learn-it)
2. [Project Architecture & Directory Layout](#2-project-architecture--directory-layout)
3. [Core Go Concepts Explained By Example](#3-core-go-concepts-explained-by-example)
   - [Packages & Visibility](#a-packages--visibility)
   - [Types, Structs & Enums](#b-types-structs--enums)
   - [Pointers vs. Values](#c-pointers-vs-values)
   - [Interfaces & Dependency Injection](#d-interfaces--dependency-injection)
   - [Error Handling & Sentinel Errors](#e-error-handling--sentinel-errors)
   - [Concurrency & Mutexes](#f-concurrency--mutexes)
   - [Modern HTTP Routing & Middleware](#g-modern-http-routing--middleware)
4. [Testing in Go](#4-testing-in-go)
   - [Table-Driven Unit Tests](#table-driven-unit-tests)
   - [HTTP Testing with httptest](#http-testing-with-httptest)
   - [The Data Race Detector](#the-data-race-detector)
5. [Running the Application](#5-running-the-application)
6. [API Reference & Sample Requests](#6-api-reference--sample-requests)
7. [Next Steps in Your Go Journey](#7-next-steps-in-your-go-journey)

---

## 1. What is Go & Why Learn It?

Go (often called Golang) was designed at Google by Robert Griesemer, Rob Pike, and Ken Thompson. It was built to solve modern software challenges: multicore processors, networked systems, massive codebases, and fast delivery.

Key characteristics:
- **Fast Compilation**: Go compiles directly to a single, standalone machine binary (no JVM or interpreter needed).
- **Simplicity**: Go has only ~25 keywords. There is deliberately only one way to write loops (`for`), no class inheritance hierarchies, and no complex type metaprogramming.
- **Built-in Concurrency**: Concurrency is a first-class language feature via **goroutines** and **channels**.
- **Standard Library**: Go includes a world-class standard library (`net/http`, `encoding/json`, `sync`, `time`, `testing`) that lets you build production web services without heavy frameworks.

---

## 2. Project Architecture & Directory Layout

This project follows the **Standard Go Project Layout**:

```
ticket-management/
├── go.mod                     # Module definition & dependency tracker
├── Makefile                   # Handy developer shortcuts (make run, test, build, demo)
├── api.http                   # Executable HTTP requests for IDE REST clients
├── .gitignore                 # Excludes binaries and test artifacts
├── bin/                       # Output directory for compiled binaries
│   └── server
├── cmd/                       # Application entrypoints
│   └── server/
│       └── main.go            # `package main`: Wires dependencies and boots server
└── internal/                  # Private application code (enforced by Go compiler)
    ├── platform/              # Cross-cutting utilities shared across domain packages
    │   ├── errors/
    │   │   └── errors.go      # Domain sentinel error definitions
    │   └── response/
    │       └── response.go    # Consistent JSON API response helpers
    ├── ticket/                # The Ticket domain package
    │   ├── model.go           # Domain structs, custom enum types, validation
    │   ├── repository.go      # Storage interface & thread-safe in-memory store
    │   ├── service.go         # Core business logic & state machine transitions
    │   ├── service_test.go    # Table-driven unit tests
    │   ├── handler.go         # HTTP handlers (REST API endpoints)
    │   └── handler_test.go    # HTTP integration tests using net/http/httptest
    └── server/
        ├── router.go          # HTTP routes & middleware (Logger, Panic Recovery, CORS)
        └── server.go          # Server configuration & graceful OS signal shutdown
```

### Why `cmd/` and `internal/`?
- **`cmd/`**: Each subdirectory under `cmd/` represents an executable application (e.g. `cmd/server`, `cmd/cli`). It should only contain minimal code to wire up dependencies and start the program.
- **`internal/`**: A special directory recognized by the Go compiler. Code inside `internal/` **cannot** be imported by external Go projects. This provides strict encapsulation for your private business logic.

---

## 3. Core Go Concepts Explained By Example

### A. Packages & Visibility

In Go:
- Every file belongs to a `package` (declared at the very top: e.g., `package ticket`).
- All `.go` files in the same folder must share the same package name.
- **Visibility rule**:
  - Starts with **Uppercase** (e.g., `Ticket`, `CreateTicketRequest`, `NewService`) $\rightarrow$ **Exported** (public, accessible outside the package).
  - Starts with **lowercase** (e.g., `generateID()`, `validTransitions`) $\rightarrow$ **Unexported** (private to the package).

### B. Types, Structs & Enums

Go does not have a `class` or `enum` keyword. Instead:

1. **Custom Types**:
   ```go
   type Status string

   const (
       StatusOpen       Status = "OPEN"
       StatusInProgress Status = "IN_PROGRESS"
       StatusResolved   Status = "RESOLVED"
       StatusClosed     Status = "CLOSED"
   )
   ```
2. **Structs & Struct Tags**:
   ```go
   type Ticket struct {
       ID          string    `json:"id"`
       Title       string    `json:"title"`
       Status      Status    `json:"status"`
       Priority    Priority  `json:"priority"`
       CreatedAt   time.Time `json:"created_at"`
   }
   ```
   The backtick strings like `json:"id"` are **struct tags**. When Go's `encoding/json` serializes or deserializes data, it reads these tags to match JSON keys to struct fields.

### C. Pointers vs. Values

- **Value (`Ticket`)**: Passing a value creates an independent copy in memory. Modifying the copy does not affect the original.
- **Pointer (`*Ticket`)**: Passing a pointer (`&ticket`) passes the memory address. Modifying fields modifies the original object.
- **Why use pointers for updates?**
  In `UpdateTicketRequest`:
  ```go
  type UpdateTicketRequest struct {
      Title *string `json:"title,omitempty"`
  }
  ```
  In Go, a normal `string` defaults to `""` (the zero-value). A pointer `*string` can be `nil`. If it is `nil`, we know the client omitted the field in their update request. If it is non-nil, the client explicitly supplied a new value!

### D. Interfaces & Dependency Injection

Go interfaces are implemented **implicitly** (structural typing / duck typing):

```go
type Repository interface {
    Create(ticket *Ticket) error
    GetByID(id string) (Ticket, error)
    // ...
}
```

Any struct that implements these methods automatically satisfies `Repository`. There is **no** `implements` keyword.

In `internal/ticket/service.go`:
```go
type Service struct {
    repo Repository // Service depends on the INTERFACE, not a concrete database!
}
```
This means we can test `Service` with `InMemoryRepository` today, and later swap in a `PostgreSQLRepository` without changing a single line of business logic.

### E. Error Handling & Sentinel Errors

Go does not have `try/catch` exceptions. Instead, functions return errors as regular values:

```go
ticket, err := svc.GetByID(id)
if err != nil {
    // Handle the error immediately!
    return err
}
```

In `internal/platform/errors/errors.go`, we define **Sentinel Errors**:
```go
var ErrNotFound = errors.New("resource not found")
```
When wrapping errors with extra context, use `fmt.Errorf("%w: ...", ErrNotFound)`.
Callers can test for it anywhere in the call chain using `errors.Is(err, ErrNotFound)`.

### F. Concurrency & Mutexes

Go makes concurrent programming simple, but shared memory must be protected.
When an HTTP request arrives, Go launches a lightweight **goroutine** to handle it. If two users write to a map at the exact same moment without protection, the program will crash with `fatal error: concurrent map writes`.

To prevent this, our `InMemoryRepository` uses `sync.RWMutex`:
```go
func (r *InMemoryRepository) GetByID(id string) (Ticket, error) {
    r.mu.RLock()         // Multiple readers can read simultaneously
    defer r.mu.RUnlock() // Guaranteed to unlock when function exits
    t, exists := r.tickets[id]
    // ...
}

func (r *InMemoryRepository) Update(ticket Ticket) error {
    r.mu.Lock()          // Exclusive lock: only one writer at a time
    defer r.mu.Unlock()
    r.tickets[ticket.ID] = ticket
    return nil
}
```

### G. Modern HTTP Routing & Middleware

In Go 1.22+, the standard `http.ServeMux` supports HTTP methods and path wildcards:
```go
mux.HandleFunc("POST /api/v1/tickets", h.CreateTicket)
mux.HandleFunc("GET /api/v1/tickets/{id}", h.GetTicketByID)
```
Inside the handler, extract the path parameter with `r.PathValue("id")`.

Middleware wraps HTTP handlers in a clean pipeline:
```go
// Incoming Request -> Panic Recovery -> CORS -> Logger -> Handler -> Outgoing Response
```

---

## 4. Testing in Go

Go includes a built-in test runner via the `go test` command.

### Table-Driven Unit Tests
Look at [internal/ticket/service_test.go](internal/ticket/service_test.go):

```go
testCases := []struct {
    name        string
    req         ticket.CreateTicketRequest
    expectError bool
}{
    {name: "Valid ticket", req: ..., expectError: false},
    {name: "Empty title",  req: ..., expectError: true},
}

for _, tc := range testCases {
    t.Run(tc.name, func(t *testing.T) {
        _, err := svc.Create(tc.req)
        // assertions...
    })
}
```
This is the gold standard of testing in Go: one loop, multiple test cases, isolated subtest output.

### HTTP Testing with `httptest`
Look at [internal/ticket/handler_test.go](internal/ticket/handler_test.go):
- `httptest.NewRequest(...)`: Simulates an HTTP request in memory.
- `httptest.NewRecorder(...)`: Records the status code and response body without needing an active TCP port or network connection.

### The Data Race Detector
Go has an extraordinary compiler flag: `-race`.
```bash
go test -race ./...
```
It instruments memory access and checks whether two goroutines access the same memory location simultaneously without synchronization.

---

## 5. Running the Application

Use the included `Makefile` commands:

| Command | Action |
| :--- | :--- |
| `make run` | Run server directly with `go run ./cmd/server` |
| `make test` | Run all tests with code coverage |
| `make test-race` | Run all tests with the Go Race Detector enabled |
| `make build` | Compile a standalone binary to `bin/server` |
| `make vet` | Run Go's static analysis tool (`go vet`) |
| `make fmt` | Format all source files according to Go conventions |
| `make demo` | Send a series of test `curl` requests to the running server |
| `make clean` | Delete build artifacts and binaries |

---

## 6. API Reference & Sample Requests

The server starts by default on `http://localhost:8080`.

### 1. Health Check
```bash
curl http://localhost:8080/healthz
```

### 2. List Tickets (with optional query filters)
```bash
# All tickets
curl http://localhost:8080/api/v1/tickets

# Filter by status
curl "http://localhost:8080/api/v1/tickets?status=OPEN"

# Filter by priority
curl "http://localhost:8080/api/v1/tickets?priority=HIGH"
```

### 3. Create a Ticket
```bash
curl -X POST http://localhost:8080/api/v1/tickets \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Migrate auth to JWT tokens",
    "description": "Replace session cookies with JWT",
    "priority": "HIGH",
    "assignee": "Subit"
  }'
```

### 4. Update Ticket Status (State Machine)
```bash
# Valid transitions:
# OPEN -> IN_PROGRESS or CLOSED
# IN_PROGRESS -> RESOLVED, OPEN, or CLOSED
# RESOLVED -> CLOSED or IN_PROGRESS
# CLOSED -> OPEN (Reopening)

curl -X PATCH http://localhost:8080/api/v1/tickets/<TICKET_ID>/status \
  -H "Content-Type: application/json" \
  -d '{"status": "IN_PROGRESS"}'
```

### 5. Get Metrics Summary
```bash
curl http://localhost:8080/api/v1/tickets/metrics
```
Response:
```json
{
  "success": true,
  "data": {
    "total": 4,
    "open": 2,
    "in_progress": 1,
    "resolved": 1,
    "closed": 0
  }
}
```

> [!TIP]
> ### 🔌 Using the REST Client Extension
> We have pre-configured workspace recommendations in [.vscode/extensions.json](.vscode/extensions.json).
> 1. Open the Extensions sidebar (`Cmd + Shift + X` on Mac).
> 2. Search for **REST Client** (by *Huachao Mao*, ID: `humao.rest-client`) and click **Install**.
> 3. Open [api.http](api.http).
> 4. You will see a clickable **"Send Request"** button above every endpoint!
> 5. When you click "Send Request" on request `#5 Create a New Ticket`, it automatically extracts the new ticket ID (`@ticketId`) so that all subsequent requests (Get, Update, Status Transition, Delete) run seamlessly without manual copying!

---

## 7. Next Steps in Your Go Journey

Now that you understand project layout, interfaces, testing, and REST APIs in Go, here are great next exercises to level up:
1. **Persistent Database**: Implement the `Repository` interface using SQLite or PostgreSQL (`database/sql` or `pgx`).
2. **Authentication Middleware**: Add a JWT verification middleware in `internal/server/router.go`.
3. **Structured Logging**: Explore `log/slog` (Go's built-in structured logger added in Go 1.21).
4. **Docker Container**: Create a multi-stage `Dockerfile` to build a tiny ~15MB scratch container for `bin/server`.
