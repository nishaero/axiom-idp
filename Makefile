.PHONY: help build test run dev clean lint fmt security-scan coverage release verify verify-docker verify-minikube docker-build docker-push docker-run docker-compose-up docker-compose-down

# Variables
BINARY_NAME=axiom-server
MAIN_PATH=./cmd/axiom-server
OUTPUT_DIR=bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"
IMAGE_NAME ?= ghcr.io/axiom-idp/axiom
IMAGE_TAG ?= latest

# Colors for output
RESET=\033[0m
BOLD=\033[1m
GREEN=\033[32m
YELLOW=\033[33m
BLUE=\033[36m

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

## Build targets

build: ## Build the server binary
	@echo "$(BLUE)Building $(BINARY_NAME)...$(RESET)"
	@mkdir -p $(OUTPUT_DIR)
	@CGO_ENABLED=0 go build -trimpath $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)✓ Build complete: $(OUTPUT_DIR)/$(BINARY_NAME)$(RESET)"

build-fe: ## Build frontend
	@echo "$(BLUE)Building frontend...$(RESET)"
	@cd web && npm run build
	@echo "$(GREEN)✓ Frontend build complete$(RESET)"

build-all: build build-fe ## Build both backend and frontend

install-deps: ## Install Go and Node dependencies
	@echo "$(BLUE)Installing dependencies...$(RESET)"
	@go mod download
	@cd web && npm install
	@echo "$(GREEN)✓ Dependencies installed$(RESET)"

## Development targets

dev: ## Run in development mode with auto-reload
	@echo "$(BLUE)Starting development server...$(RESET)"
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	@air

run: build ## Build and run the server
	@echo "$(BLUE)Running $(BINARY_NAME)...$(RESET)"
	@$(OUTPUT_DIR)/$(BINARY_NAME)

run-fe: ## Run frontend dev server
	@cd web && npm run dev

watch: ## Run tests in watch mode
	@which reflex > /dev/null || go install github.com/cespare/reflex@latest
	@reflex -r '\.go$$' -s -- go test ./...

## Testing targets

test: ## Run all tests
	@echo "$(BLUE)Running tests...$(RESET)"
	@go test -v -race -timeout 10m ./...
	@echo "$(GREEN)✓ All tests passed$(RESET)"

test-backend: ## Run backend tests only
	@echo "$(BLUE)Running backend tests...$(RESET)"
	@go test -v -race -timeout 10s ./internal/...
	@echo "$(GREEN)✓ Backend tests passed$(RESET)"

test-fe: ## Run frontend tests
	@cd web && npm install --no-save jsdom @testing-library/user-event >/dev/null && npm test -- --run

test-integration: ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(RESET)"
	@go test -v -race -timeout 30s -tags=integration ./tests/...
	@echo "$(GREEN)✓ Integration tests passed$(RESET)"

coverage: ## Generate coverage report
	@echo "$(BLUE)Generating coverage report...$(RESET)"
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report: coverage.html$(RESET)"

coverage-ci: ## Generate coverage for CI (outputs to stdout)
	@go test -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -func=coverage.out

## Code quality targets

fmt: ## Format all code
	@echo "$(BLUE)Formatting code...$(RESET)"
	@go fmt ./...
	@goimports -w .
	@cd web && npm run format
	@echo "$(GREEN)✓ Code formatted$(RESET)"

lint: ## Run linters
	@echo "$(BLUE)Running linters...$(RESET)"
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@golangci-lint run
	@cd web && npm run lint
	@echo "$(GREEN)✓ Lint passed$(RESET)"

vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(RESET)"
	@go vet ./...
	@echo "$(GREEN)✓ Vet passed$(RESET)"

## Security targets

security-scan: ## Run security scans
	@echo "$(BLUE)Running security scans...$(RESET)"
	@which gosec > /dev/null || go install github.com/securego/gosec/v2/cmd/gosec@latest
	@gosec ./...
	@echo "$(BLUE)Container scanning with Trivy...$(RESET)"
	@which trivy > /dev/null || echo "Install Trivy: https://github.com/aquasecurity/trivy"
	@echo "$(GREEN)✓ Security scan complete$(RESET)"

check-secrets: ## Check for secrets in code
	@echo "$(BLUE)Checking for secrets...$(RESET)"
	@which detect-secrets > /dev/null || pip install detect-secrets
	@detect-secrets scan --all-files
	@echo "$(GREEN)✓ Secret scan complete$(RESET)"

## Deployment targets

docker-build: ## Build Docker image
	@echo "$(BLUE)Building Docker image...$(RESET)"
	@docker build --build-arg VERSION=$(VERSION) --build-arg BUILD_TIME=$(BUILD_TIME) -t $(IMAGE_NAME):$(IMAGE_TAG) .
	@echo "$(GREEN)✓ Docker image built: $(IMAGE_NAME):$(IMAGE_TAG)$(RESET)"

docker-push: docker-build ## Build and push Docker image
	@echo "$(BLUE)Pushing Docker image...$(RESET)"
	@docker push $(IMAGE_NAME):$(IMAGE_TAG)
	@echo "$(GREEN)✓ Docker image pushed$(RESET)"

docker-run: docker-build ## Build and run in Docker
	@echo "$(BLUE)Running in Docker...$(RESET)"
	@docker run -p 8080:8080 $(IMAGE_NAME):$(IMAGE_TAG)

docker-compose-up: ## Start with Docker Compose
	@echo "$(BLUE)Starting Docker Compose...$(RESET)"
	@docker compose up -d --build
	@echo "$(GREEN)✓ Services started$(RESET)"

docker-compose-down: ## Stop Docker Compose services
	@echo "$(BLUE)Stopping Docker Compose...$(RESET)"
	@docker compose down -v --remove-orphans
	@echo "$(GREEN)✓ Services stopped$(RESET)"

verify: ## Verify the running Docker/Compose deployment
	@./verify-app.sh

verify-docker: ## Run Docker Compose deployment validation
	@./scripts/validate-docker.sh

verify-minikube: ## Run Minikube deployment validation
	@./scripts/validate-minikube.sh

release: clean build security-scan test coverage lint ## Build production release
	@echo "$(BOLD)$(GREEN)✓ Release ready: $(OUTPUT_DIR)/$(BINARY_NAME)$(RESET)"

## Utility targets

clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(RESET)"
	@rm -rf $(OUTPUT_DIR)
	@rm -f coverage.* 
	@rm -rf web/dist
	@echo "$(GREEN)✓ Clean complete$(RESET)"

deps-update: ## Update dependencies
	@echo "$(BLUE)Updating dependencies...$(RESET)"
	@go get -u ./...
	@go mod tidy
	@cd web && npm update
	@echo "$(GREEN)✓ Dependencies updated$(RESET)"

docs: ## Generate API documentation
	@echo "$(BLUE)Generating documentation...$(RESET)"
	@which swag > /dev/null || go install github.com/swaggo/swag/cmd/swag@latest
	@swag init -g cmd/axiom-server/main.go
	@echo "$(GREEN)✓ Documentation generated$(RESET)"

version: ## Show version
	@echo "Version: $(VERSION)"

install: build build-fe ## Install binaries locally
	@echo "$(BLUE)Installing...$(RESET)"
	@mkdir -p ~/.local/bin
	@cp $(OUTPUT_DIR)/$(BINARY_NAME) ~/.local/bin/
	@echo "$(GREEN)✓ Installed to ~/.local/bin/$(BINARY_NAME)$(RESET)"

uninstall: ## Uninstall binaries
	@echo "$(BLUE)Uninstalling...$(RESET)"
	@rm -f ~/.local/bin/$(BINARY_NAME)
	@echo "$(GREEN)✓ Uninstalled$(RESET)"
