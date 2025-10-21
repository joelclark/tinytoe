#!/usr/bin/env bash

set -euo pipefail

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required to run integration tests" >&2
  exit 1
fi

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

PG_IMAGE="${TINYTOE_PG_IMAGE:-postgres:16-alpine}"
PG_USER="${TINYTOE_PG_USER:-tt}"
PG_PASSWORD="${TINYTOE_PG_PASSWORD:-tt}"
PG_DATABASE="${TINYTOE_PG_DATABASE:-tt}"
# Listen on a non-default port to catch code paths that hard-code 5432.
PG_LISTEN_PORT="${TINYTOE_PG_LISTEN_PORT:-5544}"
CONTAINER_NAME="tinytoe-it-$$-$RANDOM"

# Tear down the container—even on error—to avoid leaving stray instances behind.
cleanup() {
  if docker container inspect "$CONTAINER_NAME" >/dev/null 2>&1; then
    docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT INT TERM

# Launch a temporary PostgreSQL container on a random high port.
docker run \
  --rm \
  --detach \
  --name "$CONTAINER_NAME" \
  -e POSTGRES_USER="$PG_USER" \
  -e POSTGRES_PASSWORD="$PG_PASSWORD" \
  -e POSTGRES_DB="$PG_DATABASE" \
  -p "127.0.0.1::${PG_LISTEN_PORT}" \
  "$PG_IMAGE" -c "port=${PG_LISTEN_PORT}" >/dev/null

PORT_MAPPING="$(docker port "$CONTAINER_NAME" "${PG_LISTEN_PORT}"/tcp)"
if [[ -z "$PORT_MAPPING" ]]; then
  echo "failed to discover exposed port for PostgreSQL container" >&2
  exit 1
fi

HOST="${PORT_MAPPING%:*}"
HOST="${HOST:-127.0.0.1}"
PORT="${PORT_MAPPING##*:}"

echo "waiting for PostgreSQL container ($CONTAINER_NAME) to become ready..."
for _ in $(seq 1 60); do
  if docker exec "$CONTAINER_NAME" pg_isready -U "$PG_USER" -d "$PG_DATABASE" -p "$PG_LISTEN_PORT" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if ! docker exec "$CONTAINER_NAME" pg_isready -U "$PG_USER" -d "$PG_DATABASE" -p "$PG_LISTEN_PORT" >/dev/null 2>&1; then
  echo "PostgreSQL did not become ready in time" >&2
  exit 1
fi

# Advertise connection details so downstream tooling (e.g. go test) uses this instance.
export DATABASE_URL="postgres://${PG_USER}:${PG_PASSWORD}@${HOST}:${PORT}/${PG_DATABASE}?sslmode=disable"
export PGHOST="$HOST"
export PGPORT="$PORT"
export PGUSER="$PG_USER"
export PGPASSWORD="$PG_PASSWORD"
export PGDATABASE="$PG_DATABASE"
export PGSSLMODE="disable"

echo "DATABASE_URL=${DATABASE_URL}"

# Run all Go tests with the freshly provisioned database.
if [[ -f go.mod ]]; then
  go test ./...
else
  echo "go.mod not found; skipping go test ./..." >&2
fi
