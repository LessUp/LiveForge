# live-webrtc-go Makefile
# Comprehensive build, test, and deployment automation

.PHONY: help build test test-unit test-integration test-e2e test-performance test-security coverage lint fmt vet clean docker-build docker-run install-tools ci

# Default target
help:
	@echo "live-webrtc-go - Comprehensive Testing and Build System"
	@echo "====================================================="
	@echo ""
	@echo "Development Commands:"
	@echo "  make build          - Build the application"
	@echo "  make test           - Run all tests"
	@echo "  make test-unit      - Run unit tests only"
	@echo "  make test-integration - Run integration tests only"
	@echo "  make test-e2e       - Run end-to-end tests only"
	@echo "  make test-performance - Run performance tests"
	@echo "  make test-security  - Run security tests"
	@echo "  make coverage       - Generate test coverage report"
	@echo "  make lint           - Run linters"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Run go vet"
	@echo "  make clean          - Clean build artifacts"
	@echo ""
	@echo "Docker Commands:"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-run     - Run Docker container"
	@echo ""
	@echo "CI/CD Commands:"
	@echo "  make ci             - Run full CI pipeline"
	@echo "  make install-tools  - Install development tools"
	@echo ""

# Build configuration
BINARY_NAME=live-webrtc-go
DOCKER_IMAGE=live-webrtc-go:latest
GO_VERSION=1.21

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) ./cmd/server
	@echo "Build completed: bin/$(BINARY_NAME)"

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@go install github.com/axw/gocov/gocov@latest
	@go install github.com/AlekSi/gocov-xml@latest
	@go install github.com/matm/gocov-html@latest
	@echo "Development tools installed"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Code formatting completed"

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "Go vet completed"

# Run linters
lint: fmt vet
	@echo "Running golangci-lint..."
	@golangci-lint run ./...
	@echo "Linting completed"

# Security analysis
security:
	@echo "Running security analysis..."
	@gosec -fmt=json -out=reports/security-report.json ./...
	@echo "Security analysis completed - report: reports/security-report.json"

# Unit tests
test-unit:
	@echo "Running unit tests..."
	@go test -v -race -coverprofile=coverage-unit.out ./internal/...
	@echo "Unit tests completed"

# Integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -v -race -tags=integration -coverprofile=coverage-integration.out ./test/integration/...
	@echo "Integration tests completed"

# End-to-end tests
test-e2e:
	@echo "Running end-to-end tests..."
	@go test -v -tags=e2e -timeout=10m ./test/e2e/...
	@echo "End-to-end tests completed"

# Performance tests
test-performance:
	@echo "Running performance tests..."
	@go test -v -tags=performance -bench=. -benchmem ./test/performance/...
	@echo "Performance tests completed"

# Run all tests
test: test-unit test-integration
	@echo "All tests completed"

# Generate coverage report
coverage:
	@echo "Generating coverage report..."
	@mkdir -p reports
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o reports/coverage.html
	@gocov convert coverage.out | gocov-xml > reports/coverage.xml
	@echo "Coverage report generated:"
	@echo "  HTML: reports/coverage.html"
	@echo "  XML:  reports/coverage.xml"
	@go tool cover -func=coverage.out | tail -1

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf reports/
	@rm -f coverage*.out
	@rm -f *.test
	@echo "Cleanup completed"

# Docker commands
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) .
	@echo "Docker image built: $(DOCKER_IMAGE)"

docker-run:
	@echo "Running Docker container..."
	@docker run --rm -p 8080:8080 \
		-e RECORD_ENABLED=1 \
		-e RECORD_DIR=/records \
		-v "$$(pwd)/records:/records" \
		$(DOCKER_IMAGE)

# Full CI pipeline
ci: clean install-tools lint security test coverage
	@echo "CI pipeline completed successfully"

# Development server
dev:
	@echo "Starting development server..."
	@./scripts/start.sh

# Load testing
load-test:
	@echo "Running load tests..."
	@go run test/load/main.go -duration=30s -concurrent=10
	@echo "Load tests completed"

# Benchmark tests
benchmark:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...
	@echo "Benchmarks completed"