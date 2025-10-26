package app_test

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"tinytoe/internal/app"
	"tinytoe/internal/config"
	"tinytoe/internal/ui"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestRunDropAllDropsSchema(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("DATABASE_URL not set")
	}

	schema := fmt.Sprintf("tt_dropall_%d", time.Now().UnixNano())
	ctx := context.Background()

	adminDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open admin database: %v", err)
	}
	defer adminDB.Close()

	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA %s", quoteIdent(schema))); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s.demo (id INT)", quoteIdent(schema))); err != nil {
		t.Fatalf("create table in schema: %v", err)
	}

	cfg := config.Config{
		DatabaseURL:  dsn,
		TargetSchema: schema,
		Force:        true,
	}

	var out bytes.Buffer
	if err := app.RunDropAll(ctx, cfg, nil, &out); err != nil {
		t.Fatalf("RunDropAll: %v", err)
	}

	var count int
	if err := adminDB.QueryRowContext(ctx, `
SELECT COUNT(*) FROM information_schema.schemata
WHERE schema_name = $1
`, schema).Scan(&count); err != nil {
		t.Fatalf("query schema existence: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected schema %s to be dropped, but it still exists", schema)
	}

	result := out.String()
	expected := fmt.Sprintf("tinytoe dropall %s schema %q dropped", ui.Arrow, schema)
	if !strings.Contains(result, expected) {
		t.Fatalf("expected dropall success output, got %q", result)
	}
}

func TestRunDropAllRequiresConfirmationWhenNotForced(t *testing.T) {
	cfg := config.Config{
		TargetSchema:   "public",
		NonInteractive: true,
	}

	err := app.RunDropAll(context.Background(), cfg, nil, nil)
	if err == nil {
		t.Fatalf("expected error when confirmation required in non-interactive mode")
	}
	if !strings.Contains(err.Error(), "rerun with --force") {
		t.Fatalf("expected non-interactive error, got %v", err)
	}
}

func TestRunDropAllAbortsWhenUserDeclines(t *testing.T) {
	cfg := config.Config{
		TargetSchema: "public",
	}

	stdin := strings.NewReader("n\n")
	var stdout bytes.Buffer

	err := app.RunDropAll(context.Background(), cfg, stdin, &stdout)
	if err == nil {
		t.Fatalf("expected error when user declines dropall")
	}
	if !strings.Contains(err.Error(), "dropall aborted by user") {
		t.Fatalf("expected abort error, got %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "drop the \"public\" schema") {
		t.Fatalf("expected warning in output, got %q", output)
	}
	if !strings.Contains(output, "Proceed with dropping schema \"public\"? [y/N]: ") {
		t.Fatalf("expected prompt in output, got %q", output)
	}
}
