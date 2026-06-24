// acl-scan inspects ELF binaries for Android/ACL compatibility and emits
// structured compatibility reports.
//
// Usage:
//
//	acl-scan [--output json|text] compat    <file> [file...]
//	acl-scan [--output json|text] compat-json <file> [file...]   (alias; always JSON)
//	acl-scan validate-compat      <file> [file...]
//	acl-scan validate-compat-json <file> [file...]
//
// Flags:
//
//	--output json|text   Output format (default: text for compat, json for compat-json)
//
// Exit codes:
//
//	0  — all binaries are Android-compatible (no patching needed)
//	1  — one or more binaries need patching or are unsupported
//	2  — usage / flag error
//	3  — internal error (I/O, JSON marshal failure)
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/arduino/arduino-cli/acl/scanner"
)

// outputFormat controls how results are rendered.
type outputFormat string

const (
	formatText outputFormat = "text"
	formatJSON outputFormat = "json"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	// Parse the leading --output flag before the sub-command name so that both
	//   acl-scan --output json compat ...
	//   acl-scan compat --output json ...
	// work naturally.
	format, remaining, err := parseOutputFlag(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "acl-scan: %v\n", err)
		printUsage()
		return 2
	}

	if len(remaining) == 0 {
		printUsage()
		return 2
	}

	sub := remaining[0]
	targets := remaining[1:]

	// Strip any trailing --output flag that appeared after the sub-command name.
	format2, targets, err := parseOutputFlag(targets)
	if err != nil {
		fmt.Fprintf(os.Stderr, "acl-scan: %v\n", err)
		printUsage()
		return 2
	}
	// The post-sub-command --output wins if the pre-sub-command was default.
	if format == formatText && format2 != formatText {
		format = format2
	}

	switch sub {
	case "compat":
		return runCompat(targets, format)
	case "compat-json":
		// compat-json is always JSON regardless of --output.
		return runCompat(targets, formatJSON)
	case "validate-compat":
		return runValidateCompat(targets, format)
	case "validate-compat-json":
		return runValidateCompat(targets, formatJSON)
	case "help", "--help", "-h":
		printUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "acl-scan: unknown command %q\n", sub)
		printUsage()
		return 2
	}
}

// ─── Sub-command: compat ─────────────────────────────────────────────────────

// runCompat scans the given files and emits a compatibility report.
// Exit code 1 when any binary needs patching or is unsupported/unknown.
func runCompat(targets []string, format outputFormat) int {
	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "acl-scan compat: at least one file path is required")
		return 2
	}

	report := scanner.ScanPaths(targets)

	switch format {
	case formatJSON:
		return emitJSON(report)
	default:
		return emitText(report)
	}
}

// ─── Sub-command: validate-compat ────────────────────────────────────────────

// runValidateCompat validates that all supplied binaries are classified and
// emits a structured validation result.  It exits non-zero when any binary
// cannot be classified or needs patching.
func runValidateCompat(targets []string, format outputFormat) int {
	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "acl-scan validate-compat: at least one file path is required")
		return 2
	}

	report := scanner.ScanPaths(targets)
	allOK := report.Summary.Errors == 0 &&
		report.Summary.Unknown == 0 &&
		report.Summary.NeedsPatch == 0

	switch format {
	case formatJSON:
		type validationResult struct {
			Report scanner.ScanReport `json:"report"`
			Valid  bool               `json:"valid"`
			Reason string             `json:"reason,omitempty"`
		}
		result := validationResult{
			Report: report,
			Valid:  allOK,
		}
		if !allOK {
			result.Reason = buildValidationReason(report)
		}
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "acl-scan: JSON marshal error: %v\n", err)
			return 3
		}
		fmt.Println(string(data))
	default:
		printTextReport(report)
		if !allOK {
			fmt.Fprintf(os.Stderr, "\nValidation FAILED: %s\n", buildValidationReason(report))
		} else {
			fmt.Println("\nValidation PASSED: all binaries are Android-compatible.")
		}
	}

	if !allOK {
		return 1
	}
	return 0
}

// ─── Emitters ─────────────────────────────────────────────────────────────────

func emitJSON(report scanner.ScanReport) int {
	data, err := scanner.MarshalReport(report)
	if err != nil {
		fmt.Fprintf(os.Stderr, "acl-scan: JSON marshal error: %v\n", err)
		return 3
	}
	fmt.Println(string(data))
	if needsAction(report) {
		return 1
	}
	return 0
}

func emitText(report scanner.ScanReport) int {
	printTextReport(report)
	if needsAction(report) {
		return 1
	}
	return 0
}

