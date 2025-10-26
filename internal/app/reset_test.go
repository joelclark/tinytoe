package app_test

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tinytoe/internal/app"
	"tinytoe/internal/config"
	"tinytoe/internal/ui"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestRunResetDropsSchemaAndReappliesMigrations(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("DATABASE_URL not set")
	}

	schema := fmt.Sprintf("tt_reset_apply_%d", time.Now().UnixNano())
	ctx := context.Background()

	adminDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open admin database: %v", err)
	}
	defer adminDB.Close()

	t.Cleanup(func() {
		_, _ = adminDB.ExecContext(context.Background(), fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quoteIdent(schema)))
	})

	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quoteIdent(schema))); err != nil {
		t.Fatalf("ensure schema absent: %v", err)
	}

	tempDir := t.TempDir()
	migrationsDir := filepath.Join(tempDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations dir: %v", err)
	}

	firstMigration := filepath.Join(migrationsDir, "20230101010101_create_widgets.sql")
	secondMigration := filepath.Join(migrationsDir, "20230101010202_seed_widgets.sql")

	firstSQL := strings.Join([]string{
		"CREATE TABLE widgets (id SERIAL PRIMARY KEY, name TEXT NOT NULL);",
		"INSERT INTO widgets (name) VALUES ('bootstrap');",
	}, "\n")
	if err := os.WriteFile(firstMigration, []byte(firstSQL+"\n"), 0o644); err != nil {
		t.Fatalf("write first migration: %v", err)
	}
	if err := os.WriteFile(secondMigration, []byte("INSERT INTO widgets (name) VALUES ('alpha'), ('beta');\n"), 0o644); err != nil {
		t.Fatalf("write second migration: %v", err)
	}

	cfg := config.Config{
		DatabaseURL:   dsn,
		MigrationsDir: migrationsDir,
		TargetSchema:  schema,
		Force:         true,
	}

	if err := app.RunUp(ctx, cfg, nil); err != nil {
		t.Fatalf("RunUp: %v", err)
	}

	schemaDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open schema database: %v", err)
	}
	defer schemaDB.Close()

	if _, err := schemaDB.ExecContext(ctx, fmt.Sprintf("SET search_path = %s", quoteIdent(schema))); err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	if _, err := schemaDB.ExecContext(ctx, "INSERT INTO widgets (name) VALUES ('manual');"); err != nil {
		t.Fatalf("insert manual row: %v", err)
	}

	var beforeCount int
	if err := schemaDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM widgets").Scan(&beforeCount); err != nil {
		t.Fatalf("count widgets before reset: %v", err)
	}
	if beforeCount != 4 {
		t.Fatalf("expected 4 widgets rows before reset, got %d", beforeCount)
	}

	var output bytes.Buffer
	if err := app.RunReset(ctx, cfg, nil, &output); err != nil {
		t.Fatalf("RunReset: %v", err)
	}

	if _, err := schemaDB.ExecContext(ctx, fmt.Sprintf("SET search_path = %s", quoteIdent(schema))); err != nil {
		t.Fatalf("reset search_path: %v", err)
	}

	var afterCount int
	if err := schemaDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM widgets").Scan(&afterCount); err != nil {
		t.Fatalf("count widgets after reset: %v", err)
	}
	if afterCount != 3 {
		t.Fatalf("expected 3 widgets rows after reset, got %d", afterCount)
	}

	var migrationsCount int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", qualify(schema, "tinytoe_migrations"))
	if err := schemaDB.QueryRowContext(ctx, query).Scan(&migrationsCount); err != nil {
		t.Fatalf("count migrations records: %v", err)
	}
	if migrationsCount != 2 {
		t.Fatalf("expected 2 recorded migrations after reset, got %d", migrationsCount)
	}

	result := output.String()
	resetLine := fmt.Sprintf("tinytoe reset %s schema %q dropped", ui.Arrow, schema)
	if !strings.Contains(result, resetLine) {
		t.Fatalf("expected schema drop output, got %q", result)
	}
	initLine := fmt.Sprintf("tinytoe init %s ready to migrate", ui.Arrow)
	if !strings.Contains(result, initLine) {
		t.Fatalf("expected init success output, got %q", result)
	}
	upLine := fmt.Sprintf("tinytoe up %s migrations applied successfully", ui.Arrow)
	if !strings.Contains(result, upLine) {
		t.Fatalf("expected up success output, got %q", result)
	}
	if !strings.Contains(result, "Applied: 2 migration(s)") {
		t.Fatalf("expected applied detail in output, got %q", result)
	}
}

func TestRunResetRequiresConfirmationWhenNotForced(t *testing.T) {
	tempDir := t.TempDir()
	migrationsDir := filepath.Join(tempDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations dir: %v", err)
	}

	cfg := config.Config{
		MigrationsDir:  migrationsDir,
		TargetSchema:   "public",
		NonInteractive: true,
	}

	err := app.RunReset(context.Background(), cfg, nil, nil)
	if err == nil {
		t.Fatalf("expected error when confirmation required in non-interactive mode")
	}
	if !strings.Contains(err.Error(), "rerun with --force") {
		t.Fatalf("expected non-interactive error, got %v", err)
	}
}

func TestRunResetAbortsWhenUserDeclines(t *testing.T) {
	tempDir := t.TempDir()
	migrationsDir := filepath.Join(tempDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations dir: %v", err)
	}

	cfg := config.Config{
		MigrationsDir: migrationsDir,
		TargetSchema:  "public",
	}

	stdin := strings.NewReader("n\n")
	var stdout bytes.Buffer

	err := app.RunReset(context.Background(), cfg, stdin, &stdout)
	if err == nil {
		t.Fatalf("expected error when user declines reset")
	}
	if !strings.Contains(err.Error(), "reset aborted by user") {
		t.Fatalf("expected abort error, got %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "drop the \"public\" schema") {
		t.Fatalf("expected warning in output, got %q", output)
	}
	if !strings.Contains(output, "Proceed with resetting schema \"public\"? [y/N]: ") {
		t.Fatalf("expected prompt in output, got %q", output)
	}
}

func qualify(schema, name string) string {
	return quoteIdent(schema) + "." + quoteIdent(name)
}
