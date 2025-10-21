#!/usr/bin/env bash

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

if ! command -v go >/dev/null 2>&1; then
  echo "Go is required to bootstrap Tiny Toe development dependencies." >&2
  exit 1
}

GOTESTSUM_VERSION="v1.13.0"
echo "Installing gotestsum ${GOTESTSUM_VERSION}..."
GO111MODULE=on go install "gotest.tools/gotestsum@${GOTESTSUM_VERSION}"

BIN_DIR="$(go env GOBIN)"
if [[ -z "$BIN_DIR" ]]; then
  BIN_DIR="$(go env GOPATH)/bin"
fi

echo "gotestsum installed. Ensure ${BIN_DIR} is on your PATH."

