package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tinytoe/internal/config"
	"tinytoe/internal/ui"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// RunUp applies all pending migrations in timestamp order. It assumes the
// configuration has been validated and returns an error when drift is detected.
func RunUp(ctx context.Context, cfg config.Config, stdout io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if stdout == nil {
		stdout = io.Discard
	}

	printer := ui.NewPrinter(stdout)

	if err := requireMigrationsDir(cfg.MigrationsDir); err != nil {
		return err
	}

	files, err := discoverMigrations(cfg.MigrationsDir)
	if err != nil {
		return err
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := pingDatabase(ctx, db); err != nil {
		return err
	}
	if err := ensureTargetSchema(ctx, db, cfg.TargetSchema); err != nil {
		return err
	}
	exists, err := migrationsTableExists(ctx, db, cfg.TargetSchema)
	if err != nil {
		return err
	}
	if !exists {
		if err := RunInit(ctx, cfg, stdout); err != nil {
			return err
		}
	}
	if err := ensureMigrationsTable(ctx, db, cfg.TargetSchema); err != nil {
		return err
	}

	applied, err := loadAppliedMigrations(ctx, db, cfg.TargetSchema)
	if err != nil {
		return err
	}

	if err := detectDrift(files, applied); err != nil {
		return err
	}

	pending := pendingMigrations(files, applied)
	if len(pending) == 0 {
		printer.PrintDelight(ui.Delight{
			Command: "up",
			Result:  "database already up to date",
		})
		return nil
	}

	appliedFiles := make([]string, 0, len(pending))
	for _, migration := range pending {
		if err := applyMigration(ctx, db, cfg.TargetSchema, migration); err != nil {
			return err
		}
		appliedFiles = append(appliedFiles, migration.filename)
		printer.PrintSuccessLine("Applied %s", migration.filename)
	}

	fmt.Fprintln(stdout)

	details := []ui.Detail{
		{Label: "Applied", Value: fmt.Sprintf("%d migration(s)", len(appliedFiles))},
	}

	printer.PrintDelight(ui.Delight{
		Command: "up",
		Result:  "migrations applied successfully",
		Details: details,
	})

	return nil
}

type migrationFile struct {
	version  string
	filename string
	path     string
}

type appliedMigration struct {
	version  string
	filename string
}

func migrationsTableExists(parent context.Context, db *sql.DB, schema string) (bool, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	query := `
SELECT COUNT(*) FROM information_schema.tables
WHERE table_schema = $1 AND table_name = 'tinytoe_migrations'
`
	var count int
	if err := db.QueryRowContext(ctx, query, schema).Scan(&count); err != nil {
		return false, fmt.Errorf("check migrations table: %w", err)
	}
	return count > 0, nil
}

func requireMigrationsDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("migrations directory %s does not exist; run `toe init` first", dir)
		}
		return fmt.Errorf("stat migrations directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("migrations path is not a directory: %s", dir)
	}
	return nil
}

func discoverMigrations(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}

	var files []migrationFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		if len(name) < 20 { // 14 digits + "_" + at least one char + ".sql"
			return nil, fmt.Errorf("invalid migration filename: %s", name)
		}

		version := name[:14]
		if !isDigits(version) {
			return nil, fmt.Errorf("invalid migration version in %s", name)
		}
		if name[14] != '_' {
			return nil, fmt.Errorf("invalid migration filename: %s", name)
		}

		path := filepath.Join(dir, name)
		files = append(files, migrationFile{
			version:  version,
			filename: name,
			path:     path,
		})
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].version == files[j].version {
			return files[i].filename < files[j].filename
		}
		return files[i].version < files[j].version
	})

	for i := 1; i < len(files); i++ {
		if files[i].version == files[i-1].version {
			return nil, fmt.Errorf("duplicate migration version %s (%s and %s)", files[i].version, files[i-1].filename, files[i].filename)
		}
	}

	return files, nil
}

func isDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func loadAppliedMigrations(parent context.Context, db *sql.DB, schema string) ([]appliedMigration, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	query := fmt.Sprintf(`SELECT version, filename FROM %s ORDER BY version`, qualifyIdent(schema, "tinytoe_migrations"))
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("load applied migrations: %w", err)
	}
	defer rows.Close()

	var applied []appliedMigration
	for rows.Next() {
		var row appliedMigration
		if err := rows.Scan(&row.version, &row.filename); err != nil {
			return nil, fmt.Errorf("scan applied migration: %w", err)
		}
		applied = append(applied, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applied migrations: %w", err)
	}
	return applied, nil
}

func detectDrift(files []migrationFile, applied []appliedMigration) error {
	if len(applied) > len(files) {
		return fmt.Errorf("detected drift: database reports more migrations than available files; run `toe reset` to reconcile")
	}

	for i, appliedMigration := range applied {
		file := files[i]
		if file.version != appliedMigration.version {
			return fmt.Errorf("detected drift: expected migration %s but database lists %s; run `toe reset`", file.filename, appliedMigration.filename)
		}
		if file.filename != appliedMigration.filename {
			return fmt.Errorf("detected drift: migration %s recorded as %s in database; run `toe reset`", file.filename, appliedMigration.filename)
		}

		diskFile, err := os.Stat(file.path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("detected drift: applied migration %s no longer exists; run `toe reset`", appliedMigration.filename)
			}
			return fmt.Errorf("stat migration %s: %w", appliedMigration.filename, err)
		}
		if !diskFile.Mode().IsRegular() {
			return fmt.Errorf("detected drift: migration %s is no longer a regular file; run `toe reset`", appliedMigration.filename)
		}
	}

	return nil
}

func pendingMigrations(files []migrationFile, applied []appliedMigration) []migrationFile {
	pending := make([]migrationFile, 0)
	appliedCount := len(applied)
	for i := appliedCount; i < len(files); i++ {
		pending = append(pending, files[i])
	}
	return pending
}

func applyMigration(parent context.Context, db *sql.DB, schema string, file migrationFile) error {
	data, err := os.ReadFile(file.path)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", file.filename, err)
	}

	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction for %s: %w", file.filename, err)
	}

	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL search_path = %s", quoteIdent(schema))); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("set search_path for %s: %w", file.filename, err)
	}

	if _, err := tx.ExecContext(ctx, string(data)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("execute migration %s: %w", file.filename, err)
	}

	insert := fmt.Sprintf(`INSERT INTO %s (version, filename) VALUES ($1, $2)`, qualifyIdent(schema, "tinytoe_migrations"))
	if _, err := tx.ExecContext(ctx, insert, file.version, file.filename); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record migration %s: %w", file.filename, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", file.filename, err)
	}

	return nil
}
