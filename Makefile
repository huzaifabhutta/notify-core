# Makefile for notify-core
# Supports both Simple and Multi-Tenant deployment modes

.PHONY: help build build-simple build-multitenant test clean run-simple run-multitenant docker-simple docker-multitenant

# Default target
help:
	@echo "notify-core - Multi-mode notification service"
	@echo ""
	@echo "🚀 Build Targets:"
	@echo "  build              - Build both simple and multi-tenant binaries"
	@echo "  build-simple       - Build simple mode (no database)"
	@echo "  build-multitenant  - Build multi-tenant mode (with database)"
	@echo ""
	@echo "🧪 Test Targets:"
	@echo "  test               - Run all tests"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo ""
	@echo "🏃 Run Targets (Development):"
	@echo "  run-simple         - Run simple mode server"
	@echo "  run-multitenant    - Run multi-tenant mode server"
	@echo ""
	@echo "🐳 Docker Targets:"
	@echo "  docker-simple      - Build Docker image for simple mode"
	@echo "  docker-multitenant - Build Docker image for multi-tenant mode"
	@echo "  docker-up-simple   - Start simple mode with docker-compose"
	@echo "  docker-up-multi    - Start multi-tenant mode with docker-compose"
	@echo ""
	@echo "🔧 Development Targets:"
	@echo "  deps               - Install dependencies"
	@echo "  lint               - Run linter (requires golangci-lint)"
	@echo "  fmt                - Format code"
	@echo "  vet                - Run go vet"
	@echo "  clean              - Remove built binaries"
	@echo ""
	@echo "⚡ Quick Start:"
	@echo "  quickstart-simple       - Show simple mode quick start guide"
	@echo "  quickstart-multitenant  - Show multi-tenant mode quick start guide"
	@echo ""
	@echo "📚 Documentation:"
	@echo "  See DEPLOYMENT_MODES.md for detailed deployment guide"

# Variables
APP_NAME=notify-core
BIN_DIR=bin

# Build targets
build: build-simple build-multitenant
	@echo "✅ Both binaries built successfully"
	@ls -lh $(BIN_DIR)/

build-simple:
	@echo "🔨 Building simple mode..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/notify-simple ./cmd/server/main.go
	@echo "✅ Simple mode binary: $(BIN_DIR)/notify-simple"

build-multitenant:
	@echo "🔨 Building multi-tenant mode..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/notify-multitenant ./cmd/server/main_tenant.go
	@echo "✅ Multi-tenant mode binary: $(BIN_DIR)/notify-multitenant"

# Dependencies
deps:
	@echo "📦 Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies installed"

# Test targets
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

test-coverage:
	@echo "🧪 Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# Clean target
clean:
	@echo "🧹 Cleaning..."
	@rm -rf $(BIN_DIR)/
	@rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

# Run targets (for development)
run-simple:
	@echo "🚀 Running simple mode..."
	@go run ./cmd/server/main.go

run-multitenant:
	@echo "🚀 Running multi-tenant mode..."
	@go run ./cmd/server/main_tenant.go

# Docker targets
docker-simple:
	@echo "🐳 Building Docker image for simple mode..."
	@docker build -t $(APP_NAME):simple -f Dockerfile.simple .
	@echo "✅ Image built: $(APP_NAME):simple"

docker-multitenant:
	@echo "🐳 Building Docker image for multi-tenant mode..."
	@docker build -t $(APP_NAME):multitenant -f Dockerfile.multitenant .
	@echo "✅ Image built: $(APP_NAME):multitenant"

docker-up-simple:
	@echo "🐳 Starting simple mode with Docker..."
	@docker-compose -f docker-compose.simple.yml up -d
	@echo "✅ Simple mode running at http://localhost:8080"

docker-up-multi:
	@echo "🐳 Starting multi-tenant mode with Docker..."
	@docker-compose -f docker-compose.yml up -d
	@echo "✅ Multi-tenant mode running at http://localhost:8080"

docker-down:
	@echo "🐳 Stopping Docker containers..."
	@docker-compose down
	@docker-compose -f docker-compose.simple.yml down 2>/dev/null || true
	@echo "✅ Containers stopped"

