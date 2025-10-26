package app

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"tinytoe/internal/config"
	"tinytoe/internal/ui"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// RunReset drops and recreates the target schema, then reapplies all migrations.
func RunReset(ctx context.Context, cfg config.Config, stdin io.Reader, stdout io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if stdout == nil {
		stdout = io.Discard
	}

	if err := requireMigrationsDir(cfg.MigrationsDir); err != nil {
		return err
	}

	if !cfg.Force {
		if cfg.NonInteractive {
			return fmt.Errorf("reset requires confirmation but TINYTOE_NON_INTERACTIVE is set; rerun with --force to proceed")
		}

		ok, err := confirmReset(stdin, stdout, cfg.TargetSchema)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("reset aborted by user")
		}
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := pingDatabase(ctx, db); err != nil {
		return err
	}

	if err := dropTargetSchema(ctx, db, cfg.TargetSchema); err != nil {
		return err
	}

	ui.NewPrinter(stdout).PrintDelight(ui.Delight{
		Command: "reset",
		Result:  fmt.Sprintf("schema %q dropped", cfg.TargetSchema),
	})

	if err := RunInit(ctx, cfg, stdout); err != nil {
		return err
	}

	return RunUp(ctx, cfg, stdout)
}

func confirmReset(stdin io.Reader, stdout io.Writer, schema string) (bool, error) {
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}

	reader := bufio.NewReader(stdin)
	alert := fmt.Sprintf("This will drop the %q schema and erase all managed data.", schema)
	ui.NewPrinter(stdout).PrintWarning(alert)

	prompt := fmt.Sprintf("Proceed with resetting schema %q? [y/N]: ", schema)
	if _, err := fmt.Fprint(stdout, prompt); err != nil {
		return false, fmt.Errorf("prompt reset confirmation: %w", err)
	}

	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read reset confirmation: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	fmt.Fprintln(stdout)

	return response == "y" || response == "yes", nil
}

func dropTargetSchema(parent context.Context, db *sql.DB, schema string) error {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()

	stmt := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quoteIdent(schema))
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("drop schema %q: %w", schema, err)
	}
	return nil
}
