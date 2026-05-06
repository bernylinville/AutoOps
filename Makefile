# AutoOps Development Makefile
# Quick commands for common development tasks

.PHONY: bootstrap dev-check fmt lint test presubmit dev build clean help

# Default target
.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "AutoOps Development Commands"
	@echo "============================"
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

bootstrap: ## Check development environment prerequisites
	@./scripts/bootstrap

dev-check: ## Start services and run smoke tests
	@echo "Starting required services..."
	@cd docker && docker compose up -d postgres valkey prometheus pushgateway
	@echo "Waiting for services to be ready..."
	@sleep 3
	@echo "Running smoke tests..."
	@./scripts/smoke-test

fmt: ## Format Go code
	@echo "Formatting Go code..."
	@cd api && go fmt ./...

lint: ## Run linters (Go + Vue)
	@echo "Running Go linter..."
	@cd api && golangci-lint run --new ./...
	@echo "Running Vue linter..."
	@cd web && npm run lint

presubmit: ## Run local checks equivalent to CI
	@./scripts/presubmit

test: ## Run tests (Go + Vue)
	@echo "Running Go tests..."
	@cd api && go test ./... -v -count=1
	@echo "Running Vue tests..."
	@cd web && npm run lint && npm run build

dev: ## Start development environment (services + backend + frontend)
	@echo "Starting background services..."
	@cd docker && docker compose up -d postgres valkey prometheus pushgateway
	@echo "Services started. Now run:"
	@echo "  Terminal 1: cd api && go run ."
	@echo "  Terminal 2: cd web && npm run dev"

build: ## Build production artifacts
	@echo "Building backend..."
	@cd api && go build -o devops-api .
	@echo "Building frontend..."
	@cd web && npm run build

clean: ## Clean build artifacts and Docker volumes
	@echo "Cleaning build artifacts..."
	@rm -f api/devops-api
	@rm -rf web/dist
	@echo "Stopping services..."
	@cd docker && docker compose down
