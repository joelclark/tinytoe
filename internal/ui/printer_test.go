package ui_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"tinytoe/internal/ui"
)

func TestPrinterPrintDelightFormatsWithoutColor(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.NewPrinter(&buf)

	printer.PrintDelight(ui.Delight{
		Command: "init",
		Result:  "ready to migrate",
		Details: []ui.Detail{
			{Label: "Migrations directory", Value: "/tmp/migrations"},
			{Label: "Database", Value: "connection verified"},
		},
	})

	got := buf.String()
	want := "" +
		fmt.Sprintf("tinytoe init %s ready to migrate\n", ui.Arrow) +
		"  - Migrations directory: /tmp/migrations\n" +
		"  - Database: connection verified\n\n"

	if got != want {
		t.Fatalf("unexpected output:\nwant:\n%s\ngot:\n%s", want, got)
	}

	if strings.Contains(got, "\033") {
		t.Fatalf("expected no ANSI escape sequences in non-tty output, got %q", got)
	}
}

func TestPrinterAvoidsDoubleTinytoePrefix(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.NewPrinter(&buf)

	printer.PrintDelight(ui.Delight{
		Command: "tinytoe status",
		Result:  "all migrations applied",
	})

	got := buf.String()
	want := "" +
		fmt.Sprintf("tinytoe status %s all migrations applied\n\n", ui.Arrow)

	if got != want {
		t.Fatalf("unexpected output:\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestPrinterPrintsWarningDetail(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.NewPrinter(&buf)

	printer.PrintDelight(ui.Delight{
		Command: "new",
		Result:  "migration created",
		Details: []ui.Detail{
			{Label: "Filename", Value: "20240101010101_add_users.sql"},
			{Label: "Heads up", Value: "1 other migration shares this slug", Kind: ui.DetailWarning},
		},
	})

	got := buf.String()
	if !strings.Contains(got, "  - ⚠️ Heads up: 1 other migration shares this slug") {
		t.Fatalf("expected warning detail, got %q", got)
	}
}

func TestPrinterPrintWarningLine(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.NewPrinter(&buf)

	printer.PrintWarning("Potential duplicate detected")

	want := fmt.Sprintf("%s %s\n", ui.WarningEmoji, "Potential duplicate detected")
	if buf.String() != want {
		t.Fatalf("unexpected warning output, want %q got %q", want, buf.String())
	}
}
