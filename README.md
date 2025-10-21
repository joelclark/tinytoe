# Tiny Toe Database Migration Tool

## Integration Tests

Run `./scripts/run-integration-tests.sh` to start a disposable PostgreSQL container and execute the Go test suite against it. The container listens on port 5544 to verify non-standard port handling, and the script cleans up automatically when it exits.