// printTextReport writes a human-readable compatibility report to stdout.
func printTextReport(report scanner.ScanReport) {
	fmt.Printf("ACL Compatibility Scan Report\n")
	fmt.Printf("Generated: %s\n", report.GeneratedAt)
	fmt.Printf("Schema:    %s\n\n", report.SchemaVersion)

	for _, b := range report.Binaries {
		fmt.Printf("File: %s\n", b.Path)
		fmt.Printf("  Category:    %s\n", b.CompatCategory)
		if b.ELF != nil {
			fmt.Printf("  ELF Class:   %s\n", b.ELF.Class)
			fmt.Printf("  Machine:     %s\n", b.ELF.Machine)
			if b.ELF.Interpreter != "" {
				fmt.Printf("  PT_INTERP:   %s\n", b.ELF.Interpreter)
			}
			if b.ELF.Rpath != "" {
				fmt.Printf("  RPATH:       %s\n", b.ELF.Rpath)
			}
			if b.ELF.Runpath != "" {
				fmt.Printf("  RUNPATH:     %s\n", b.ELF.Runpath)
			}
			if len(b.ELF.Needed) > 0 {
				fmt.Printf("  DT_NEEDED:   %s\n", strings.Join(b.ELF.Needed, ", "))
			}
		}
		if len(b.MissingSymbols) > 0 {
			fmt.Printf("  Missing (Bionic-incompatible):\n")
			for _, ms := range b.MissingSymbols {
				fmt.Printf("    - %s: %s\n", ms.Library, ms.Reason)
			}
		}
		fmt.Printf("  Recommendation: %s\n", b.Recommendation.Action)
		fmt.Printf("    %s\n", b.Recommendation.Rationale)
		if b.Recommendation.SuggestedInterpreter != "" {
			fmt.Printf("    Suggested interpreter: %s\n", b.Recommendation.SuggestedInterpreter)
		}
		if b.Recommendation.SuggestedRpath != "" {
			fmt.Printf("    Suggested RPATH: %s\n", b.Recommendation.SuggestedRpath)
		}
		if b.Error != "" {
			fmt.Printf("  ERROR: %s\n", b.Error)
		}
		fmt.Println()
	}

	s := report.Summary
	fmt.Printf("Summary: total=%d native=%d glibc=%d static=%d script=%d unknown=%d unsupported=%d errors=%d needs-patch=%d\n",
		s.Total, s.NativeAndroid, s.LinuxGlibc, s.Static, s.Script,
		s.Unknown, s.Unsupported, s.Errors, s.NeedsPatch)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// needsAction returns true when the report contains binaries that require
// patching, are unknown, or had inspection errors.
func needsAction(report scanner.ScanReport) bool {
	return report.Summary.NeedsPatch > 0 ||
		report.Summary.Unknown > 0 ||
		report.Summary.Errors > 0
}

func buildValidationReason(report scanner.ScanReport) string {
	var parts []string
	s := report.Summary
	if s.Errors > 0 {
		parts = append(parts, fmt.Sprintf("%d inspection error(s)", s.Errors))
	}
	if s.Unknown > 0 {
		parts = append(parts, fmt.Sprintf("%d unclassified binary/ies", s.Unknown))
	}
	if s.NeedsPatch > 0 {
		parts = append(parts, fmt.Sprintf("%d binary/ies need patching", s.NeedsPatch))
	}
	return strings.Join(parts, "; ")
}

// parseOutputFlag extracts --output <value> from args and returns the format
// and the remaining args.  It is lenient: an unrecognised --output value
// returns an error; an absent flag returns formatText.
func parseOutputFlag(args []string) (outputFormat, []string, error) {
	format := formatText
	remaining := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--output" || arg == "-output":
			if i+1 >= len(args) {
				return format, nil, fmt.Errorf("--output requires a value (json or text)")
			}
			i++
			val := strings.ToLower(args[i])
			switch val {
			case "json":
				format = formatJSON
			case "text":
				format = formatText
			default:
				return format, nil, fmt.Errorf("--output: unsupported value %q (want json or text)", args[i])
			}
		case strings.HasPrefix(arg, "--output="):
			val := strings.ToLower(strings.TrimPrefix(arg, "--output="))
			switch val {
			case "json":
				format = formatJSON
			case "text":
				format = formatText
			default:
				return format, nil, fmt.Errorf("--output: unsupported value %q (want json or text)", val)
			}
		default:
			remaining = append(remaining, arg)
		}
	}
	return format, remaining, nil
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage:
  acl-scan [--output json|text] compat            <file> [file...]
  acl-scan [--output json|text] compat-json       <file> [file...]   (always JSON)
  acl-scan [--output json|text] validate-compat   <file> [file...]
  acl-scan validate-compat-json                   <file> [file...]   (always JSON)

Output format:
  text  — human-readable report (default for compat / validate-compat)
  json  — machine-readable JSON report (always used for *-json sub-commands)

Exit codes:
  0 — all binaries are Android-compatible (no patching needed)
  1 — one or more binaries need patching, are unknown, or had errors
  2 — usage / argument error
  3 — internal error`)
}
