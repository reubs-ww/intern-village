# Copyright (c) 2026 Intern Village. All rights reserved.
# SPDX-License-Identifier: Proprietary

.PHONY: help setup dev dev-api dev-frontend stop stop-db logs clean build test status restart migrate

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
help:
	@echo "Intern Village - Development Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Setup:"
	@echo "  setup          Install all dependencies (Go, npm)"
	@echo "  migrate        Run database migrations"
	@echo ""
	@echo "Development:"
	@echo "  dev            Start all services (PostgreSQL, API, Frontend)"
	@echo "  dev-api        Start PostgreSQL and API only (foreground)"
	@echo "  dev-frontend   Start frontend dev server only"
	@echo "  restart        Stop and restart all services"
	@echo "  stop           Stop all services (API, Frontend)"
	@echo "  stop-db        Stop PostgreSQL container"
	@echo "  status         Show status of all services"
	@echo "  logs           Tail orchestrator logs"
	@echo ""
	@echo "Build & Test:"
	@echo "  build          Build orchestrator and frontend"
	@echo "  test           Run all tests"
	@echo "  clean          Remove build artifacts and containers"

## setup: Install all dependencies
setup:
	@echo "Installing Go dependencies..."
	cd orchestrator && go mod download
	@echo "Installing npm dependencies..."
	cd frontend && npm install
	@echo "Setup complete!"

## dev: Start all services (PostgreSQL, API, Frontend)
dev: stop start-db build-api
	@echo "Starting orchestrator..."
	@cd orchestrator && \
		export $$(grep -v '^#' ../.env | xargs) && \
		export DATABASE_URL="postgres://intern:internpass@localhost:5432/intern_village?sslmode=disable" && \
		export DATA_DIR="$${DATA_DIR:-$(CURDIR)/data}" && \
		export PROMPTS_DIR="./prompts" && \
		./bin/orchestrator > /tmp/orchestrator.log 2>&1 &
	@sleep 2
	@# Verify orchestrator started
	@if curl -s http://localhost:8080/health > /dev/null 2>&1; then \
		echo "✓ Orchestrator is healthy"; \
	else \
		echo "✗ Orchestrator failed to start. Check: tail -f /tmp/orchestrator.log"; \
		exit 1; \
	fi
	@echo "Starting frontend dev server..."
	@cd frontend && npm run dev > /tmp/frontend.log 2>&1 &
	@sleep 3
	@# Verify frontend started
	@if curl -s http://localhost:5173 > /dev/null 2>&1; then \
		echo "✓ Frontend is running"; \
	else \
		echo "✗ Frontend failed to start. Check: tail -f /tmp/frontend.log"; \
	fi
	@echo ""
	@echo "============================================"
	@echo "  Intern Village is running!"
	@echo "============================================"
	@echo ""
	@echo "  Frontend:  http://localhost:5173"
	@echo "  API:       http://localhost:8080"
	@echo "  Database:  localhost:5432"
	@echo ""
	@echo "  Logs:"
	@echo "    make logs             - Tail orchestrator logs"
	@echo "    make logs-frontend    - Tail frontend logs"
	@echo ""
	@echo "  Commands:"
	@echo "    make stop             - Stop all services"
	@echo "    make restart          - Restart all services"
	@echo "    make status           - Check service status"
	@echo ""

## restart: Stop and restart all services
restart: stop dev

## dev-api: Start PostgreSQL and API only (foreground mode)
dev-api: start-db build-api
	@# Stop any existing orchestrator process first
	@-lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	@echo "Starting orchestrator (foreground mode, Ctrl+C to stop)..."
	@cd orchestrator && \
		export $$(grep -v '^#' ../.env | xargs) && \
		export DATABASE_URL="postgres://intern:internpass@localhost:5432/intern_village?sslmode=disable" && \
		export DATA_DIR="$${DATA_DIR:-$(CURDIR)/data}" && \
		export PROMPTS_DIR="./prompts" && \
		./bin/orchestrator

## dev-frontend: Start frontend dev server only
dev-frontend:
	@echo "Starting frontend dev server..."
	cd frontend && npm run dev

