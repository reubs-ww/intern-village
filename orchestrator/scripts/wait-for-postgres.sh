#!/bin/bash
# Copyright (c) 2026 Intern Village. All rights reserved.
# SPDX-License-Identifier: Proprietary

# Wait for PostgreSQL to be ready before starting the orchestrator.
# Usage: wait-for-postgres.sh [host] [port] [max_attempts]

set -e

HOST="${1:-postgres}"
PORT="${2:-5432}"
MAX_ATTEMPTS="${3:-30}"

echo "Waiting for PostgreSQL at $HOST:$PORT..."

attempt=1
while [ $attempt -le $MAX_ATTEMPTS ]; do
    if nc -z "$HOST" "$PORT" 2>/dev/null; then
        echo "PostgreSQL is ready!"
        exit 0
    fi

    echo "Attempt $attempt/$MAX_ATTEMPTS: PostgreSQL not ready, waiting..."
    sleep 1
    attempt=$((attempt + 1))
done

echo "ERROR: PostgreSQL did not become ready in time"
exit 1
