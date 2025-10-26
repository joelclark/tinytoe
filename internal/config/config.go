package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config captures the minimal configuration needed for Tiny Toe operations.
type Config struct {
	DatabaseURL    string
	MigrationsDir  string
	Force          bool
	NonInteractive bool
	TargetSchema   string
}

// LoadOptions tune how LoadWithOptions behaves for individual commands.
type LoadOptions struct {
	// RequireDatabase controls whether DATABASE_URL must be provided.
	RequireDatabase *bool
	// ForceOverride allows callers to bypass environment detection for the force flag.
	ForceOverride *bool
}

// Load reads configuration from environment variables with the default options,
// requiring a database URL.
func Load() (Config, error) {
	return LoadWithOptions(LoadOptions{})
}

// LoadWithOptions reads configuration from environment variables, applying
// defaults and validation according to the supplied options. The caller is
// expected to preload any additional configuration sources (e.g. .env files).
func LoadWithOptions(opts LoadOptions) (Config, error) {
	cfg := Config{
		DatabaseURL:   strings.TrimSpace(os.Getenv("DATABASE_URL")),
		MigrationsDir: strings.TrimSpace(os.Getenv("TINYTOE_MIGRATIONS_DIR")),
		TargetSchema:  strings.TrimSpace(os.Getenv("TINYTOE_TARGET_SCHEMA")),
	}

	if cfg.MigrationsDir == "" {
		cfg.MigrationsDir = "migrations"
	}

	// Normalize relative paths to avoid subtle duplicates.
	if !filepath.IsAbs(cfg.MigrationsDir) {
		cfg.MigrationsDir = filepath.Clean(cfg.MigrationsDir)
	}

	if cfg.TargetSchema == "" {
		cfg.TargetSchema = "public"
	}
	if err := validateTargetSchema(cfg.TargetSchema); err != nil {
		return Config{}, err
	}

	force, err := parseBoolEnv(os.Getenv("TINYTOE_FORCE"), "TINYTOE_FORCE")
	if err != nil {
		return Config{}, err
	}
	cfg.Force = force

	nonInteractive, err := parseBoolEnv(os.Getenv("TINYTOE_NON_INTERACTIVE"), "TINYTOE_NON_INTERACTIVE")
	if err != nil {
		return Config{}, err
	}
	cfg.NonInteractive = nonInteractive

	if opts.ForceOverride != nil {
		cfg.Force = *opts.ForceOverride
	}

	requireDatabase := true
	if opts.RequireDatabase != nil {
		requireDatabase = *opts.RequireDatabase
	}

	if requireDatabase && cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func parseBoolEnv(raw, name string) (bool, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}

func validateTargetSchema(schema string) error {
	if schema == "" {
		return fmt.Errorf("TINYTOE_TARGET_SCHEMA must not be empty")
	}

	switch strings.ToLower(schema) {
	case "pg_catalog", "pg_toast", "pg_temp", "pg_temp_1", "pg_toast_temp_1":
		return fmt.Errorf("TINYTOE_TARGET_SCHEMA %q is a reserved PostgreSQL schema", schema)
	}

	if strings.Contains(schema, ",") {
		return fmt.Errorf("TINYTOE_TARGET_SCHEMA must reference a single schema, got %q", schema)
	}

	return nil
}