## start-db: Start PostgreSQL container
start-db:
	@if [ -z "$$(docker ps -q -f name=intern-village-postgres)" ]; then \
		if [ -z "$$(docker ps -aq -f name=intern-village-postgres)" ]; then \
			echo "Creating PostgreSQL container..."; \
			docker run -d \
				--name intern-village-postgres \
				-e POSTGRES_USER=intern \
				-e POSTGRES_PASSWORD=internpass \
				-e POSTGRES_DB=intern_village \
				-p 5432:5432 \
				postgres:16-alpine; \
		else \
			echo "Starting existing PostgreSQL container..."; \
			docker start intern-village-postgres; \
		fi; \
		echo "Waiting for PostgreSQL to be ready..."; \
		sleep 3; \
		until docker exec intern-village-postgres pg_isready -U intern -d intern_village > /dev/null 2>&1; do \
			sleep 1; \
		done; \
		echo "PostgreSQL is ready!"; \
	else \
		echo "PostgreSQL is already running."; \
	fi

## stop: Stop all services
stop:
	@echo "Stopping services..."
	@-lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	@-lsof -ti:5173 | xargs kill -9 2>/dev/null || true
	@echo "Services stopped."

## stop-db: Stop PostgreSQL container
stop-db:
	@echo "Stopping PostgreSQL container..."
	@-docker stop intern-village-postgres 2>/dev/null || true
	@echo "PostgreSQL stopped."

## status: Show status of all services
status:
	@echo "Service Status:"
	@echo ""
	@# Check PostgreSQL
	@if docker ps -q -f name=intern-village-postgres > /dev/null 2>&1 && [ -n "$$(docker ps -q -f name=intern-village-postgres)" ]; then \
		echo "  PostgreSQL:   ✓ Running (port 5432)"; \
	else \
		echo "  PostgreSQL:   ✗ Not running"; \
	fi
	@# Check Orchestrator
	@if curl -s http://localhost:8080/health > /dev/null 2>&1; then \
		echo "  Orchestrator: ✓ Running (port 8080)"; \
	else \
		echo "  Orchestrator: ✗ Not running"; \
	fi
	@# Check Frontend
	@if curl -s http://localhost:5173 > /dev/null 2>&1; then \
		echo "  Frontend:     ✓ Running (port 5173)"; \
	else \
		echo "  Frontend:     ✗ Not running"; \
	fi
	@echo ""

## logs: Tail orchestrator logs
logs:
	@tail -f /tmp/orchestrator.log

## logs-frontend: Tail frontend logs
logs-frontend:
	@tail -f /tmp/frontend.log

## build: Build orchestrator and frontend
build: build-api build-frontend

## build-api: Build orchestrator binary
build-api:
	@echo "Building orchestrator..."
	cd orchestrator && go build -o bin/orchestrator ./cmd/orchestrator

## build-frontend: Build frontend for production
build-frontend:
	@echo "Building frontend..."
	cd frontend && npm run build

## test: Run all tests
test:
	@echo "Running orchestrator tests..."
	cd orchestrator && go test -v ./...
	@echo "Running frontend tests..."
	cd frontend && npm run test:run

## clean: Remove build artifacts and containers
clean: stop stop-db
	@echo "Cleaning up..."
	@-docker rm intern-village-postgres 2>/dev/null || true
	@-rm -rf orchestrator/bin
	@-rm -rf orchestrator/data
	@-rm -rf frontend/dist
	@-rm -f /tmp/orchestrator.log /tmp/frontend.log
	@echo "Clean complete."

## migrate: Run database migrations
migrate:
	@echo "Running database migrations..."
	@if [ -z "$$(docker ps -q -f name=intern-village-postgres)" ]; then \
		echo "Error: PostgreSQL is not running. Run 'make start-db' first."; \
		exit 1; \
	fi
	@for f in orchestrator/migrations/*.sql; do \
		echo "Applying $$f..."; \
		cat "$$f" | sed -n '/+goose Up/,/+goose Down/p' | grep -v '+goose' | \
			docker exec -i intern-village-postgres psql -U intern -d intern_village > /dev/null 2>&1 || true; \
	done
	@echo "Migrations complete!"
	@echo ""
	@echo "Tables in database:"
	@docker exec intern-village-postgres psql -U intern -d intern_village -c "\dt" 2>/dev/null | grep -E "^\s+(public|users|projects|tasks|subtasks|agent_runs)" || echo "  (no tables found)"
