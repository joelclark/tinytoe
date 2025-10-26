package app

import (
	"context"
	"io"

	"tinytoe/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// RunReset drops and recreates the target schema, then reapplies all migrations.
func RunReset(ctx context.Context, cfg config.Config, stdin io.Reader, stdout io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := requireMigrationsDir(cfg.MigrationsDir); err != nil {
		return err
	}

	if err := RunDropAll(ctx, cfg, stdin, stdout); err != nil {
		return err
	}

	if err := RunInit(ctx, cfg, stdout); err != nil {
		return err
	}

	return RunUp(ctx, cfg, stdout)
}
