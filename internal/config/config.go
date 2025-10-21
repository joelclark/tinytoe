package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config captures the minimal configuration needed for Tiny Toe operations.
type Config struct {
	DatabaseURL   string
	MigrationsDir string
}

// Load reads configuration from environment variables, applying defaults where
// appropriate. The caller is expected to have already loaded .env files or any
// other sources that populate the process environment.
func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:   strings.TrimSpace(os.Getenv("DATABASE_URL")),
		MigrationsDir: strings.TrimSpace(os.Getenv("TINYTOE_MIGRATIONS_DIR")),
	}

	if cfg.MigrationsDir == "" {
		cfg.MigrationsDir = "migrations"
	}

	// Normalize relative paths to avoid subtle duplicates.
	if !filepath.IsAbs(cfg.MigrationsDir) {
		cfg.MigrationsDir = filepath.Clean(cfg.MigrationsDir)
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}
