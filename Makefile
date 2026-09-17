# Makefile for Ticket Management Go Project

.PHONY: help run build test test-race vet fmt clean demo

help: ## Show available make commands
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

run: ## Run application directly with `go run`
	go run ./cmd/server

build: ## Compile production binary into bin/server
	@mkdir -p bin
	go build -o bin/server ./cmd/server
	@echo "Binary built successfully at bin/server"

test: ## Run all unit and integration tests with coverage
	go test -v -cover ./...

test-race: ## Run all tests with Go data race detector
	go test -v -race ./...

vet: ## Run Go static analysis tool
	go vet ./...

fmt: ## Format Go source code using standard gofmt rules
	go fmt ./...

clean: ## Remove compiled binaries and test artifacts
	rm -rf bin/ *.out

demo: ## Send sample curl requests to running server on localhost:8080
	@echo "1. Health Check:"
	curl -s http://localhost:8080/healthz | json_pp || curl -s http://localhost:8080/healthz
	@echo "\n\n2. List All Tickets:"
	curl -s http://localhost:8080/api/v1/tickets | json_pp || curl -s http://localhost:8080/api/v1/tickets
	@echo "\n\n3. Create New Ticket:"
	curl -s -X POST http://localhost:8080/api/v1/tickets \
		-H "Content-Type: application/json" \
		-d '{"title": "Implement Redis Cache", "description": "Cache frequent queries", "priority": "HIGH", "assignee": "Subit"}' | json_pp || true
	@echo "\n\n4. Get Metrics:"
	curl -s http://localhost:8080/api/v1/tickets/metrics | json_pp || curl -s http://localhost:8080/api/v1/tickets/metrics
