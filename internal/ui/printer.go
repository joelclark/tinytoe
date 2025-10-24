package ui

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// DetailKind describes how a detail line should be rendered.
type DetailKind int

const (
	// DetailInfo renders using the standard label/value styling.
	DetailInfo DetailKind = iota
	// DetailWarning renders the detail as a highlighted warning message.
	DetailWarning
)

// Detail represents a label/value pair rendered as part of a block of output.
type Detail struct {
	Label string
	Value string
	Kind  DetailKind
}

// Delight encapsulates the formatted output for a successful command.
type Delight struct {
	Command string
	Result  string
	Details []Detail
}

// Printer is responsible for producing consistently styled command output.
type Printer struct {
	w           io.Writer
	useColor    bool
	colorScheme palette
}

// NewPrinter constructs a Printer targeting the supplied writer.
func NewPrinter(w io.Writer) Printer {
	if w == nil {
		w = io.Discard
	}
	return Printer{
		w:           w,
		useColor:    shouldUseColor(w),
		colorScheme: defaultPalette(),
	}
}

// PrintDelight renders a Delight block using the printer's style rules.
func (p Printer) PrintDelight(block Delight) {
	if p.w == nil {
		return
	}

	command := strings.TrimSpace(block.Command)
	if command == "" {
		command = "tinytoe"
	} else if !strings.HasPrefix(command, "tinytoe") {
		command = "tinytoe " + command
	}

	line := p.decorateTitle(command)
	if block.Result != "" {
		line += " "
		line += p.decorateMuted(Arrow)
		line += " "
		line += p.decorateMessage(strings.TrimSpace(block.Result))
	}

	fmt.Fprintln(p.w, line)

	for _, detail := range block.Details {
		label := strings.TrimSpace(detail.Label)
		value := strings.TrimSpace(detail.Value)

		if detail.Kind == DetailWarning {
			message := value
			if label != "" {
				if message != "" {
					message = label + ": " + message
				} else {
					message = label
				}
			}
			message = strings.TrimSpace(message)
			if message != "" {
				fmt.Fprintf(p.w, "  - %s\n", p.decorateWarning(fmt.Sprintf("%s %s", WarningEmoji, message)))
			}
			continue
		}

		switch {
		case label != "" && value != "":
			fmt.Fprintf(p.w, "  - %s: %s\n", p.decorateLabel(label), p.decorateValue(value))
		case value != "":
			fmt.Fprintf(p.w, "  - %s\n", p.decorateValue(value))
		case label != "":
			fmt.Fprintf(p.w, "  - %s\n", p.decorateLabel(label))
		}
	}

	fmt.Fprintln(p.w)
}

// PrintWarning renders a highlighted warning line distinct from delight blocks.
func (p Printer) PrintWarning(message string) {
	if p.w == nil {
		return
	}
	text := strings.TrimSpace(message)
	if text == "" {
		return
	}
	fmt.Fprintln(p.w, p.decorateWarning(fmt.Sprintf("%s %s", WarningEmoji, text)))
}

func shouldUseColor(w io.Writer) bool {
	if disableColor() {
		return false
	}

	type fdWriter interface {
		Fd() uintptr
	}

	if f, ok := w.(fdWriter); ok {
		return term.IsTerminal(int(f.Fd()))
	}

	return false
}

func disableColor() bool {
	if value := strings.TrimSpace(os.Getenv("NO_COLOR")); value != "" {
		return true
	}

	if value := strings.TrimSpace(os.Getenv("TINYTOE_NO_COLOR")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return true
		}
		return parsed
	}

	return false
}

type palette struct {
	title   string
	message string
	label   string
	value   string
	muted   string
	warning string
}

func defaultPalette() palette {
	return palette{
		title:   "\033[1;36m", // bright cyan
		message: "\033[1;32m", // bright green
		label:   "\033[90m",   // muted gray
		value:   "\033[1;37m", // bright white
		muted:   "\033[37m",   // soft white
		warning: "\033[1;33m", // bright yellow
	}
}

func (p Printer) decorateTitle(text string) string {
	if !p.useColor {
		return text
	}
	return p.colorScheme.title + text + resetCode
}

func (p Printer) decorateMessage(text string) string {
	if !p.useColor {
		return text
	}
	return p.colorScheme.message + text + resetCode
}

func (p Printer) decorateLabel(text string) string {
	if !p.useColor {
		return text
	}
	return p.colorScheme.label + text + resetCode
}

func (p Printer) decorateValue(text string) string {
	if !p.useColor {
		return text
	}
	return p.colorScheme.value + text + resetCode
}

func (p Printer) decorateMuted(text string) string {
	if !p.useColor {
		return text
	}
	return p.colorScheme.muted + text + resetCode
}

func (p Printer) decorateWarning(text string) string {
	if !p.useColor {
		return text
	}
	return p.colorScheme.warning + text + resetCode
}

const resetCode = "\033[0m"

// Arrow identifies the visual separator between the command and its result.
const Arrow = "➜"

// WarningEmoji is the leading symbol for warning messages.
const WarningEmoji = "⚠️"

// WarningLabel standardizes the text shown for warning details.
const WarningLabel = "WARNING"
