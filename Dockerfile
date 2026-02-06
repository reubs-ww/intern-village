# Copyright (c) 2026 Intern Village. All rights reserved.
# SPDX-License-Identifier: Proprietary

# Frontend build stage
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Go build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files first for better layer caching
COPY orchestrator/go.mod orchestrator/go.sum ./
RUN go mod download

# Copy source code
COPY orchestrator/ ./

# Copy frontend dist from frontend builder for embedding
COPY --from=frontend-builder /app/frontend/dist ./internal/api/frontend/dist

# Build the binary with embedded frontend
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags embed_frontend -ldflags="-w -s" -o /orchestrator ./cmd/orchestrator

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    git \
    bash \
    curl \
    nodejs \
    npm

# Install Claude CLI (via npm)
# Note: Users should configure ANTHROPIC_API_KEY in their environment
RUN npm install -g @anthropic-ai/claude-code

# Create app user
RUN adduser -D -u 1000 orchestrator

# Create data directories
RUN mkdir -p /data/projects /data/prompts /data/logs && \
    chown -R orchestrator:orchestrator /data

# Copy binary from builder
COPY --from=builder /orchestrator /usr/local/bin/orchestrator

# Copy prompt templates
COPY --from=builder /app/prompts /app/prompts

# Copy migrations
COPY --from=builder /app/migrations /app/migrations

# Copy wait script
COPY orchestrator/scripts/wait-for-postgres.sh /usr/local/bin/wait-for-postgres.sh
RUN chmod +x /usr/local/bin/wait-for-postgres.sh

# Set working directory
WORKDIR /app

# Switch to non-root user
USER orchestrator

# Expose HTTP port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default command
ENTRYPOINT ["/usr/local/bin/orchestrator"]
