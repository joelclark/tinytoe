# Agent Guidance

## General Guidelines

- Write industry standard, idiomatic Go code
- Value simplicity above all else
- Review `FEATURES.md` for the current product requirements before implementing changes or reviewing code.
    - If in doubt, point out contradictions or inconsistencies

## Project Structure & Module Organization
- `cmd/tinytoe/main.go` is the CLI entrypoint; keep wiring minimal and delegate to internal packages.
- `internal/app` holds command workflows such as `init`; add new subcommands here.
- `internal/config` centralizes configuration loading and validation; extend it before touching consumers.
- `migrations` stores ordered SQL migration files; follow the existing naming scheme to preserve apply order.
- `scripts/` contains automation like bootstrap and integration helpers; prefer extending these over duplicating logic.
- `pinkytoe` wraps `go run ./cmd/tinytoe` for quicker local executions, which uses the `migrations` dir

## Test Guidelines

- All code should have test coverage where possible
- Do not mock up PostgreSQL, just connect to it, the .env file will be set up for you and `./scripts/run-integration-tests.sh` launches an ephemeral database for you to test against
- Use `./scripts/run-integration-tests.sh` to launch an ephemeral PostgreSQL container on port 5544 and run the Go test suite end-to-end.
- Place unit tests alongside source files (e.g., `internal/app/init_test.go`) and name them `Test<Scenario>`.
- Add regression coverage for new behaviors and document any skipped cases in test comments.
- Never remove a test just to get it to pass, only remove tests when the code being covered is being removed
- As an agent, you should run `./scripts/run-unit-tests.sh` as often as needed to ensure things are working
- IMPORTANT: As an agent, you must run `./scripts/run-integration-tests.sh` after code changes to ensure nothing is broken
    - we also call this the smoke test sometimes
    - you will have to ask permission to call this, this is fine

