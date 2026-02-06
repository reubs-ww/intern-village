# Copyright (c) 2026 Intern Village. All rights reserved.
# SPDX-License-Identifier: Proprietary

.PHONY: build test lint run clean fmt vet tidy migrate sqlc help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet
GOTIDY=$(GOCMD) mod tidy
BINARY_NAME=orchestrator
BINARY_PATH=bin/$(BINARY_NAME)

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

## build: Build the orchestrator binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) -o $(BINARY_PATH) ./cmd/orchestrator

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run golangci-lint
lint:
	@echo "Running linter..."
	golangci-lint run ./...

## fmt: Format Go source code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

## tidy: Tidy go modules
tidy:
	@echo "Tidying modules..."
	$(GOTIDY)

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

## run: Run the orchestrator locally
run: build
	@echo "Starting $(BINARY_NAME)..."
	./$(BINARY_PATH)

## sqlc: Generate sqlc code
sqlc:
	@echo "Generating sqlc code..."
	sqlc generate

## migrate-create: Create a new migration file (usage: make migrate-create NAME=migration_name)
migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=migration_name"; exit 1; fi
	@TIMESTAMP=$$(date +%Y%m%d%H%M%S); \
	touch migrations/$${TIMESTAMP}_$(NAME).up.sql; \
	touch migrations/$${TIMESTAMP}_$(NAME).down.sql; \
	echo "Created migrations/$${TIMESTAMP}_$(NAME).up.sql and .down.sql"

## dev: Run with hot reload (requires air)
dev:
	@echo "Starting development server with hot reload..."
	@air -c .air.toml || echo "Install air: go install github.com/cosmtrek/air@latest"

## check: Run fmt, vet, and lint
check: fmt vet lint

## all: Clean, tidy, check, and build
all: clean tidy check build
