// Package tests contains ACL integration-style tests that exercise script
// fixture files from testdata/scripts/ through the scanner's shebang logic.
//
// Each test creates a minimal fake Termux PREFIX, calls the scanner's
// ScanShebang / AddScriptEntry helpers, and asserts the expected
// InterpreterStatus outcome.
package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arduino/arduino-cli/acl/scanner"
)

// fixtureDir is the path to the script fixtures relative to this file.
const fixtureDir = "testdata/scripts"

// buildFakePrefix creates a minimal Termux PREFIX tree under a temp directory,
// populating <prefix>/bin/ with the supplied binary names.
func buildFakePrefix(t *testing.T, binaries ...string) string {
	t.Helper()
	prefix := t.TempDir()
	binDir := filepath.Join(prefix, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", binDir, err)
	}
	for _, name := range binaries {
		p := filepath.Join(binDir, name)
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return prefix
}

// ─── bash ─────────────────────────────────────────────────────────────────────

func TestFixture_BashScript_Remapped(t *testing.T) {
	prefix := buildFakePrefix(t, "bash")
	script := filepath.Join(fixtureDir, "bash_script.sh")

	res, err := scanner.ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatalf("ScanShebang: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil ShebangResult for bash_script.sh")
	}
	if res.InterpreterPath != "/bin/bash" {
		t.Errorf("interpreter: got %q, want /bin/bash", res.InterpreterPath)
	}
	if res.Status != scanner.InterpreterRemapped {
		t.Errorf("status: got %q, want remapped", res.Status)
	}
	want := filepath.Join(prefix, "bin", "bash")
	if res.ResolvedPath != want {
		t.Errorf("resolved: got %q, want %q", res.ResolvedPath, want)
	}
}

func TestFixture_UsrBinBashScript_Remapped(t *testing.T) {
	prefix := buildFakePrefix(t, "bash")
	script := filepath.Join(fixtureDir, "usr_bin_bash_script.sh")

	res, err := scanner.ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatalf("ScanShebang: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil ShebangResult for usr_bin_bash_script.sh")
	}
	if res.InterpreterPath != "/usr/bin/bash" {
		t.Errorf("interpreter: got %q, want /usr/bin/bash", res.InterpreterPath)
	}
	if res.Status != scanner.InterpreterRemapped {
		t.Errorf("status: got %q, want remapped", res.Status)
	}
}

func TestFixture_EnvBashScript_Remapped(t *testing.T) {
	prefix := buildFakePrefix(t, "bash")
	script := filepath.Join(fixtureDir, "env_bash_script.sh")

	res, err := scanner.ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatalf("ScanShebang: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil ShebangResult for env_bash_script.sh")
	}
	if res.InterpreterPath != "/usr/bin/env" {
		t.Errorf("interpreter: got %q, want /usr/bin/env", res.InterpreterPath)
	}
	if len(res.InterpreterArgs) == 0 || res.InterpreterArgs[0] != "bash" {
		t.Errorf("args: got %v, want [bash]", res.InterpreterArgs)
	}
	if res.Status != scanner.InterpreterRemapped {
		t.Errorf("status: got %q, want remapped", res.Status)
	}
	want := filepath.Join(prefix, "bin", "bash")
	if res.ResolvedPath != want {
		t.Errorf("resolved: got %q, want %q", res.ResolvedPath, want)
	}
}

func TestFixture_BashScript_Missing(t *testing.T) {
	prefix := buildFakePrefix(t /* nothing */)
	script := filepath.Join(fixtureDir, "bash_script.sh")

	res, err := scanner.ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatalf("ScanShebang: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil ShebangResult")
	}
	if res.Status != scanner.InterpreterMissing {
		t.Errorf("status: got %q, want missing", res.Status)
	}
}

// ─── python3 ─────────────────────────────────────────────────────────────────

func TestFixture_Python3Script_Remapped(t *testing.T) {
	prefix := buildFakePrefix(t, "python3")
	script := filepath.Join(fixtureDir, "python3_script.py")

	res, err := scanner.ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatalf("ScanShebang: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil ShebangResult for python3_script.py")
	}
	if res.InterpreterPath != "/usr/bin/python3" {
		t.Errorf("interpreter: got %q, want /usr/bin/python3", res.InterpreterPath)
	}
	if res.Status != scanner.InterpreterRemapped {
		t.Errorf("status: got %q, want remapped", res.Status)
	}
	want := filepath.Join(prefix, "bin", "python3")
	if res.ResolvedPath != want {
		t.Errorf("resolved: got %q, want %q", res.ResolvedPath, want)
	}
}

