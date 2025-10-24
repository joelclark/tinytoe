package app_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tinytoe/internal/app"
	"tinytoe/internal/config"
	"tinytoe/internal/ui"
)

func TestRunNewCreatesMigrationFile(t *testing.T) {
	t.Setenv("USER", "tinytoe-test")

	tempDir := t.TempDir()
	cfg := config.Config{
		MigrationsDir: tempDir,
	}

	var out bytes.Buffer
	path, err := app.RunNew(cfg, "Add Users Table", nil, &out)
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
	output := out.String()
	if !strings.Contains(output, fmt.Sprintf("tinytoe new %s migration created", ui.Arrow)) {
		t.Fatalf("expected success message, got %q", output)
	}
	if !strings.Contains(output, "Filename: "+filepath.Base(path)) {
		t.Fatalf("expected filename in output, got %q", output)
	}
}

func TestRunNewWarnsWhenSlugExists(t *testing.T) {
	t.Setenv("USER", "tinytoe-test")

	tempDir := t.TempDir()
	cfg := config.Config{
		MigrationsDir: tempDir,
	}

	existingFilename := "20230101010101_add_users_table.sql"
	existingPath := filepath.Join(tempDir, existingFilename)
	if err := os.WriteFile(existingPath, []byte("-- existing migration\n"), 0o644); err != nil {
		t.Fatalf("write existing migration: %v", err)
	}

	var out bytes.Buffer
	input := bytes.NewBufferString("y\n")
	path, err := app.RunNew(cfg, "Add Users Table", input, &out)
	if err != nil {
		t.Fatalf("RunNew: %v", err)
	}

	if path == existingPath {
		t.Fatalf("expected new migration file, got existing path %s", path)
	}

	output := out.String()
	expectedNote := fmt.Sprintf("%s: 1 other migration(s) share this slug: %s", ui.WarningLabel, existingFilename)
	if !strings.Contains(output, expectedNote) {
		t.Fatalf("expected warning detail about existing slug, got %q", output)
	}
	if !strings.Contains(output, fmt.Sprintf("%s %s: 1 other migration(s) share the slug %q (%s).", ui.WarningEmoji, ui.WarningLabel, "add_users_table", existingFilename)) {
		t.Fatalf("expected warning line, got %q", output)
	}
	if !strings.Contains(output, fmt.Sprintf("Create another migration using slug %q? [y/N]: ", "add_users_table")) {
		t.Fatalf("expected confirmation prompt, got %q", output)
	}
}

func TestRunNewAbortsWhenSlugExistsAndDeclined(t *testing.T) {
	t.Setenv("USER", "tinytoe-test")

	tempDir := t.TempDir()
	cfg := config.Config{
		MigrationsDir: tempDir,
	}

	existingFilename := "20230101010101_add_users_table.sql"
	if err := os.WriteFile(filepath.Join(tempDir, existingFilename), []byte("-- existing migration\n"), 0o644); err != nil {
		t.Fatalf("write existing migration: %v", err)
	}

	var out bytes.Buffer
	input := bytes.NewBufferString("n\n")

	path, err := app.RunNew(cfg, "Add Users Table", input, &out)
	if err == nil {
		t.Fatalf("expected error when declining duplicate, got path %s", path)
	}

	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected only the original file to remain, got %d entries", len(files))
	}
}

func TestRunNewDuplicateRespectsNonInteractive(t *testing.T) {
	t.Setenv("USER", "tinytoe-test")

	tempDir := t.TempDir()
	cfg := config.Config{
		MigrationsDir:  tempDir,
		NonInteractive: true,
	}

	if err := os.WriteFile(filepath.Join(tempDir, "20230101010101_add_users_table.sql"), []byte("-- existing migration\n"), 0o644); err != nil {
		t.Fatalf("write existing migration: %v", err)
	}

	var out bytes.Buffer
	path, err := app.RunNew(cfg, "Add Users Table", nil, &out)
	if err == nil {
		t.Fatalf("expected error when non-interactive, got path %s", path)
	}
	if !strings.Contains(err.Error(), "TINYTOE_NON_INTERACTIVE") {
		t.Fatalf("expected error referencing TINYTOE_NON_INTERACTIVE, got %v", err)
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

	if _, err := app.RunNew(cfg, "Forced Dir", nil, nil); err != nil {
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

	if _, err := app.RunNew(cfg, "Needs Init", nil, nil); err == nil || !strings.Contains(err.Error(), "toe init") {
		t.Fatalf("expected error suggesting init, got %v", err)
	}
}

func TestRunNewRejectsEmptyDescription(t *testing.T) {
	cfg := config.Config{MigrationsDir: t.TempDir()}
	if _, err := app.RunNew(cfg, "   ", nil, nil); err == nil || !strings.Contains(err.Error(), "description") {
		t.Fatalf("expected description error, got %v", err)
	}
}

func TestRunNewSlugifiesComplexDescription(t *testing.T) {
	cfg := config.Config{MigrationsDir: t.TempDir()}
	path, err := app.RunNew(cfg, "Create!!! Primary#Key", nil, nil)
	if err != nil {
		t.Fatalf("RunNew: %v", err)
	}
	if !strings.HasSuffix(filepath.Base(path), "_create_primary_key.sql") {
		t.Fatalf("expected slugified filename, got %s", filepath.Base(path))
	}
}
