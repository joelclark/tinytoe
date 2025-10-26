#!/usr/bin/env bash

set -euo pipefail

if ! command -v go >/dev/null 2>&1; then
  echo "go is required to run tests" >&2
  exit 1
fi

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOCACHE_DIR="${PROJECT_ROOT}/.gocache"
GOMODCACHE_DIR="${PROJECT_ROOT}/.gomod"

mkdir -p "${GOCACHE_DIR}" "${GOMODCACHE_DIR}"

# Pin caches inside the workspace so we never write outside the sandbox.
export GOCACHE="${GOCACHE_DIR}"
export GOMODCACHE="${GOMODCACHE_DIR}"

# Default to offline mode; callers can override explicitly.
export GOPROXY="${GOPROXY:-off}"
export GOSUMDB="${GOSUMDB:-off}"

cd "${PROJECT_ROOT}"

if [[ "$#" -eq 0 ]]; then
  set -- ./...
fi

if command -v gotestsum >/dev/null 2>&1; then
  gotestsum --format=short-verbose -- "$@"
else
  # Fallback to go run so we still benefit from rich reporting when gotestsum is absent.
  if go run gotest.tools/gotestsum@v1.13.0 --format=short-verbose -- "$@"; then
    :
  else
    echo "gotestsum unavailable and go run fallback failed; running go test directly" >&2
    go test "$@"
  fi
fi