func TestFixture_EnvPython3Script_Remapped(t *testing.T) {
	prefix := buildFakePrefix(t, "python3")
	script := filepath.Join(fixtureDir, "env_python3_script.py")

	res, err := scanner.ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatalf("ScanShebang: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil ShebangResult for env_python3_script.py")
	}
	if res.InterpreterPath != "/usr/bin/env" {
		t.Errorf("interpreter: got %q, want /usr/bin/env", res.InterpreterPath)
	}
	if len(res.InterpreterArgs) == 0 || res.InterpreterArgs[0] != "python3" {
		t.Errorf("args: got %v, want [python3]", res.InterpreterArgs)
	}
	if res.Status != scanner.InterpreterRemapped {
		t.Errorf("status: got %q, want remapped", res.Status)
	}
}

func TestFixture_Python3Script_Missing(t *testing.T) {
	prefix := buildFakePrefix(t /* nothing */)
	script := filepath.Join(fixtureDir, "python3_script.py")

	res, err := scanner.ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatalf("ScanShebang: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil ShebangResult")
	}
	if res.Status != scanner.InterpreterMissing {
		t.Errorf("status: got %q, want missing", res.Status)
	}
	if res.Recommendation == "" {
		t.Error("expected non-empty recommendation")
	}
}

func TestFixture_EnvPython3Script_Missing(t *testing.T) {
	prefix := buildFakePrefix(t /* nothing */)
	script := filepath.Join(fixtureDir, "env_python3_script.py")

	res, err := scanner.ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatalf("ScanShebang: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil ShebangResult")
	}
	if res.Status != scanner.InterpreterMissing {
		t.Errorf("status: got %q, want missing", res.Status)
	}
}

// ─── perl ─────────────────────────────────────────────────────────────────────

func TestFixture_PerlScript_Remapped(t *testing.T) {
	prefix := buildFakePrefix(t, "perl")
	script := filepath.Join(fixtureDir, "perl_script.pl")

	res, err := scanner.ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatalf("ScanShebang: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil ShebangResult for perl_script.pl")
	}
	if res.InterpreterPath != "/usr/bin/perl" {
		t.Errorf("interpreter: got %q, want /usr/bin/perl", res.InterpreterPath)
	}
	if res.Status != scanner.InterpreterRemapped {
		t.Errorf("status: got %q, want remapped", res.Status)
	}
	want := filepath.Join(prefix, "bin", "perl")
	if res.ResolvedPath != want {
		t.Errorf("resolved: got %q, want %q", res.ResolvedPath, want)
	}
}

func TestFixture_EnvPerlScript_Remapped(t *testing.T) {
	prefix := buildFakePrefix(t, "perl")
	script := filepath.Join(fixtureDir, "env_perl_script.pl")

	res, err := scanner.ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatalf("ScanShebang: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil ShebangResult for env_perl_script.pl")
	}
	if res.InterpreterPath != "/usr/bin/env" {
		t.Errorf("interpreter: got %q, want /usr/bin/env", res.InterpreterPath)
	}
	if len(res.InterpreterArgs) == 0 || res.InterpreterArgs[0] != "perl" {
		t.Errorf("args: got %v, want [perl]", res.InterpreterArgs)
	}
	if res.Status != scanner.InterpreterRemapped {
		t.Errorf("status: got %q, want remapped", res.Status)
	}
}

func TestFixture_PerlScript_Missing(t *testing.T) {
	prefix := buildFakePrefix(t /* nothing */)
	script := filepath.Join(fixtureDir, "perl_script.pl")

	res, err := scanner.ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatalf("ScanShebang: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil ShebangResult")
	}
	if res.Status != scanner.InterpreterMissing {
		t.Errorf("status: got %q, want missing", res.Status)
	}
}

// ─── env delegation edge cases ───────────────────────────────────────────────

func TestFixture_EnvUnknown_Missing(t *testing.T) {
	// env_unknown_script.sh uses "notarealinterpreter" which is neither in
	// the well-known map nor installed in the fake prefix.
	prefix := buildFakePrefix(t /* nothing */)
	script := filepath.Join(fixtureDir, "env_unknown_script.sh")

	res, err := scanner.ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatalf("ScanShebang: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil ShebangResult")
	}
	if res.Status != scanner.InterpreterMissing {
		t.Errorf("status: got %q, want missing", res.Status)
	}
}

