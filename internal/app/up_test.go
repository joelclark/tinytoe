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

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestRunUpAppliesPendingMigrations(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("DATABASE_URL not set")
	}

	schema := fmt.Sprintf("tt_up_apply_%d", time.Now().UnixNano())
	migrationDSN := dsn

	ctx := context.Background()
	adminDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open admin database: %v", err)
	}
	defer adminDB.Close()

	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA %s", quoteIdent(schema))); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminDB.ExecContext(context.Background(), fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quoteIdent(schema)))
	})

	tempDir := t.TempDir()
	migrationsDir := filepath.Join(tempDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations dir: %v", err)
	}

	firstMigration := filepath.Join(migrationsDir, "20230101010101_create_widgets.sql")
	secondMigration := filepath.Join(migrationsDir, "20230101010202_seed_widgets.sql")

	firstMigrationSQL := strings.Join([]string{
		"CREATE TABLE widgets (id SERIAL PRIMARY KEY, name TEXT NOT NULL);",
		"CREATE INDEX widgets_name_idx ON widgets (name);",
		"INSERT INTO widgets (name) VALUES ('bootstrap');",
	}, "\n")
	if err := os.WriteFile(firstMigration, []byte(firstMigrationSQL+"\n"), 0o644); err != nil {
		t.Fatalf("write first migration: %v", err)
	}
	if err := os.WriteFile(secondMigration, []byte("INSERT INTO widgets (name) VALUES ('alpha'), ('beta');\n"), 0o644); err != nil {
		t.Fatalf("write second migration: %v", err)
	}

	cfg := config.Config{
		DatabaseURL:   migrationDSN,
		MigrationsDir: migrationsDir,
		TargetSchema:  schema,
	}

	var out bytes.Buffer
	if err := app.RunUp(ctx, cfg, &out); err != nil {
		t.Fatalf("RunUp: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Applying 20230101010101_create_widgets.sql") {
		t.Fatalf("expected applying message for first migration, got %q", output)
	}
	if !strings.Contains(output, "migrations applied successfully") {
		t.Fatalf("expected success message, got %q", output)
	}
	if !strings.Contains(output, "Applied: 2 migration(s)") {
		t.Fatalf("expected count detail, got %q", output)
	}
	if !strings.Contains(output, "Latest: 20230101010202_seed_widgets.sql") {
		t.Fatalf("expected latest detail, got %q", output)
	}

	schemaDB, err := sql.Open("pgx", migrationDSN)
	if err != nil {
		t.Fatalf("open schema database: %v", err)
	}
	defer schemaDB.Close()
	if _, err := schemaDB.ExecContext(ctx, fmt.Sprintf("SET search_path = %s", quoteIdent(schema))); err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	var count int
	if err := schemaDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM widgets").Scan(&count); err != nil {
		t.Fatalf("query widgets count: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 widgets rows, got %d", count)
	}

	var indexExists bool
	indexQuery := `
SELECT EXISTS (
	SELECT 1
	FROM pg_indexes
	WHERE schemaname = $1 AND tablename = 'widgets' AND indexname = 'widgets_name_idx'
)`
	if err := schemaDB.QueryRowContext(ctx, indexQuery, schema).Scan(&indexExists); err != nil {
		t.Fatalf("query widgets index: %v", err)
	}
	if !indexExists {
		t.Fatalf("expected widgets_name_idx to exist")
	}

	out.Reset()
	if err := app.RunUp(ctx, cfg, &out); err != nil {
		t.Fatalf("RunUp second pass: %v", err)
	}
	if !strings.Contains(out.String(), "database already up to date") {
		t.Fatalf("expected already up-to-date message, got %q", out.String())
	}
}

func TestRunUpDetectsMissingMigrationFile(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("DATABASE_URL not set")
	}

	schema := fmt.Sprintf("tt_up_missing_%d", time.Now().UnixNano())
	migrationDSN := dsn

	ctx := context.Background()
	adminDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open admin database: %v", err)
	}
	defer adminDB.Close()

	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA %s", quoteIdent(schema))); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminDB.ExecContext(context.Background(), fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quoteIdent(schema)))
	})

	tempDir := t.TempDir()
	migrationsDir := filepath.Join(tempDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations dir: %v", err)
	}

	migrationFile := filepath.Join(migrationsDir, "20230101010101_create_table.sql")
	if err := os.WriteFile(migrationFile, []byte("CREATE TABLE demo (id INT PRIMARY KEY);\n"), 0o644); err != nil {
		t.Fatalf("write migration: %v", err)
	}

	cfg := config.Config{
		DatabaseURL:   migrationDSN,
		MigrationsDir: migrationsDir,
		TargetSchema:  schema,
	}

	if err := app.RunUp(ctx, cfg, nil); err != nil {
		t.Fatalf("initial RunUp: %v", err)
	}

	if err := os.Remove(migrationFile); err != nil {
		t.Fatalf("remove migration file: %v", err)
	}

	err = app.RunUp(ctx, cfg, nil)
	if err == nil {
		t.Fatalf("expected error when migration file is missing")
	}
	if !strings.Contains(err.Error(), "detected drift") || !strings.Contains(err.Error(), "toe reset") {
		t.Fatalf("expected drift error referencing toe reset, got %v", err)
	}
}

func TestRunUpDetectsRenamedMigrationFile(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("DATABASE_URL not set")
	}

	schema := fmt.Sprintf("tt_up_renamed_%d", time.Now().UnixNano())
	migrationDSN := dsn

	ctx := context.Background()
	adminDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open admin database: %v", err)
	}
	defer adminDB.Close()

	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA %s", quoteIdent(schema))); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminDB.ExecContext(context.Background(), fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quoteIdent(schema)))
	})

	tempDir := t.TempDir()
	migrationsDir := filepath.Join(tempDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations dir: %v", err)
	}

	original := filepath.Join(migrationsDir, "20230101010101_create_table.sql")
	if err := os.WriteFile(original, []byte("CREATE TABLE demo (id INT PRIMARY KEY);\n"), 0o644); err != nil {
		t.Fatalf("write migration: %v", err)
	}

	cfg := config.Config{
		DatabaseURL:   migrationDSN,
		MigrationsDir: migrationsDir,
		TargetSchema:  schema,
	}

	if err := app.RunUp(ctx, cfg, nil); err != nil {
		t.Fatalf("initial RunUp: %v", err)
	}

	renamed := filepath.Join(migrationsDir, "20230101010101_create_table_v2.sql")
	if err := os.Rename(original, renamed); err != nil {
		t.Fatalf("rename migration: %v", err)
	}

	err = app.RunUp(ctx, cfg, nil)
	if err == nil {
		t.Fatalf("expected error when migration file renamed")
	}
	if !strings.Contains(err.Error(), "detected drift") {
		t.Fatalf("expected drift error, got %v", err)
	}
}
