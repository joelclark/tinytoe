### Tiny Toe Database Migration Tool

#### 1. Core Philosophy
*   **Simplicity:** A command-line database migration tool, written in pure Go (no CGO), installs as a single self-contained binary.
*   **Portability:** Fully compatible with Linux and macOS.
*   **Convention over Configuration:** Uses clear, file-based conventions to minimize setup.
*   **No Down Migrations:** These are always a bad idea.
*   **Reliable:** Simple code with a comprehensive test suite.
*   **Easy to Get Started:** Installs with a simple curl command.
*   **Focused:** Only PostgreSQL is supported, more platforms to come in the future.

#### 2. Prerequisites and Constraints
*   Supports 64-bit Linux and macOS (ARM and x86).

#### 3. Configuration
*   Configuration is managed via environment variables.
*   A `.env` file will be loaded if found.
*   `DATABASE_URL`: The url of the target database.  (e.g. `postgres://bob:secret@1.2.3.4:5432/mydb?sslmode=verify-full`)
    *   The script will also use standard PostgreSQL environment variables (`PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`). 
*   `TINYTOE_MIGRATIONS_DIR`: Path to migrations directory. (Defaults to `./migrations`)
*   `TINYTOE_FORCE`: Set this to 1 or TRUE to avoid "Are you sure?" type messages.  


#### 4. State Management
*   A special table named `tinytoe_migrations` is used to track applied migrations.
*   The script will create this table automatically if it doesn't exist.
*   The table has a two columns
    * `version VARCHAR(255) PRIMARY KEY`
    * `filename VARCHAR(255)`
    * `applied_at TIMESTAMP`
*   The `version` stores the unique timestamp prefix of the applied migration file.

#### 5. Migration File Structure
*   Each migration is represented by a single `.sql` file that makes the desired changes.
*   The filename format enforces chronological order: `YYYYMMDDHHMMSS_description.sql`.

#### 6. Command Specification
*   **`toe init`**: 
    *   Creates the migrations directory (`./migrations` by default).
*   **`toe new <description>`**: Creates a new migration file using a UTC timestamp.
    *   Example: `toe new add_users_table` creates `20231027123000_add_users_table.sql`
*   **`toe up`**: Applies all pending migrations in chronological order.
    *   Note: Migrations are run within a transaction to ensure atomicity. If a script fails, the transaction is rolled back, and the tool exits.
*   **`toe reset`**: Completely resets the database and then applies all migrations.
    *   This is a destructive action and **must prompt the user for confirmation** (`Are you sure? [y/N]`).
    *   Can be run non-interactively with a `--force` flag.
*   **`toe status`**: Displays the status of all discoverable migrations.
    *   Checks and reports on config validity.
    *   Checks if init has been run.
        *   If init has been run, lists each migration file and its status: `[applied at 2025-06-05 12:34:00 UTC]` or `[pending]`.
*   **`toe help` or `toe --help`**: Displays a usage summary of all available commands.
