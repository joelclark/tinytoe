package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tinytoe/internal/app"
	"tinytoe/internal/config"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if _, skip := os.LookupEnv("TINYTOE_SKIP_DOTENV"); !skip {
		if err := loadDotenv(".env"); err != nil {
			return fmt.Errorf("load .env: %w", err)
		}
	}

	if len(args) == 0 || isHelp(args[0]) {
		printUsage(stdout)
		return nil
	}

	switch args[0] {
	case "init":
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		return app.RunInit(context.Background(), cfg, stdout)
	case "up":
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		return app.RunUp(context.Background(), cfg, stdout)
	case "reset":
		return runResetCommand(args[1:], stdout, stderr)
	case "new":
		return runNewCommand(args[1:], stdout, stderr)
	case "help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func printUsage(w io.Writer) {
	if w == nil {
		w = io.Discard
	}

	fmt.Fprintln(w, "Tiny Toe — lightweight PostgreSQL migrations")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  tinytoe init     Initialize migrations directory and database state")
	fmt.Fprintln(w, "  tinytoe up       Apply pending migrations to the database")
	fmt.Fprintln(w, "  tinytoe reset    Drop the target schema and reapply all migrations (tinytoe reset [--force])")
	fmt.Fprintln(w, "  tinytoe new      Generate a new migration (tinytoe new [--force] <description>)")
	fmt.Fprintln(w, "  tinytoe help     Show this message")
}

func isHelp(arg string) bool {
	return arg == "--help" || arg == "-h"
}

func loadDotenv(path string) error {
	if path == "" {
		return nil
	}

	expanded := os.ExpandEnv(path)
	if !filepath.IsAbs(expanded) {
		expanded = filepath.Join(".", expanded)
	}

	file, err := os.Open(expanded)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		if !strings.Contains(line, "=") {
			return fmt.Errorf("invalid line in %s: %q", path, line)
		}

		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			return fmt.Errorf("invalid environment variable declaration in %s: %q", path, line)
		}

		// Support quoted values.
		if len(value) >= 2 && strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			if unquoted, err := strconv.Unquote(value); err == nil {
				value = unquoted
			}
		} else if len(value) >= 2 && strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			value = value[1 : len(value)-1]
		}

		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("set %s from %s: %w", key, path, err)
			}
		}
	}

	return scanner.Err()
}

func runNewCommand(args []string, stdout, stderr io.Writer) error {
	for len(args) > 0 && isHelp(args[0]) {
		printNewUsage(stdout)
		return nil
	}

	if len(args) == 0 {
		printNewUsage(stderr)
		return fmt.Errorf("description is required")
	}

	forceFlag := false
	forceSpecified := false
	var descriptionParts []string
	for _, arg := range args {
		switch {
		case arg == "--force":
			forceFlag = true
			forceSpecified = true
		case strings.HasPrefix(arg, "--"):
			printNewUsage(stderr)
			return fmt.Errorf("unknown flag %s", arg)
		default:
			descriptionParts = append(descriptionParts, arg)
		}
	}

	if len(descriptionParts) == 0 {
		printNewUsage(stderr)
		return fmt.Errorf("description is required")
	}

	description := strings.Join(descriptionParts, " ")

	requireDatabase := false
	opts := config.LoadOptions{
		RequireDatabase: &requireDatabase,
	}
	if forceSpecified {
		opts.ForceOverride = &forceFlag
	}

	cfg, err := config.LoadWithOptions(opts)
	if err != nil {
		return err
	}

	_, err = app.RunNew(cfg, description, os.Stdin, stdout)
	return err
}

func printNewUsage(w io.Writer) {
	if w == nil {
		w = io.Discard
	}
	fmt.Fprintln(w, "Usage: tinytoe new [--force] <description>")
	fmt.Fprintln(w, "Creates a new migration file using a UTC timestamp prefix and the provided description.")
}

func runResetCommand(args []string, stdout, stderr io.Writer) error {
	for len(args) > 0 && isHelp(args[0]) {
		printResetUsage(stdout)
		return nil
	}

	forceFlag := false
	forceSpecified := false
	for _, arg := range args {
		switch {
		case arg == "--force":
			forceFlag = true
			forceSpecified = true
		case strings.HasPrefix(arg, "--"):
			printResetUsage(stderr)
			return fmt.Errorf("unknown flag %s", arg)
		default:
			printResetUsage(stderr)
			return fmt.Errorf("unexpected argument %s", arg)
		}
	}

	opts := config.LoadOptions{}
	if forceSpecified {
		opts.ForceOverride = &forceFlag
	}

	cfg, err := config.LoadWithOptions(opts)
	if err != nil {
		return err
	}

	return app.RunReset(context.Background(), cfg, os.Stdin, stdout)
}

func printResetUsage(w io.Writer) {
	if w == nil {
		w = io.Discard
	}
	fmt.Fprintln(w, "Usage: tinytoe reset [--force]")
	fmt.Fprintln(w, "Drops the target schema, recreates it, and reapplies all migrations from disk.")
	fmt.Fprintln(w, "Use --force to skip the confirmation prompt (or set TINYTOE_FORCE=1).")
}
