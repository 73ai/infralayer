.PHONY: dev build build-console check format clean help tail-log

# Default target
help:
	@echo "Available targets:"
	@echo "  dev       - Start development servers (console + backend + agent)"
	@echo "  build     - Build production binary with embedded console"
	@echo "  check     - Run linting and type checking"
	@echo "  format    - Format all code"
	@echo "  clean     - Clean build artifacts"
	@echo "  tail-log  - Show the last 100 lines of the log"

# Development mode - start both console, agent and backend
dev:
	@ENV=development ./development/shoreman.sh

# Production build
build: build-console
	@echo "Building production binary..."
	@mkdir -p bin
	@go build -o bin/infralayer ./services/backend/cmd/main.go
	@echo "Production binary created at bin/infralayer"

# Build console for production
build-console:
	@echo "Building console..."
	@cd services/console && npm install && npm run build

# Linting and type checking
check:
	@echo "Running Go checks..."
	-@cd services/backend && go vet ./... && go mod tidy
	@echo "Running console checks..."
	-@cd services/console && npm run lint
	@echo "Running website checks..."
	-@cd services/website && npm run lint
	@echo "Running agent checks..."
	-@cd services/agent && ruff check .
	@echo "Running CLI checks..."
	-@cd cli && ruff check src/
	@echo "Check complete."


# Format code
format:
	@echo "Formatting Go code..."
	@cd services/backend && go fmt ./...
	@echo "Formatting console code..."
	@cd services/console && npm run format
	@echo "Formatting website code..."
	@cd services/website && npm run format
	@echo "Formatting agent code..."
	@cd services/agent && ruff format .
	@echo "Formatting CLI code..."
	@cd cli && ruff format src/
	@echo "All code formatted."

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf services/console/dist/
	@rm -rf services/console/node_modules/
	@rm -rf services/agent/__pycache__/
	@rm -rf cli/__pycache__/
	@rm -rf cli/src/infralayer/__pycache__/
	@rm -rf cli/src/infralayer/**/__pycache__/
	@rm -rf cli/.pytest_cache/
	@rm -rf cli/infralayer.egg-info/
	@rm -rf cli/dist/
	@rm -rf cli/build/
	@echo "Clean complete."

# Display the last 100 lines of development log with ANSI codes stripped
tail-log:
	@tail -100 ./dev.log | perl -pe 's/\e\[[0-9;]*m(?:\e\[K)?//g'