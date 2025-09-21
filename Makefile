# Makefile for Payment Processor

BINARY_NAME=payment-processor

# Build the application
build:
	go build -o $(BINARY_NAME) -v ./...

# Build for Linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)_unix -v ./...

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)_unix

# Run tests
test:
	go test -race -parallel 6 -count=1 -v -cover -coverprofile=unit_coverage.out ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Run the application
run:
	go build -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)

# Format code
fmt:
	gofmt -s -w .

# Run go vet
vet:
	go vet ./...

# Run linting tools
lint: fmt vet
	@echo "Running basic linting tools..."

# Download dependencies
deps:
	go mod download
	go mod tidy

# Generate mocks
generate-mocks:
	@echo "Generating mocks..."
	go generate ./...

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go get -u golang.org/x/tools/cmd/goimports
	go install go.uber.org/mock/mockgen@latest

# All-in-one development setup
dev-setup: install-tools deps
	@echo "Development environment setup complete"

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  build-linux  - Build for Linux"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  run          - Build and run the application"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  lint         - Run linting tools"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  generate-mocks- Generate mocks using gomock"
	@echo "  install-tools- Install development tools"
	@echo "  dev-setup    - Setup development environment"
	@echo "  help         - Show this help"

.PHONY: build build-linux clean test test-coverage run fmt vet lint deps generate-mocks install-tools dev-setup help
