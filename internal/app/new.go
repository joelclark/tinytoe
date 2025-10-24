package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tinytoe/internal/config"
	"tinytoe/internal/ui"
)

// RunNew creates a new migration file using the provided description. It
// returns the absolute path to the generated file.
func RunNew(cfg config.Config, description string, stdin io.Reader, stdout io.Writer) (string, error) {
	slug, err := slugify(description)
	if err != nil {
		return "", err
	}

	if err := ensureMigrationsDirForNew(cfg); err != nil {
		return "", err
	}

	printer := ui.NewPrinter(stdout)

	existing, err := existingMigrationsForSlug(cfg.MigrationsDir, slug)
	if err != nil {
		return "", err
	}

	if len(existing) > 0 {
		message := fmt.Sprintf("%s: %d other migration(s) share the slug %q (%s).", ui.WarningLabel, len(existing), slug, strings.Join(existing, ", "))
		printer.PrintWarning(message)

		if cfg.NonInteractive {
			return "", fmt.Errorf("slug %q already exists; duplicate creation requires confirmation but TINYTOE_NON_INTERACTIVE is set", slug)
		}

		ok, err := confirmDuplicateSlug(stdin, stdout, slug)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", fmt.Errorf("aborted creating migration for duplicate slug %q", slug)
		}
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

	details := []ui.Detail{
		{Label: "Filename", Value: filename},
	}
	if len(existing) > 0 {
		details = append(details, ui.Detail{
			Label: ui.WarningLabel,
			Value: fmt.Sprintf("%d other migration(s) share this slug: %s", len(existing), strings.Join(existing, ", ")),
			Kind:  ui.DetailWarning,
		})
	}

	printer.PrintDelight(ui.Delight{
		Command: "new",
		Result:  "migration created",
		Details: details,
	})

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

func existingMigrationsForSlug(dir, slug string) ([]string, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("migrations directory is required")
	}
	pattern := filepath.Join(dir, fmt.Sprintf("*_%s.sql", slug))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("discover existing migrations for slug %s: %w", slug, err)
	}
	if len(matches) == 0 {
		return nil, nil
	}

	sort.Strings(matches)

	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, filepath.Base(match))
	}
	return names, nil
}

func confirmDuplicateSlug(stdin io.Reader, stdout io.Writer, slug string) (bool, error) {
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}

	reader := bufio.NewReader(stdin)
	prompt := fmt.Sprintf("Create another migration using slug %q? [y/N]: ", slug)
	if _, err := fmt.Fprint(stdout, prompt); err != nil {
		return false, fmt.Errorf("prompt duplicate confirmation: %w", err)
	}

	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read confirmation: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	fmt.Fprintln(stdout)

	return response == "y" || response == "yes", nil
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
