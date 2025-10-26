package app_test

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/url"
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

func TestRunInitCreatesDirAndTable(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("DATABASE_URL not set")
	}

	// Use a unique schema per test to avoid state bleed.
	schema := fmt.Sprintf("tt_init_%d", time.Now().UnixNano())
	initDSN, err := withSearchPath(dsn, schema)
	if err != nil {
		t.Fatalf("prepare DSN: %v", err)
	}

	ctx := context.Background()
	adminDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer adminDB.Close()

	t.Cleanup(func() {
		_, _ = adminDB.ExecContext(context.Background(), fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quoteIdent(schema)))
	})
	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quoteIdent(schema))); err != nil {
		t.Fatalf("ensure schema absent: %v", err)
	}

	tempDir := t.TempDir()
	cfg := config.Config{
		DatabaseURL:   initDSN,
		MigrationsDir: filepath.Join(tempDir, "migrations"),
		TargetSchema:  schema,
	}

	var out bytes.Buffer
	if err := app.RunInit(ctx, cfg, &out); err != nil {
		t.Fatalf("first RunInit: %v", err)
	}

	if err := app.RunInit(ctx, cfg, &out); err != nil {
		t.Fatalf("second RunInit: %v", err)
	}

	output := out.String()
	expectedLine := fmt.Sprintf("tinytoe init %s ready to migrate", ui.Arrow)
	if count := strings.Count(output, expectedLine); count != 2 {
		t.Fatalf("expected two success messages, got %d; output: %q", count, output)
	}
	if !strings.Contains(output, "Migrations Directory: "+cfg.MigrationsDir) {
		t.Fatalf("expected migrations directory in output, got %q", output)
	}
	if !strings.Contains(output, "Target Schema: "+schema) {
		t.Fatalf("expected target schema in output, got %q", output)
	}

	if stat, err := os.Stat(cfg.MigrationsDir); err != nil {
		t.Fatalf("stat migrations dir: %v", err)
	} else if !stat.IsDir() {
		t.Fatalf("expected %s to be a directory", cfg.MigrationsDir)
	}

	schemaDB, err := sql.Open("pgx", initDSN)
	if err != nil {
		t.Fatalf("open schema database: %v", err)
	}
	defer schemaDB.Close()

	var count int
	err = adminDB.QueryRowContext(ctx, `
SELECT COUNT(*) FROM information_schema.schemata
WHERE schema_name = $1
`, schema).Scan(&count)
	if err != nil {
		t.Fatalf("query schema existence: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected schema %s to exist after init", schema)
	}

	count = 0
	err = schemaDB.QueryRowContext(ctx, `
SELECT COUNT(*) FROM information_schema.tables
WHERE table_schema = $1 AND table_name = 'tinytoe_migrations'
`, schema).Scan(&count)
	if err != nil {
		t.Fatalf("query migrations table: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected tinytoe_migrations table in schema %s", schema)
	}
}

func withSearchPath(dsn, schema string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	if u.Scheme == "" {
		return "", fmt.Errorf("DATABASE_URL must be a PostgreSQL connection URI")
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func quoteIdent(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}
