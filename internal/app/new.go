package app

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"tinytoe/internal/config"
)

// RunNew creates a new migration file using the provided description. It
// returns the absolute path to the generated file.
func RunNew(cfg config.Config, description string, stdout io.Writer) (string, error) {
	slug, err := slugify(description)
	if err != nil {
		return "", err
	}

	if err := ensureMigrationsDirForNew(cfg); err != nil {
		return "", err
	}

	now := time.Now().UTC()
	version := now.Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.sql", version, slug)
	fullPath := filepath.Join(cfg.MigrationsDir, filename)

	file, err := os.OpenFile(fullPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("migration already exists: %s", filename)
		}
		return "", fmt.Errorf("create migration file: %w", err)
	}
	defer file.Close()

	header := buildMigrationHeader(version, filename, now, createdBy())
	if _, err := file.WriteString(header); err != nil {
		return "", fmt.Errorf("write migration header: %w", err)
	}

	if stdout != nil {
		fmt.Fprintf(stdout, "Created migration %s\n", filename)
	}

	return fullPath, nil
}

func ensureMigrationsDirForNew(cfg config.Config) error {
	info, err := os.Stat(cfg.MigrationsDir)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("migrations path is not a directory: %s", cfg.MigrationsDir)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("stat migrations directory: %w", err)
	}
	if !cfg.Force {
		return fmt.Errorf("migrations directory %s does not exist; run `toe init` or pass --force", cfg.MigrationsDir)
	}
	if err := os.MkdirAll(cfg.MigrationsDir, 0o755); err != nil {
		return fmt.Errorf("create migrations directory: %w", err)
	}
	return nil
}

func slugify(description string) (string, error) {
	desc := strings.TrimSpace(description)
	if desc == "" {
		return "", fmt.Errorf("description is required")
	}

	var b strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToLower(desc) {
		switch {
		case 'a' <= r && r <= 'z':
			b.WriteRune(r)
			lastUnderscore = false
		case '0' <= r && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}

	slug := strings.Trim(b.String(), "_")
	if slug == "" {
		return "", fmt.Errorf("description %q results in an empty slug", description)
	}
	return slug, nil
}

func buildMigrationHeader(version, filename string, createdAt time.Time, createdBy string) string {
	return fmt.Sprintf(`-- Tiny Toe Migration
-- Version: %s
-- Filename: %s
-- Created At (UTC): %s
-- Created By: %s

`, version, filename, createdAt.UTC().Format(time.RFC3339), createdBy)
}

func createdBy() string {
	username := strings.TrimSpace(os.Getenv("USER"))
	if username == "" {
		username = strings.TrimSpace(os.Getenv("USERNAME"))
	}
	if username == "" {
		if current, err := user.Current(); err == nil && current.Username != "" {
			username = current.Username
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}

	username = strings.TrimSpace(username)
	hostname = strings.TrimSpace(hostname)

	switch {
	case username != "" && hostname != "":
		return fmt.Sprintf("%s@%s", username, hostname)
	case username != "":
		return username
	case hostname != "":
		return hostname
	default:
		return "unknown"
	}
}
