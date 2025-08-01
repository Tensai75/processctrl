.PHONY: build test lint clean fmt vet example deps help install-tools

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Build info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Directories
EXAMPLE_DIR=./example

all: deps fmt vet lint test build ## Run all checks and build

help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) verify

build: ## Build the package
	$(GOBUILD) -v ./...

example: ## Build and run the example
	cd $(EXAMPLE_DIR) && $(GOBUILD) -o example main.go

test: ## Run tests (excluding example directory)
	$(GOTEST) -v -race -coverprofile="coverage.out"

test-coverage: test ## Run tests with coverage report
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

bench: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem ./...

fmt: ## Format code
	$(GOFMT) -s -w .

vet: ## Run go vet
	$(GOVET) ./...

lint: install-tools ## Run golangci-lint
	golangci-lint run

lint-fix: install-tools ## Run golangci-lint with auto-fix
	golangci-lint run --fix

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -f coverage.out coverage.html
	rm -f $(EXAMPLE_DIR)/example
	rm -f $(EXAMPLE_DIR)/example.exe

install-tools: ## Install development tools
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

check: deps fmt vet lint test ## Run all checks

release-check: check ## Check if ready for release
	@echo "Checking if ready for release..."
	@git diff --exit-code > /dev/null || (echo "ERROR: Working directory is not clean" && exit 1)
	@echo "✓ Working directory is clean"
	@$(GOTEST) ./... > /dev/null || (echo "ERROR: Tests are failing" && exit 1)
	@echo "✓ All tests pass"
	@golangci-lint run > /dev/null || (echo "ERROR: Linting issues found" && exit 1)
	@echo "✓ No linting issues"
	@echo "Ready for release!"

# Cross-platform testing
test-windows: ## Test on Windows (requires Windows environment)
	GOOS=windows $(GOTEST) -v ./...

test-linux: ## Test on Linux (requires Linux environment)
	GOOS=linux $(GOTEST) -v ./...

test-darwin: ## Test on macOS (requires macOS environment)
	GOOS=darwin $(GOTEST) -v ./...

# Go module maintenance
mod-tidy: ## Tidy go modules
	$(GOMOD) tidy

mod-update: ## Update all dependencies
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

mod-graph: ## Show dependency graph
	$(GOMOD) graph

# Development utilities
watch-test: ## Watch for changes and run tests
	@which fswatch > /dev/null || (echo "fswatch not found. Install with: brew install fswatch" && exit 1)
	fswatch -o . | xargs -n1 -I{} make test

doc: ## Generate and serve documentation
	@echo "Starting documentation server at http://localhost:6060"
	@echo "Visit http://localhost:6060/pkg/github.com/tensai75/processctrl/"
	godoc -http=:6060
