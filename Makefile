.PHONY: build run clean test test-coverage lint lint-fix docker-build docker-run proto swagger help

# Variables
APP_NAME = go-user-api
MAIN_PATH = cmd/server/main.go
BUILD_DIR = build
BUILD_PATH = $(BUILD_DIR)/$(APP_NAME)
PROTO_DIR = api/grpc/proto
PROTO_OUT_DIR = api/grpc/pb

# Go commands
GO = go
GORUN = $(GO) run
GOBUILD = $(GO) build
GOTEST = $(GO) test
GOGET = $(GO) get
GOMOD = $(GO) mod
GOFMT = $(GO) fmt
GOLINT = golangci-lint

# Docker commands
DOCKER = docker
DOCKER_COMPOSE = docker-compose

# Default target
.DEFAULT_GOAL := help

# Build the application
build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) -o $(BUILD_PATH) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_PATH)"

# Run the application
run: ## Run the application
	@echo "Running $(APP_NAME)..."
	@$(GORUN) $(MAIN_PATH) || true

# Clean build artifacts
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Run tests
test: ## Run tests
	@echo "Running tests..."
	@$(GOTEST) -v ./...

# Run tests with coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@$(GOTEST) -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out
	@rm coverage.out

# Install dependencies
deps: ## Install dependencies
	@echo "Installing dependencies..."
	@$(GOMOD) download
	@$(GOGET) github.com/swaggo/swag/cmd/swag
	@$(GOGET) google.golang.org/protobuf/cmd/protoc-gen-go
	@$(GOGET) google.golang.org/grpc/cmd/protoc-gen-go-grpc
	@echo "Dependencies installed"

# Lint code
lint: ## Lint code
	@echo "Linting code..."
	@$(GOLINT) run

# Fix linting issues
lint-fix: ## Fix linting issues
	@echo "Fixing linting issues..."
	@$(GOFMT) ./...
	@$(GOLINT) run --fix

# Generate protocol buffers
proto: ## Generate protocol buffers
	@echo "Generating protocol buffers..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto
	@echo "Protocol buffers generated"

# Generate swagger documentation
swagger: ## Generate swagger documentation
	@echo "Generating swagger documentation..."
	@swag init -g $(MAIN_PATH) -o docs
	@echo "Swagger documentation generated"

# Build docker image
docker-build: ## Build docker image
	@echo "Building docker image..."
	@$(DOCKER) build -t $(APP_NAME) .
	@echo "Docker image built"

# Run docker container
docker-run: ## Run docker container
	@echo "Running docker container..."
	@$(DOCKER) run -p 8080:8080 -p 50051:50051 --name $(APP_NAME) $(APP_NAME)

# Start all services with docker-compose
docker-up: ## Start all services with docker-compose
	@echo "Starting all services with docker-compose..."
	@$(DOCKER_COMPOSE) up

# Stop all services with docker-compose
docker-down: ## Stop all services with docker-compose
	@echo "Stopping all services with docker-compose..."
	@$(DOCKER_COMPOSE) down

# Show logs from docker-compose
docker-logs: ## Show logs from docker-compose
	@echo "Showing logs from docker-compose..."
	@$(DOCKER_COMPOSE) logs -f

# Initialize database (apply migrations)
db-init: ## Initialize database (apply migrations)
	@echo "Initializing database..."
	@$(GORUN) $(MAIN_PATH) db migrate
	@echo "Database initialized"

# Help
help: ## Show this help
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'