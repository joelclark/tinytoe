#!/usr/bin/env bash

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

GOMODCACHE="${PROJECT_ROOT}/.gomod"
GOCACHE="${PROJECT_ROOT}/.gocache"

mkdir -p "${GOMODCACHE}" "${GOCACHE}"

echo "warming module cache under ${GOMODCACHE}"

GOMODCACHE="${GOMODCACHE}" \
GOCACHE="${GOCACHE}" \
GOPROXY="${GOPROXY:-https://proxy.golang.org,direct}" \
GOSUMDB="${GOSUMDB:-sum.golang.org}" \
go mod download

echo "module cache warmed successfully"
