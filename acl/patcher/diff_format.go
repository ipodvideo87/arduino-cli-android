// Package patcher — diff_format.go
//
// Renders a slice of PatchPlans as a human-readable diff-style plan that
// operators can review in a terminal or CI log before committing any changes to
// disk.
//
// Output format mirrors a simplified unified diff:
//
//	=== path/to/binary ===
//	--- PT_INTERP (current)
//	+++ PT_INTERP (proposed)
//	  reason: <why>
//
// When a binary needs no patching a short "OK" line is printed instead so
// operators know it was inspected.
package patcher

import (
	"fmt"
	"io"
	"strings"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorCyan   = "\033[36m"
	colorYellow = "\033[33m"
	colorGray   = "\033[90m"
)

// FormatOptions controls how the diff plan is rendered.
type FormatOptions struct {
	// Color enables ANSI colour codes.  Set to false for plain-text output
	// (e.g. log files, CI systems that do not support colour).
	Color bool

	// Verbose prints one-liner status for binaries that need no patching.
	// When false only files that need changes (or are skipped with a
	// non-trivial reason) are printed.
	Verbose bool
}

// WritePlan renders plans to w according to opts.
// It always returns the first write error encountered, if any.
func WritePlan(w io.Writer, plans []PatchPlan, opts FormatOptions) error {
	summary := Summarise(plans)

	for _, plan := range plans {
		if err := writePlanEntry(w, plan, opts); err != nil {
			return err
		}
	}

	// Footer summary line.
	line := fmt.Sprintf(
		"\n── Dry-run summary: %d file(s) — %d need patching, %d already OK, %d skipped ──\n",
		summary.Total, summary.NeedPatching, summary.AlreadyOK, summary.Skipped,
	)
	if opts.Color {
		line = colorCyan + line + colorReset
	}
	_, err := fmt.Fprint(w, line)
	return err
}

func writePlanEntry(w io.Writer, plan PatchPlan, opts FormatOptions) error {
	header := fmt.Sprintf("=== %s ===\n", plan.Path)
	if opts.Color {
		header = colorCyan + header + colorReset
	}

	switch {
	case plan.Skipped:
		if !opts.Verbose {
			return nil
		}
		msg := fmt.Sprintf("    [skipped] %s\n\n", plan.SkipReason)
		if opts.Color {
			msg = colorGray + msg + colorReset
		}
		if _, err := fmt.Fprint(w, header); err != nil {
			return err
		}
		_, err := fmt.Fprint(w, msg)
		return err

	case !plan.NeedsPatching():
		if !opts.Verbose {
			return nil
		}
		msg := "    [ok] no patching required\n\n"
		if opts.Color {
			msg = colorGray + msg + colorReset
		}
		if _, err := fmt.Fprint(w, header); err != nil {
			return err
		}
		_, err := fmt.Fprint(w, msg)
		return err

	default:
		if _, err := fmt.Fprint(w, header); err != nil {
			return err
		}
		for _, edit := range plan.Edits {
			if err := writeEditBlock(w, edit, opts); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}

func writeEditBlock(w io.Writer, edit FieldEdit, opts FormatOptions) error {
	currentLabel := fmt.Sprintf("--- %s\t(current)  %s\n", edit.Field, displayValue(edit.Current))
	proposedLabel := fmt.Sprintf("+++ %s\t(proposed) %s\n", edit.Field, displayValue(edit.Proposed))
	reasonLabel := fmt.Sprintf("    reason: %s\n", edit.Reason)

	if opts.Color {
		currentLabel = colorRed + currentLabel + colorReset
		proposedLabel = colorGreen + proposedLabel + colorReset
		reasonLabel = colorYellow + reasonLabel + colorReset
	}

	for _, line := range []string{currentLabel, proposedLabel, reasonLabel} {
		if _, err := fmt.Fprint(w, line); err != nil {
			return err
		}
	}
	return nil
}

// displayValue renders a field value for display.  Empty string is shown as
// the literal token "<absent>" so readers understand the field was not set.
func displayValue(v string) string {
	if v == "" {
		return "<absent>"
	}
	return v
}

// WritePlanText is a convenience wrapper that writes a plain-text (no colour)
// diff plan to w.
func WritePlanText(w io.Writer, plans []PatchPlan, verbose bool) error {
	return WritePlan(w, plans, FormatOptions{Color: false, Verbose: verbose})
}

// PlanToString renders plans as a plain-text string.  Useful in tests.
func PlanToString(plans []PatchPlan, verbose bool) string {
	var sb strings.Builder
	_ = WritePlanText(&sb, plans, verbose)
	return sb.String()
}
