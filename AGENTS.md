# Agent Guidance

## General Guidelines

- Write industry standard, idiomatic Go code
- Value simplicity above all else

## Test Guidelines

- All code should have test coverage where possible
    - Do not mock up PostgreSQL, just connect to it, the .env file will be set up for you
- Use `./scripts/run-integration-tests.sh` to launch an ephemeral PostgreSQL container on port 5544 and run the Go test suite end-to-end.
