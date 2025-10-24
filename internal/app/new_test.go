package app_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tinytoe/internal/app"
	"tinytoe/internal/config"
)

func TestRunNewCreatesMigrationFile(t *testing.T) {
	t.Setenv("USER", "tinytoe-test")

	tempDir := t.TempDir()
	cfg := config.Config{
		MigrationsDir: tempDir,
	}

	var out bytes.Buffer
	path, err := app.RunNew(cfg, "Add Users Table", &out)
	if err != nil {
		t.Fatalf("RunNew: %v", err)
	}

	if !strings.HasSuffix(filepath.Base(path), "_add_users_table.sql") {
		t.Fatalf("expected filename to use slug, got %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) < 5 {
		t.Fatalf("expected migration header, got %q", string(data))
	}
	if lines[0] != "-- Tiny Toe Migration" {
		t.Fatalf("unexpected header line: %s", lines[0])
	}
	if !strings.HasPrefix(lines[1], "-- Version: ") {
		t.Fatalf("expected version line, got %s", lines[1])
	}
	version := strings.TrimPrefix(lines[1], "-- Version: ")
	if len(version) != 14 {
		t.Fatalf("expected version to be 14 digits, got %q", version)
	}
	if !strings.HasPrefix(lines[2], "-- Filename: ") {
		t.Fatalf("expected filename line, got %s", lines[2])
	}
	filenameLine := strings.TrimPrefix(lines[2], "-- Filename: ")
	if filenameLine != filepath.Base(path) {
		t.Fatalf("expected filename %s, got %s", filepath.Base(path), filenameLine)
	}
	if !strings.HasPrefix(lines[3], "-- Created At (UTC): ") {
		t.Fatalf("expected created at line, got %s", lines[3])
	}
	if !strings.HasPrefix(lines[4], "-- Created By: ") {
		t.Fatalf("expected created by line, got %s", lines[4])
	}
	if out.Len() == 0 || !strings.Contains(out.String(), "Created migration") {
		t.Fatalf("expected output message, got %q", out.String())
	}
}

func TestRunNewCreatesDirWithForce(t *testing.T) {
	t.Setenv("USER", "tinytoe-test")

	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "migrations")
	cfg := config.Config{
		MigrationsDir: targetDir,
		Force:         true,
	}

	if _, err := app.RunNew(cfg, "Forced Dir", nil); err != nil {
		t.Fatalf("RunNew with force: %v", err)
	}

	if stat, err := os.Stat(targetDir); err != nil || !stat.IsDir() {
		t.Fatalf("expected migrations dir to be created, stat err=%v", err)
	}
}

func TestRunNewFailsWithoutForceWhenDirMissing(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "missing")
	cfg := config.Config{
		MigrationsDir: targetDir,
	}

	if _, err := app.RunNew(cfg, "Needs Init", nil); err == nil || !strings.Contains(err.Error(), "toe init") {
		t.Fatalf("expected error suggesting init, got %v", err)
	}
}

func TestRunNewRejectsEmptyDescription(t *testing.T) {
	cfg := config.Config{MigrationsDir: t.TempDir()}
	if _, err := app.RunNew(cfg, "   ", nil); err == nil || !strings.Contains(err.Error(), "description") {
		t.Fatalf("expected description error, got %v", err)
	}
}

func TestRunNewSlugifiesComplexDescription(t *testing.T) {
	cfg := config.Config{MigrationsDir: t.TempDir()}
	path, err := app.RunNew(cfg, "Create!!! Primary#Key", nil)
	if err != nil {
		t.Fatalf("RunNew: %v", err)
	}
	if !strings.HasSuffix(filepath.Base(path), "_create_primary_key.sql") {
		t.Fatalf("expected slugified filename, got %s", filepath.Base(path))
	}
}