docker-logs:
	@docker-compose logs -f

# Lint target
lint:
	@echo "🔍 Running linter..."
	@golangci-lint run ./...

fmt:
	@echo "🎨 Formatting code..."
	@go fmt ./...

vet:
	@echo "🔍 Running go vet..."
	@go vet ./...

# Database migration targets (for multi-tenant mode)
migrate-up:
	@echo "📦 Running database migrations..."
	@go run ./cmd/migrate/main.go up
	@echo "✅ Migrations complete"

migrate-down:
	@echo "⚠️  Rolling back database migrations..."
	@go run ./cmd/migrate/main.go down
	@echo "✅ Rollback complete"

# Install targets
install-simple: build-simple
	@echo "📦 Installing simple mode to /usr/local/bin..."
	@sudo cp $(BIN_DIR)/notify-simple /usr/local/bin/
	@echo "✅ Installed: /usr/local/bin/notify-simple"

install-multitenant: build-multitenant
	@echo "📦 Installing multi-tenant mode to /usr/local/bin..."
	@sudo cp $(BIN_DIR)/notify-multitenant /usr/local/bin/
	@echo "✅ Installed: /usr/local/bin/notify-multitenant"

# Quick start helpers
quickstart-simple:
	@echo "⚡ Quick Start - Simple Mode"
	@echo ""
	@echo "1. Set environment variables:"
	@echo "   export API_KEYS=\"abc123:tenant1\""
	@echo "   export SMTP_HOST=\"smtp.gmail.com\""
	@echo "   export SMTP_PORT=587"
	@echo "   export SMTP_USER=\"your-email@gmail.com\""
	@echo "   export SMTP_PASS=\"your-app-password\""
	@echo ""
	@echo "2. Run:"
	@echo "   make run-simple"
	@echo ""
	@echo "3. Test:"
	@echo "   curl -X POST http://localhost:8080/send \\"
	@echo "     -H \"X-API-Key: abc123\" \\"
	@echo "     -H \"Content-Type: application/json\" \\"
	@echo "     -d '{\"channel\": \"email\", \"to\": \"user@example.com\", \"subject\": \"Test\", \"body\": \"Hello\"}'"
	@echo ""
	@echo "📚 See DEPLOYMENT_MODES.md for details"

quickstart-multitenant:
	@echo "⚡ Quick Start - Multi-Tenant Mode"
	@echo ""
	@echo "1. Start PostgreSQL:"
	@echo "   docker run --name notify-postgres \\"
	@echo "     -e POSTGRES_USER=notify \\"
	@echo "     -e POSTGRES_PASSWORD=securepassword \\"
	@echo "     -e POSTGRES_DB=notify \\"
	@echo "     -p 5432:5432 -d postgres:15"
	@echo ""
	@echo "2. Set environment variables:"
	@echo "   export DB_HOST=localhost"
	@echo "   export DB_USER=notify"
	@echo "   export DB_PASSWORD=securepassword"
	@echo "   export DB_NAME=notify"
	@echo "   export ENCRYPTION_KEY=\"$$(openssl rand -base64 32)\""
	@echo ""
	@echo "3. Run:"
	@echo "   make run-multitenant"
	@echo ""
	@echo "4. Create tenant:"
	@echo "   curl -X POST http://localhost:8080/v2/tenants \\"
	@echo "     -H \"Content-Type: application/json\" \\"
	@echo "     -d '{\"name\": \"acme\", \"smtp_host\": \"smtp.gmail.com\", ...}'"
	@echo ""
	@echo "📚 See DEPLOYMENT_MODES.md for details"

# All-in-one quality check
all: deps fmt vet test build
	@echo "✅ All checks passed!"

# Version info
version:
	@echo "$(APP_NAME) version information:"
	@echo "  Go version: $$(go version)"
	@echo "  Git commit: $$(git rev-parse --short HEAD 2>/dev/null || echo 'N/A')"
	@echo "  Git branch: $$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo 'N/A')"

.DEFAULT_GOAL := help
