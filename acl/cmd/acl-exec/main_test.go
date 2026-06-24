package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ──────────────────────────────────────────────────────────────────────────────
// Helpers shared by tests
// ──────────────────────────────────────────────────────────────────────────────

func capture(args []string) (stdout, stderr string, code int) {
	var outBuf, errBuf bytes.Buffer
	code = run(args, &outBuf, &errBuf)
	return outBuf.String(), errBuf.String(), code
}

// writeScript writes a small shell script so we have a non-ELF file in the
// fixture directory.
func writeScript(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatalf("writeScript: %v", err)
	}
	return p
}

// ──────────────────────────────────────────────────────────────────────────────
// Unit tests for run()
// ──────────────────────────────────────────────────────────────────────────────

func TestRun_NoArgs_ExitsTwo(t *testing.T) {
	_, _, code := capture([]string{})
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestRun_Help(t *testing.T) {
	stdout, _, code := capture([]string{"--help"})
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "acl-exec") {
		t.Error("help output should mention acl-exec")
	}
	if !strings.Contains(stdout, "--dry-run") {
		t.Error("help output should mention --dry-run")
	}
	if !strings.Contains(stdout, "--apply") {
		t.Error("help output should mention --apply")
	}
	if !strings.Contains(stdout, "--runtime") {
		t.Error("help output should mention --runtime")
	}
}

func TestRun_DryRun_NonELFScript_ExitsZero(t *testing.T) {
	dir := t.TempDir()
	script := writeScript(t, dir, "run.sh")

	stdout, stderr, code := capture([]string{
		"--dry-run",
		"--runtime", "/acl/runtime",
		script,
	})
	if code != 0 {
		t.Errorf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	// The script should appear in the summary (as skipped).
	if !strings.Contains(stdout, "Dry-run summary") {
		t.Error("expected Dry-run summary in output")
	}
}

func TestRun_DryRun_MissingFile_ExitsTwo(t *testing.T) {
	_, _, code := capture([]string{
		"--dry-run",
		"--runtime", "/acl/runtime",
		"/does/not/exist/binary",
	})
	if code != 2 {
		t.Errorf("exit code = %d, want 2 for missing file", code)
	}
}

func TestRun_DryRun_Directory_ExpandsELFs(t *testing.T) {
	dir := t.TempDir()
	// Plant a couple of non-ELF files — they will be collected by
	// CollectELFPaths only if they have the ELF magic bytes.
	writeScript(t, dir, "tool.sh")
	// Also write a minimal ELF-magic file so CollectELFPaths returns something.
	elfMagic := []byte{0x7f, 'E', 'L', 'F', 0x02, 0x01, 0x01, 0x00}
	elfPath := filepath.Join(dir, "bin.elf")
	if err := os.WriteFile(elfPath, elfMagic, 0o755); err != nil {
		t.Fatalf("write elf magic: %v", err)
	}

	stdout, stderr, code := capture([]string{
		"--dry-run",
		"--runtime", "/acl/runtime",
		dir,
	})
	// We expect exit 0; the ELF magic file will fail full ELF parsing
	// (too short) and be marked skipped, which is fine.
	if code != 0 {
		t.Errorf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Dry-run summary") {
		t.Error("expected Dry-run summary in output")
	}
}

func TestRun_DefaultIsDryRun(t *testing.T) {
	dir := t.TempDir()
	script := writeScript(t, dir, "run.sh")

	// Running without --dry-run or --apply should behave like --dry-run.
	stdout, _, code := capture([]string{
		"--runtime", "/acl/runtime",
		script,
	})
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Dry-run summary") {
		t.Errorf("default mode should produce dry-run output; got: %s", stdout)
	}
}

func TestRun_VerboseFlag_ShowsSkipped(t *testing.T) {
	dir := t.TempDir()
	script := writeScript(t, dir, "run.sh")

	stdout, _, code := capture([]string{
		"--dry-run",
		"--verbose",
		"--runtime", "/acl/runtime",
		script,
	})
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "[skipped]") {
		t.Error("verbose mode should show [skipped] entries")
	}
}

func TestRun_NoColorFlag_NoANSISequences(t *testing.T) {
	dir := t.TempDir()
	script := writeScript(t, dir, "run.sh")

	stdout, _, _ := capture([]string{
		"--dry-run",
		"--no-color",
		"--verbose",
		"--runtime", "/acl/runtime",
		script,
	})
	if strings.Contains(stdout, "\033[") {
		t.Error("--no-color should suppress all ANSI escape sequences")
	}
}

func TestRun_UnknownFlag_ExitsTwo(t *testing.T) {
	dir := t.TempDir()
	script := writeScript(t, dir, "run.sh")

	_, _, code := capture([]string{
		"--this-flag-does-not-exist",
		script,
	})
	if code != 2 {
		t.Errorf("exit code = %d, want 2 for unknown flag", code)
	}
}

func TestRun_RuntimeEnvVar(t *testing.T) {
	dir := t.TempDir()
	script := writeScript(t, dir, "run.sh")

	t.Setenv("ACL_RUNTIME_DIR", "/env/runtime")

	stdout, _, code := capture([]string{
		"--dry-run",
		"--verbose",
		script,
	})
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// The plan is computed with the env-provided runtime.  The script is
	// skipped, so its SkipReason appears in verbose output.  We just need
	// to confirm the command succeeded.
	_ = stdout
}

func TestRun_EmptyDirectory_ExitsZero(t *testing.T) {
	dir := t.TempDir()
	// An empty directory has no ELF files; CollectELFPaths returns nil.
	stdout, stderr, code := capture([]string{
		"--dry-run",
		"--runtime", "/acl/runtime",
		dir,
	})
	if code != 0 {
		t.Errorf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	// With 0 files the warning path is taken; check for graceful exit.
	_ = stdout
}

// TestExpandTargets_FileAndDir exercises expandTargets with mixed inputs.
func TestExpandTargets_FileAndDir(t *testing.T) {
	dir := t.TempDir()
	// ELF magic file.
	elfPath := filepath.Join(dir, "binary")
	if err := os.WriteFile(elfPath, []byte{0x7f, 'E', 'L', 'F', 0x02}, 0o755); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Script (not ELF).
	scriptPath := writeScript(t, dir, "run.sh")

	subDir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	subELF := filepath.Join(subDir, "subbinary")
	if err := os.WriteFile(subELF, []byte{0x7f, 'E', 'L', 'F', 0x02}, 0o755); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Pass the individual script file + directory.
	expanded, err := expandTargets([]string{scriptPath, subDir})
	if err != nil {
		t.Fatalf("expandTargets: %v", err)
	}
	// script is included as-is (ComputePlan will skip it).
	if len(expanded) < 2 {
		t.Errorf("expected ≥2 entries (script + sub ELF), got %d", len(expanded))
	}
	_ = elfPath // not passed directly, should not appear
}
