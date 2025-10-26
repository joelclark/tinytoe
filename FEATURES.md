### Tiny Toe Database Migration Tool

#### 1. Core Philosophy
*   **Simplicity:** A command-line database migration tool, written in pure Go (no CGO), installs as a single self-contained binary.
*   **Portability:** Fully compatible with Linux and macOS.
*   **Convention over Configuration:** Uses clear, file-based conventions to minimize setup.
*   **No Down Migrations:** These are always a bad idea.
*   **Reliable:** Simple code with a comprehensive test suite.
*   **Easy to Get Started:** Installs with a simple curl command.
*   **Focused:** Only PostgreSQL is supported, more platforms to come in the future.
*   **Delightful:** Unless disabled, the output of the commands is pleasant and tasteful, using color where appropriate.

#### 2. Prerequisites and Constraints
*   Supports 64-bit Linux and macOS (ARM and x86).

#### 3. Configuration
*   Configuration is managed via environment variables.
*   A `.env` file will be loaded if found.
*   `DATABASE_URL`: The url of the target database.  (e.g. `postgres://bob:secret@1.2.3.4:5432/mydb?sslmode=verify-full`)
    *   The script will also use standard PostgreSQL environment variables (`PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`).
*   `TINYTOE_TARGET_SCHEMA`: Explicit schema Tiny Toe manages. Defaults to `public` when unset. Values matching system schemas (e.g. `pg_catalog`, `pg_temp`) or empty strings are rejected.
*   `TINYTOE_MIGRATIONS_DIR`: Path to migrations directory. (Defaults to `./migrations`).
*   `TINYTOE_FORCE`: Set this to `1` or `TRUE` to bypass interactive confirmation prompts.
*   `TINYTOE_NON_INTERACTIVE`: When set to `1` or `TRUE`, commands that require confirmation exit with an error instead of prompting.
*   `TINYTOE_NO_COLOR`: Set to disable colorized output globally (mirrors the `--no-color` CLI flag).


#### 4. State Management
*   A special table named `tinytoe_migrations` is used to track applied migrations.
*   The tool will create this table automatically if it doesn't exist before the first migration runs.
*   Table definition (managed solely by Tiny Toe):
    *   `version VARCHAR(255) PRIMARY KEY` – the UTC timestamp prefix from the migration filename.
    *   `filename VARCHAR(1024) NOT NULL` – full basename of the migration file as it was applied.
    *   `applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()` – populated automatically at apply time (UTC).
*   Applied migrations are immutable. If a previously applied migration file is modified or removed, Tiny Toe will surface an error instructing the user to perform a `toe reset` to reconcile the database state.
*   The combination of `version` and `filename` is authoritative; renaming an applied file without a reset is treated as drift and blocks further execution.

#### 5. Migration File Structure
*   Each migration is represented by a single `.sql` file that makes the desired changes.
*   The filename format enforces chronological order: `YYYYMMDDHHMMSS_description.sql`.
*   Generated migrations start with a SQL comment header inserted by `toe new`:
    ```sql
    -- Tiny Toe Migration
    -- Version: <UTC timestamp prefix>
    -- Filename: <full filename>
    -- Created At (UTC): <ISO8601 timestamp>
    -- Created By: <os/user info if available>
    ```
    followed by a blank line ready for SQL statements. The header captures the on-disk metadata for traceability.
*   Migration bodies are authored by hand. Tiny Toe wraps each migration file in a single database transaction so the file succeeds or fails atomically; authors should generally provide plain SQL statements without additional `BEGIN/COMMIT` wrappers.  Each connection issues `SET search_path = <TINYTOE_TARGET_SCHEMA>` before executing statements so objects land in the managed schema. Tiny Toe migrations run inside pgx’s simple protocol.

#### 6. Command Specification
*   **`toe init`**
    *   Ensures the migrations directory (`./migrations` by default) exists, creating it if necessary.
    *   Validates connectivity to the target database and creates the `tinytoe_migrations` table when missing.
    *   Prints an idempotent success message; re-running is safe.
*   **`toe new <description>`**
    *   Creates a new migration file using a UTC timestamp prefix and the provided slugified description.
    *   Writes the standard comment header plus an empty line for SQL content.
    *   Fails if the migrations directory does not exist (user must run `toe init` first) unless `--force` is provided to create it implicitly.
    *   Example: `toe new add_users_table` creates `20231027123000_add_users_table.sql`.
*   **`toe up`**
    *   Discovers migrations in timestamp order, compares against `tinytoe_migrations`, and applies only pending files.
    *   Each migration runs inside its own database transaction; failure rolls back that migration and stops processing.
    *   Logs progress to stdout using friendly, colorized output when writing to an interactive TTY. A `--no-color` (and CI-driven `TINYTOE_NO_COLOR`) override forces plain text for pipelines.
    *   Exits with non-zero status on the first failure and, on success, prints the count of newly applied migrations.
    *   Detects drift (missing or changed applied migrations) and aborts with actionable messaging directing the user to `toe reset`.
*   **`toe reset`**
    *   Confirms destructive intent interactively unless `TINYTOE_FORCE` is set or a `--force` flag is passed.
    *   Drops and recreates the schema specified by `TINYTOE_TARGET_SCHEMA` (defaulting to `public`), effectively wiping user data, then recreates the `tinytoe_migrations` table and reapplies all migrations via the `toe up` pipeline.
    *   Intended as the only supported way to change an applied migration.
*   **`toe status`**
    *   Validates configuration and database connectivity.
    *   Produces a tabular or column-aligned list of every migration file with state `applied <timestamp>` or `pending` and highlights drift scenarios.
    *   Exits with code `0` when the database matches the migration directory, `1` when pending migrations exist, and `2` when drift or failed checks are encountered.
*   **`toe help` / `toe --help`**
    *   Displays a usage summary of all available commands and global options.

#### 7. CLI Configuration and Overrides
*   Environment variables are the primary configuration surface; CLI flags exist only for compelling overrides like `--no-color` and `--force`.
*   Precedence (lowest to highest): defaults → `.env` → environment variables → explicit CLI flags.
*   Fail fast with clear messaging when required configuration is missing. Help output documents all supported overrides.

#### 8. Testing Strategy
*   Favor black-box integration tests that exercise the compiled binary against a real PostgreSQL instance.
*   Tests spin up isolated databases or schemas per run to avoid state bleed.
*   Core scenarios to cover:
    *   Fresh initialization and repeated idempotent runs.
    *   Applying single and multiple migrations, including success and failure rollback behavior.
    *   Drift detection (missing, renamed, or modified applied migrations) halting with actionable messaging.
    *   `toe reset` wiping the configured target schema and replaying migrations from scratch.
    *   CLI configuration precedence (env overrides, `.env` loading, and flag handling).
*   Where pure unit tests provide value (e.g., parsing filenames), include them, but prioritize end-to-end coverage.

#### 9. Documentation Roadmap
*   Comprehensive README, quick-start guide, and CONTRIBUTING instructions will be authored once the feature set stabilizes.
