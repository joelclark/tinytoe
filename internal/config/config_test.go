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
	if cfg.Force {
		t.Fatalf("expected force flag to default to false")
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

func TestLoadWithOptionsAllowsMissingDatabase(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	requireDatabase := false
	cfg, err := config.LoadWithOptions(config.LoadOptions{
		RequireDatabase: &requireDatabase,
	})
	if err != nil {
		t.Fatalf("LoadWithOptions: %v", err)
	}
	if cfg.DatabaseURL != "" {
		t.Fatalf("expected empty DatabaseURL when not required")
	}
}

func TestLoadParsesForceEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example.com/db")
	t.Setenv("TINYTOE_FORCE", "1")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Force {
		t.Fatalf("expected force from env to be true")
	}
}

func TestLoadWithOptionsForceOverride(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example.com/db")
	t.Setenv("TINYTOE_FORCE", "0")

	override := true
	cfg, err := config.LoadWithOptions(config.LoadOptions{
		ForceOverride: &override,
	})
	if err != nil {
		t.Fatalf("LoadWithOptions: %v", err)
	}
	if !cfg.Force {
		t.Fatalf("expected override to set force true")
	}
}

func TestLoadFailsOnInvalidForceEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example.com/db")
	t.Setenv("TINYTOE_FORCE", "definitely-not-bool")

	if _, err := config.Load(); err == nil || !strings.Contains(err.Error(), "TINYTOE_FORCE") {
		t.Fatalf("expected error mentioning TINYTOE_FORCE, got %v", err)
	}
}
