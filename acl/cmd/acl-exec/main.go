// acl-exec is the ACL execution planner.
//
// It calculates the full set of ELF patches required to run a tool under the
// ACL runtime and either prints a dry-run diff plan or (with --apply) calls
// patchelf to rewrite the binaries.
//
// Usage:
//
//	acl-exec --dry-run  [--runtime <dir>] [--loader <name>] [--verbose] [--color] <file-or-dir> [...]
//	acl-exec --apply    [--runtime <dir>] [--loader <name>] [--verbose] [--color] <file-or-dir> [...]
//
// Flags:
//
//	--dry-run          Compute and print the patch plan without modifying files.
//	                   This is the default when neither --dry-run nor --apply is given.
//	--apply            Apply the patches using patchelf.  Implies plan output first.
//	--runtime <dir>    Path to the ACL runtime directory (loader + libraries).
//	                   Defaults to $ACL_RUNTIME_DIR or ./runtime.
//	--loader  <name>   Basename of the dynamic linker inside --runtime,
//	                   e.g. "ld-linux-aarch64.so.1".  When empty the default for
//	                   aarch64 is used.
//	--verbose          Print a status line for every file, including already-OK
//	                   and skipped entries.
//	--color            Enable ANSI colour codes in output.
//	                   Auto-detected from $NO_COLOR and terminal state when absent.
//	--help             Show this help.
//
// Exit codes:
//
//	0  Success (dry-run: plan printed; apply: all patches applied).
//	1  One or more patches failed during --apply.
//	2  Usage error.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/arduino/arduino-cli/acl/patcher"
)

const usageText = `acl-exec — ACL execution planner and ELF patcher

Usage:
  acl-exec [--dry-run] [options] <file-or-dir> [...]
  acl-exec --apply    [options] <file-or-dir> [...]

Options:
  --dry-run          Compute and print the patch plan (default when --apply is
                     not given).  No files are modified.
  --apply            Apply patches using patchelf after printing the plan.
  --runtime <dir>    ACL runtime directory containing the loader and libraries.
                     Default: $ACL_RUNTIME_DIR or ./runtime
  --loader  <name>   Basename of the dynamic linker inside --runtime.
                     Default: ld-linux-aarch64.so.1
  --verbose          Show a status line for every file (including already-OK
                     and skipped entries).
  --color            Force ANSI colour output.
  --no-color         Force plain-text output.
  --help             Show this help and exit.

Examples:
  # Preview patches for a GCC toolchain (no files modified):
  acl-exec --dry-run --runtime /data/acl/runtime /opt/gcc/bin/

  # Apply patches:
  acl-exec --apply --runtime /data/acl/runtime /opt/gcc/bin/gcc

Exit codes:
  0  Plan printed (dry-run) or all patches applied successfully.
  1  One or more patches failed.
  2  Usage error.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the testable entry point.  It returns the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("acl-exec", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		dryRunFlag  = fs.Bool("dry-run", false, "compute and print patch plan without modifying files (default)")
		applyFlag   = fs.Bool("apply", false, "apply patches using patchelf")
		runtimeFlag = fs.String("runtime", "", "ACL runtime directory")
		loaderFlag  = fs.String("loader", "", "basename of the dynamic linker inside --runtime")
		verboseFlag = fs.Bool("verbose", false, "print status for every file")
		colorFlag   = fs.Bool("color", false, "force ANSI colour output")
		noColorFlag = fs.Bool("no-color", false, "force plain-text output")
		helpFlag    = fs.Bool("help", false, "show help")
	)

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usageText)
			return 0
		}
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	if *helpFlag {
		fmt.Fprint(stdout, usageText)
		return 0
	}

	// Resolve targets from remaining positional arguments.
	targets := fs.Args()
	if len(targets) == 0 {
		fmt.Fprintf(stderr, "error: at least one <file-or-dir> argument is required\n\n")
		fmt.Fprint(stderr, usageText)
		return 2
	}

	// Expand directories to ELF file paths; pass individual files through as-is.
	expanded, err := expandTargets(targets)
	if err != nil {
		fmt.Fprintf(stderr, "error expanding targets: %v\n", err)
		return 2
	}
	if len(expanded) == 0 {
		fmt.Fprintln(stderr, "warning: no ELF files found in the given targets")
		return 0
	}

	// Resolve runtime directory: flag → env → default.
	runtimeDir := *runtimeFlag
	if runtimeDir == "" {
		runtimeDir = os.Getenv("ACL_RUNTIME_DIR")
	}
	if runtimeDir == "" {
		runtimeDir = "runtime"
	}

	// Colour auto-detection:
	//   1. $NO_COLOR env var (https://no-color.org/) disables colour.
	//   2. --no-color flag disables colour.
	//   3. --color flag forces colour.
	//   4. Otherwise: enable when stdout looks like a terminal.
	useColor := isTerminal(stdout)
	if _, noColorEnv := os.LookupEnv("NO_COLOR"); noColorEnv {
		useColor = false
	}
	if *noColorFlag {
		useColor = false
	}
	if *colorFlag {
		useColor = true
	}

	planOpts := patcher.PlanOptions{
		RuntimeDir: runtimeDir,
		LoaderName: *loaderFlag,
	}

	// Default to dry-run unless --apply is explicitly given.
	// --dry-run is accepted as an explicit marker but has no additional effect
	// beyond the default behaviour.
	_ = *dryRunFlag
	doApply := *applyFlag

	if !doApply {
		// Dry-run: compute and render the plan without modifying any file.
		err := patcher.DryRun(expanded, patcher.DryRunOptions{
			PlanOptions: planOpts,
			Out:         stdout,
			Color:       useColor,
			Verbose:     *verboseFlag,
		})
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		return 0
	}

	// Apply mode: compute plans, print the diff, then call patchelf.
	result, err := patcher.Apply(expanded, patcher.ApplyOptions{
		PlanOptions: planOpts,
		DryRun:      false,
		Out:         stdout,
		Color:       useColor,
		Verbose:     *verboseFlag,
	})
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		for _, e := range result.Errors {
			fmt.Fprintf(stderr, "  - %v\n", e)
		}
		return 1
	}
	return 0
}

// expandTargets resolves a mixed list of files and directories into a flat
// list of candidate paths.  Individual file arguments are passed through
// as-is (ComputePlan will skip non-ELF files gracefully).  Directory
// arguments are walked recursively; only files with the ELF magic header are
// returned.
func expandTargets(targets []string) ([]string, error) {
	var out []string
	for _, t := range targets {
		info, err := os.Stat(t)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", t, err)
		}
		if info.IsDir() {
			elfs, err := patcher.CollectELFPaths(t)
			if err != nil {
				return nil, fmt.Errorf("walk %s: %w", t, err)
			}
			out = append(out, elfs...)
		} else {
			out = append(out, t)
		}
	}
	return out, nil
}

// isTerminal is a best-effort check for whether w is connected to a Unix
// terminal.  It never errors — on failure it returns false (plain-text mode).
func isTerminal(w io.Writer) bool {
	type fder interface{ Fd() uintptr }
	if _, ok := w.(fder); !ok {
		return false
	}
	// Use os.Stdout.Stat() as a terminal detection proxy.  This avoids pulling
	// in golang.org/x/term as a dependency for a single heuristic.
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
