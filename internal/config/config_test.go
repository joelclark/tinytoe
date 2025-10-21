package config_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tinytoe/internal/config"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := config.Load(); err == nil {
		t.Fatalf("expected error when DATABASE_URL is missing")
	}
}

func TestLoadUsesDefaultsWhenUnset(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example.com/db")
	t.Setenv("TINYTOE_MIGRATIONS_DIR", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.MigrationsDir != "migrations" {
		t.Fatalf("expected default migrations dir, got %q", cfg.MigrationsDir)
	}
}

func TestLoadRespectsMigrationsDirEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example.com/db")
	rawDir := " ./foo/../bar  "
	if runtime.GOOS == "windows" {
		rawDir = strings.ReplaceAll(rawDir, "/", `\`)
	}
	t.Setenv("TINYTOE_MIGRATIONS_DIR", rawDir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	expected := filepath.Clean(strings.TrimSpace(rawDir))
	if cfg.MigrationsDir != expected {
		t.Fatalf("expected cleaned migrations dir %q, got %q", expected, cfg.MigrationsDir)
	}
}
