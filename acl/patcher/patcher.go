// Package patcher exposes the top-level Apply and DryRun entry points used by
// acl-exec and the patcher CLI.
//
// Apply   — compute plans then call patchelf to rewrite binaries.
// DryRun  — compute plans and write a diff-style report to an io.Writer
//            without modifying any file.
package patcher

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// DryRunOptions extends PlanOptions with display preferences.
type DryRunOptions struct {
	PlanOptions

	// Out is the writer that receives the diff report.  Defaults to os.Stdout
	// when nil.
	Out io.Writer

	// Color enables ANSI colour codes in the output.
	Color bool

	// Verbose prints a status line for every file, including those that need
	// no patching and skipped entries.
	Verbose bool
}

// DryRun computes patch plans for each path in targets and writes the
// diff-style plan to opts.Out (or os.Stdout when nil).
// No files are modified.
// It returns an error only if plan computation or writing fails; it is NOT an
// error if some binaries need patching.
func DryRun(targets []string, opts DryRunOptions) error {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}

	plans := ComputePlans(targets, opts.PlanOptions)
	return WritePlan(out, plans, FormatOptions{
		Color:   opts.Color,
		Verbose: opts.Verbose,
	})
}

// ApplyOptions extends PlanOptions with apply-phase settings.
type ApplyOptions struct {
	PlanOptions

	// DryRun, when true, skips the patchelf calls and only prints the plan.
	DryRun bool

	// Out is the writer for status output.  Defaults to os.Stdout.
	Out io.Writer

	// Color enables ANSI colour codes in status output.
	Color bool

	// Verbose prints status for files that need no patching.
	Verbose bool
}

// ApplyResult summarises an apply run.
type ApplyResult struct {
	Plans   []PatchPlan
	Applied int
	Skipped int
	Errors  []error
}

// Apply computes plans for targets, optionally prints the diff plan, then
// calls patchelf to apply each edit (unless DryRun is true).
func Apply(targets []string, opts ApplyOptions) (ApplyResult, error) {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}

	plans := ComputePlans(targets, opts.PlanOptions)

	// Always render the plan so operators can see what would happen.
	if err := WritePlan(out, plans, FormatOptions{
		Color:   opts.Color,
		Verbose: opts.Verbose,
	}); err != nil {
		return ApplyResult{Plans: plans}, fmt.Errorf("render patch plan: %w", err)
	}

	if opts.DryRun {
		return ApplyResult{Plans: plans}, nil
	}

	result := ApplyResult{Plans: plans}

	for _, plan := range plans {
		if plan.Skipped || !plan.NeedsPatching() {
			result.Skipped++
			continue
		}
		if err := applyPlan(plan, opts.PlanOptions); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", plan.Path, err))
		} else {
			result.Applied++
		}
	}

	if len(result.Errors) > 0 {
		return result, fmt.Errorf("%d patch(es) failed", len(result.Errors))
	}
	return result, nil
}

// applyPlan calls patchelf to execute the edits in a single plan.
func applyPlan(plan PatchPlan, opts PlanOptions) error {
	if err := requirePatchelf(); err != nil {
		return err
	}

	var interpValue string
	var rpathValue string

	for _, edit := range plan.Edits {
		switch edit.Field {
		case "PT_INTERP":
			interpValue = edit.Proposed
		case "RUNPATH", "RPATH":
			rpathValue = edit.Proposed
		}
	}

	// Build the patchelf argument list.  We apply interpreter and rpath in a
	// single patchelf invocation when possible to minimise the number of ELF
	// rewrites (each rewrite expands the file slightly).
	args := []string{}
	if interpValue != "" {
		args = append(args, "--set-interpreter", interpValue)
	}
	if rpathValue != "" {
		args = append(args, "--set-rpath", rpathValue)
	} else if hasRPATHEdit(plan) {
		// Explicit RPATH removal: set empty RUNPATH.
		args = append(args, "--remove-rpath")
	}

	if len(args) == 0 {
		return nil
	}

	args = append(args, plan.Path)
	cmd := exec.Command("patchelf", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("patchelf %v: %w\n%s", args, err, out)
	}
	return nil
}

// hasRPATHEdit returns true when plan contains an edit that removes RPATH.
func hasRPATHEdit(plan PatchPlan) bool {
	for _, e := range plan.Edits {
		if e.Field == "RPATH" && e.Proposed == "" {
			return true
		}
	}
	return false
}

// requirePatchelf verifies that the patchelf binary is available on PATH.
func requirePatchelf() error {
	if _, err := exec.LookPath("patchelf"); err != nil {
		return fmt.Errorf("patchelf not found on PATH — install it before applying patches")
	}
	return nil
}
