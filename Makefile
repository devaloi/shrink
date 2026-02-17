.PHONY: build run test lint clean fmt vet

# Build the server binary
build:
	go build -o bin/shrink ./cmd/server

# Run the server
run:
	go run ./cmd/server

# Run all tests
test:
	go test -v -race ./...

# Run tests with coverage
cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	gofmt -s -w .
	goimports -w .

# Vet code
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f shrink.db

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run all checks (used before commit)
check: fmt vet lint test
	@echo "All checks passed!"