func TestFixture_EnvUnknown_PresentInPrefix_Remapped(t *testing.T) {
	// Simulate someone installing "notarealinterpreter" under the prefix.
	prefix := buildFakePrefix(t, "notarealinterpreter")
	script := filepath.Join(fixtureDir, "env_unknown_script.sh")

	res, err := scanner.ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatalf("ScanShebang: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil ShebangResult")
	}
	if res.Status != scanner.InterpreterRemapped {
		t.Errorf("status: got %q, want remapped", res.Status)
	}
	want := filepath.Join(prefix, "bin", "notarealinterpreter")
	if res.ResolvedPath != want {
		t.Errorf("resolved: got %q, want %q", res.ResolvedPath, want)
	}
}

// ─── no shebang ───────────────────────────────────────────────────────────────

func TestFixture_NoShebangScript_NilResult(t *testing.T) {
	prefix := buildFakePrefix(t, "sh")
	script := filepath.Join(fixtureDir, "no_shebang_script.sh")

	res, err := scanner.ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatalf("ScanShebang: %v", err)
	}
	if res != nil {
		t.Errorf("expected nil result for file with no shebang, got %+v", res)
	}
}

// ─── ReportBuilder end-to-end with fixtures ───────────────────────────────────

func TestReportBuilder_FixtureDir_AllScripts(t *testing.T) {
	// Build a prefix with all interpreters referenced by the fixture scripts.
	prefix := buildFakePrefix(t, "bash", "python3", "perl", "env")

	b := scanner.NewReportBuilder(fixtureDir, prefix, "")

	scripts := []string{
		"bash_script.sh",
		"usr_bin_bash_script.sh",
		"env_bash_script.sh",
		"python3_script.py",
		"env_python3_script.py",
		"perl_script.pl",
		"env_perl_script.pl",
		"no_shebang_script.sh",
	}

	for _, name := range scripts {
		p := filepath.Join(fixtureDir, name)
		if err := b.AddScriptEntry(p); err != nil {
			t.Fatalf("AddScriptEntry(%s): %v", name, err)
		}
	}

	r := b.Build()

	if r.Summary.Total != len(scripts) {
		t.Errorf("total: got %d, want %d", r.Summary.Total, len(scripts))
	}
	if r.Summary.Script != len(scripts) {
		t.Errorf("script: got %d, want %d", r.Summary.Script, len(scripts))
	}

	// env_unknown_script.sh is NOT in this list, so all known interpreters
	// should be remapped (7 scripts have shebangs, 1 has none).
	// bash_script, usr_bin_bash_script, env_bash → 3× bash remapped
	// python3_script, env_python3_script → 2× python3 remapped
	// perl_script, env_perl_script → 2× perl remapped
	// no_shebang_script → no interpreter_status
	if r.Summary.ScriptRemapped != 7 {
		t.Errorf("script_interpreter_remapped: got %d, want 7", r.Summary.ScriptRemapped)
	}
	if r.Summary.ScriptMissing != 0 {
		t.Errorf("script_interpreter_missing: got %d, want 0", r.Summary.ScriptMissing)
	}
	if r.Summary.ScriptFound != 0 {
		t.Errorf("script_interpreter_found: got %d, want 0 (fake prefix, not literal match)", r.Summary.ScriptFound)
	}
}

func TestReportBuilder_FixtureDir_MissingInterpreters(t *testing.T) {
	// Empty PREFIX — all shebangs should be missing.
	prefix := buildFakePrefix(t)

	b := scanner.NewReportBuilder(fixtureDir, prefix, "")

	scripts := []string{
		"bash_script.sh",
		"python3_script.py",
		"perl_script.pl",
	}
	for _, name := range scripts {
		p := filepath.Join(fixtureDir, name)
		if err := b.AddScriptEntry(p); err != nil {
			t.Fatalf("AddScriptEntry(%s): %v", name, err)
		}
	}

	r := b.Build()

	if r.Summary.ScriptMissing != 3 {
		t.Errorf("script_interpreter_missing: got %d, want 3", r.Summary.ScriptMissing)
	}
}
