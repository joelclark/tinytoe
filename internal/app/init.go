package app

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"time"

	"tinytoe/internal/config"
	"tinytoe/internal/ui"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const migrationsTableDDL = `
CREATE TABLE IF NOT EXISTS tinytoe_migrations (
	version VARCHAR(255) PRIMARY KEY,
	filename VARCHAR(1024) NOT NULL,
	applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
)`

// RunInit performs the work for `tinytoe init`.
func RunInit(ctx context.Context, cfg config.Config, stdout io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := ensureMigrationsDir(cfg.MigrationsDir); err != nil {
		return fmt.Errorf("ensure migrations directory: %w", err)
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := pingDatabase(ctx, db); err != nil {
		return err
	}

	if err := ensureMigrationsTable(ctx, db); err != nil {
		return err
	}

	printer := ui.NewPrinter(stdout)
	printer.PrintDelight(ui.Delight{
		Command: "init",
		Result:  "ready to migrate",
		Details: []ui.Detail{
			{Label: "Migrations directory", Value: cfg.MigrationsDir},
			{Label: "Database", Value: "connection verified"},
			{Label: "Migrations table", Value: "tinytoe_migrations ensured"},
		},
	})

	return nil
}

func ensureMigrationsDir(dir string) error {
	if dir == "" {
		dir = "migrations"
	}
	return os.MkdirAll(dir, 0o755)
}

func pingDatabase(parent context.Context, db *sql.DB) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	return nil
}

func ensureMigrationsTable(parent context.Context, db *sql.DB) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, migrationsTableDDL); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}
	return nil
}
