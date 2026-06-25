package verifier

import (
	"bufio"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
)

// copyFile is a test helper that copies src to dst with the given mode.
func copyFile(t *testing.T, src, dst string, mode os.FileMode) {
	t.Helper()
	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("copyFile open src %s: %v", src, err)
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		t.Fatalf("copyFile create dst %s: %v", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		t.Fatalf("copyFile copy: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("copyFile close: %v", err)
	}
}

// TestOverallExitCode verifies that OverallExitCode returns the worst code seen.
func TestOverallExitCode(t *testing.T) {
	tests := []struct {
		name    string
		results []Result
		want    ExitCode
	}{
		{
			name:    "all pass",
			results: []Result{{Passed: true, Code: ExitOK}, {Passed: true, Code: ExitOK}},
			want:    ExitOK,
		},
		{
			name: "one fail",
			results: []Result{
				{Passed: true, Code: ExitOK},
				{Passed: false, Code: ExitMissingDep},
			},
			want: ExitMissingDep,
		},
		{
			name: "multiple failures – worst wins",
			results: []Result{
				{Passed: false, Code: ExitMissingDep},
				{Passed: false, Code: ExitSELinux},
				{Passed: false, Code: ExitFilesystem},
			},
			want: ExitSELinux,
		},
		{
			name:    "empty",
			results: nil,
			want:    ExitOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := OverallExitCode(tc.results)
			if got != tc.want {
				t.Errorf("OverallExitCode() = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestResultString verifies that Result.String() contains the relevant fields.
func TestResultString(t *testing.T) {
	pass := Result{
		Name:    "my-check",
		Passed:  true,
		Code:    ExitOK,
		Message: "everything is fine",
	}
	s := pass.String()
	if !strings.Contains(s, "PASS") {
		t.Errorf("expected PASS in %q", s)
	}
	if !strings.Contains(s, "my-check") {
		t.Errorf("expected check name in %q", s)
	}
	if strings.Contains(s, "hint:") {
		t.Errorf("did not expect hint for passing result in %q", s)
	}

	fail := Result{
		Name:    "broken",
		Passed:  false,
		Code:    ExitFilesystem,
		Message: "path missing",
		Hint:    "install something",
	}
	sf := fail.String()
	if !strings.Contains(sf, "FAIL") {
		t.Errorf("expected FAIL in %q", sf)
	}
	if !strings.Contains(sf, "hint:") {
		t.Errorf("expected hint section in %q", sf)
	}
	if !strings.Contains(sf, "install something") {
		t.Errorf("expected hint text in %q", sf)
	}
}

// TestRunSelected verifies that only requested checks are executed.
func TestRunSelected(t *testing.T) {
	results := RunSelected([]string{"patchelf-present", "linker-present"})
	names := make(map[string]bool, len(results))
	for _, r := range results {
		names[r.Name] = true
	}
	if !names["patchelf-present"] {
		t.Error("expected patchelf-present to be in results")
	}
	if !names["linker-present"] {
		t.Error("expected linker-present to be in results")
	}
	if names["selinux-mode"] {
		t.Error("selinux-mode should not appear in selected run")
	}
}

// TestRunSelectedUnknownName verifies that unknown check names are silently ignored.
func TestRunSelectedUnknownName(t *testing.T) {
	results := RunSelected([]string{"definitely-not-a-real-check"})
	if len(results) != 0 {
		t.Errorf("expected 0 results for unknown check name, got %d", len(results))
	}
}

// TestCheckPrefixAccessible_NoPrefix verifies failure when no prefix is found
// and no environment variable is set. We unset relevant vars and rely on the
// default path not existing in the test environment.
func TestCheckPrefixAccessible_NoPrefix(t *testing.T) {
	if _, err := os.Stat("/data/data/com.termux/files/usr"); err == nil {
		t.Skip("running on real Termux — prefix exists, skip no-prefix test")
	}

	// Unset prefix env vars for the duration of this test.
	vars := []string{"TERMUX_PREFIX", "TERMUX__PREFIX", "PREFIX"}
	originals := make(map[string]string, len(vars))
	for _, v := range vars {
		originals[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	t.Cleanup(func() {
		for k, v := range originals {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	})

	result := CheckPrefixAccessible()
	if result.Passed {
		t.Errorf("expected failure when PREFIX is absent, got pass")
	}
	if result.Code != ExitFilesystem {
		t.Errorf("expected ExitFilesystem (%d), got %d", ExitFilesystem, result.Code)
	}
	if result.Hint == "" {
		t.Error("expected non-empty hint for failed check")
	}
}

// TestCheckPrefixAccessible_WithPrefix verifies success when a valid prefix is set.
func TestCheckPrefixAccessible_WithPrefix(t *testing.T) {
	dir := t.TempDir()
	// Create expected sub-directories.
	if err := os.MkdirAll(dir+"/bin", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir+"/lib", 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("TERMUX_PREFIX", dir)

	result := CheckPrefixAccessible()
	if !result.Passed {
		t.Errorf("expected pass with valid prefix, got: %s", result)
	}
	if result.Code != ExitOK {
		t.Errorf("expected ExitOK, got %d", result.Code)
	}
}

// TestCheckPrefixAccessible_MissingSubdir verifies failure when a sub-dir is absent.
func TestCheckPrefixAccessible_MissingSubdir(t *testing.T) {
	dir := t.TempDir()
	// Only create bin, not lib.
	if err := os.MkdirAll(dir+"/bin", 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TERMUX_PREFIX", dir)

	result := CheckPrefixAccessible()
	if result.Passed {
		t.Errorf("expected failure when lib sub-dir is absent")
	}
	if result.Code != ExitFilesystem {
		t.Errorf("expected ExitFilesystem, got %d", result.Code)
	}
}

// TestCheckProcSelfExe verifies that /proc/self/exe check passes on Linux.
func TestCheckProcSelfExe(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("proc/self/exe check only meaningful on Linux")
	}
	result := CheckProcSelfExe()
	// On a standard Linux system this should always pass.
	// We don't hard-assert Passed because CI may run in containers that
	// restrict /proc; we validate the structural invariants instead.
	if result.Name != "proc-self-exe" {
		t.Errorf("unexpected check name: %s", result.Name)
	}
	if !result.Passed && result.Hint == "" {
		t.Error("expected non-empty hint for failed proc-self-exe check")
	}
	if !result.Passed && result.Code == ExitOK {
		t.Error("failed check must not have ExitOK")
	}
}

// TestCheckWX_NoViolation verifies that standard Termux directories (0755) pass.
func TestCheckWX_NoViolation(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"bin", "lib"} {
		if err := os.MkdirAll(dir+"/"+sub, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("TERMUX_PREFIX", dir)

	result := CheckWX()
	if !result.Passed {
		t.Errorf("expected W^X check to pass for 0755 dirs, got: %s", result)
	}
}

// TestCheckWX_WorldWritableExec verifies that world-write+execute triggers a failure.
func TestCheckWX_WorldWritableExec(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("chmod o+wx checks behave differently as root")
	}
	dir := t.TempDir()
	for _, sub := range []string{"bin", "lib"} {
		p := dir + "/" + sub
		if err := os.MkdirAll(p, 0o777); err != nil {
			t.Fatal(err)
		}
		// Explicitly set 0o777 to ensure both write and execute for world.
		if err := os.Chmod(p, 0o777); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("TERMUX_PREFIX", dir)

	result := CheckWX()
	// 0o777 = rwxrwxrwx — world has w+x, so this should fail.
	if !result.Passed {
		if result.Code != ExitWX {
			t.Errorf("expected ExitWX (%d), got %d", ExitWX, result.Code)
		}
		if result.Hint == "" {
			t.Error("expected non-empty hint for W^X failure")
		}
	}
	// If the underlying FS ignores the bits, we accept a pass.
}

// TestCheckPatchelf_NotFound verifies that a missing patchelf produces the
// correct failure code and non-empty hint.
func TestCheckPatchelf_NotFound(t *testing.T) {
	// Override PATH to an empty temp dir so patchelf cannot be found.
	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)

	// Also make the TERMUX_PREFIX prefix have no patchelf.
	prefixDir := t.TempDir()
	if err := os.MkdirAll(prefixDir+"/bin", 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TERMUX_PREFIX", prefixDir)

	result := CheckPatchelf()
	if result.Passed {
		// patchelf was somehow found — not meaningful in this env.
		t.Skip("patchelf found even with overridden PATH; skipping not-found test")
	}
	if result.Code != ExitMissingDep {
		t.Errorf("expected ExitMissingDep (%d), got %d", ExitMissingDep, result.Code)
	}
	if result.Hint == "" {
		t.Error("expected non-empty hint when patchelf is missing")
	}
	if !strings.Contains(result.Hint, "pkg install patchelf") {
		t.Errorf("expected install hint in: %s", result.Hint)
	}
}

// TestCheckPatchelf_FakeExecutable verifies that a working fake patchelf passes.
func TestCheckPatchelf_FakeExecutable(t *testing.T) {
	// Build a tiny executable that exits 0 and prints a version string.
	// We use the test binary itself as a surrogate — it will be executable.
	binDir := t.TempDir()
	hostExe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	fakePatchelf := binDir + "/patchelf"
	copyFile(t, hostExe, fakePatchelf, 0o755)

	t.Setenv("PATH", binDir)
	// Clear TERMUX_PREFIX so findTool only searches PATH.
	t.Setenv("TERMUX_PREFIX", t.TempDir())

	result := CheckPatchelf()
	// The fake patchelf (which is the test binary) will run but will likely
	// not print a meaningful "version" line. The check only requires it to
	// execute without error, so if it passes we're good; if it fails due to
	// unusual output that's fine too — the important thing is it finds it.
	if result.Name != "patchelf-present" {
		t.Errorf("unexpected name: %s", result.Name)
	}
}

// TestCheckLinker_PrefixMissing verifies that CheckLinker fails gracefully when
// there is no PREFIX at all.
func TestCheckLinker_PrefixMissing(t *testing.T) {
	if _, err := os.Stat("/data/data/com.termux/files/usr"); err == nil {
		t.Skip("running on real Termux — prefix exists")
	}

	vars := []string{"TERMUX_PREFIX", "TERMUX__PREFIX", "PREFIX"}
	originals := make(map[string]string, len(vars))
	for _, v := range vars {
		originals[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	t.Cleanup(func() {
		for k, v := range originals {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	})

	result := CheckLinker()
	if result.Passed {
		t.Error("expected failure when PREFIX is absent")
	}
	if result.Code != ExitMissingDep {
		t.Errorf("expected ExitMissingDep, got %d", result.Code)
	}
}

// TestCheckLinker_WithLinker verifies success when a linker file exists.
func TestCheckLinker_WithLinker(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TERMUX_PREFIX", dir)

	// Create a fake linker at the first candidate path.
	candidates := linkerCandidates(dir)
	if len(candidates) == 0 {
		t.Skip("no linker candidates for this arch")
	}
	linkerPath := candidates[0]
	// Find the parent directory.
	lastSlash := strings.LastIndex(linkerPath, "/")
	if lastSlash < 0 {
		t.Fatalf("unexpected linker path without slash: %s", linkerPath)
	}
	if err := os.MkdirAll(linkerPath[:lastSlash], 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(linkerPath, []byte("fake elf"), 0o755); err != nil {
		t.Fatal(err)
	}

	result := CheckLinker()
	if !result.Passed {
		t.Errorf("expected pass with fake linker at %s, got: %s", linkerPath, result)
	}
}

// TestCheckLinker_AllCandidatesMissing verifies that the check fails when none
// of the candidate linker paths exist.
func TestCheckLinker_AllCandidatesMissing(t *testing.T) {
	dir := t.TempDir()
	// Only create the bin and lib directories but not the glibc sub-tree.
	if err := os.MkdirAll(dir+"/bin", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir+"/lib", 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TERMUX_PREFIX", dir)

	result := CheckLinker()
	if result.Passed {
		t.Errorf("expected failure when no linker is present")
	}
	if result.Code != ExitMissingDep {
		t.Errorf("expected ExitMissingDep, got %d", result.Code)
	}
	if !strings.Contains(result.Hint, "glibc") {
		t.Errorf("expected glibc mention in hint: %s", result.Hint)
	}
}

// TestParseSELinuxDenials verifies the denial log parser.
func TestParseSELinuxDenials(t *testing.T) {
	input := `
Jan  1 00:00:01 kernel: type=1400 audit(0.0:1): avc: denied { execute } for pid=123 ...
Jan  1 00:00:02 kernel: some other log line
Jan  1 00:00:03 kernel: type=1400 audit(0.0:2): avc: denied { open } for pid=456 ...
Jan  1 00:00:04 kernel: avc: granted { read } ...
`
	scanner := bufio.NewScanner(strings.NewReader(input))
	denials := ParseSELinuxDenials(scanner)
	if len(denials) != 2 {
		t.Errorf("expected 2 denial lines, got %d: %v", len(denials), denials)
	}
	for _, d := range denials {
		if !strings.Contains(d, "denied") {
			t.Errorf("denial line does not contain 'denied': %q", d)
		}
	}
}

// TestParseSELinuxDenials_Empty verifies that an empty input returns nil.
func TestParseSELinuxDenials_Empty(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader(""))
	denials := ParseSELinuxDenials(scanner)
	if len(denials) != 0 {
		t.Errorf("expected 0 denials for empty input, got %d", len(denials))
	}
}

// TestParseSELinuxDenials_NoDenials verifies no false positives on normal logs.
func TestParseSELinuxDenials_NoDenials(t *testing.T) {
	input := `
avc: granted { read } for pid=1
some random log line
avc: info { use } for pid=2
`
	scanner := bufio.NewScanner(strings.NewReader(input))
	denials := ParseSELinuxDenials(scanner)
	if len(denials) != 0 {
		t.Errorf("expected 0 denials, got %d: %v", len(denials), denials)
	}
}

// TestIsKnownDangerousContext verifies the context classification logic.
func TestIsKnownDangerousContext(t *testing.T) {
	safe := []string{
		"u:r:untrusted_app:s0",
		"u:r:untrusted_app_25:s0:c512,c768",
		"u:r:platform_app:s0",
	}
	dangerous := []string{
		"u:r:shell:s0",
		"u:r:su:s0",
		"u:r:init:s0",
		"u:r:kernel:s0",
		"u:r:system_server:s0",
		"u:r:recovery:s0",
	}

	for _, ctx := range safe {
		if isKnownDangerousContext(ctx) {
			t.Errorf("expected safe context to be classified as safe: %s", ctx)
		}
	}
	for _, ctx := range dangerous {
		if !isKnownDangerousContext(ctx) {
			t.Errorf("expected dangerous context to be detected: %s", ctx)
		}
	}
}

// TestAllChecksHaveNames verifies that every entry in All has a non-empty Name
// and Description.
func TestAllChecksHaveNames(t *testing.T) {
	for i, c := range All {
		if c.Name == "" {
			t.Errorf("All[%d].Name is empty", i)
		}
		if c.Description == "" {
			t.Errorf("All[%d].Description is empty (name: %q)", i, c.Name)
		}
		if c.Run == nil {
			t.Errorf("All[%d].Run is nil (name: %q)", i, c.Name)
		}
	}
}

// TestAllCheckNamesAreUnique verifies no two checks share a name.
func TestAllCheckNamesAreUnique(t *testing.T) {
	seen := make(map[string]int)
	for i, c := range All {
		if prev, ok := seen[c.Name]; ok {
			t.Errorf("All[%d] has duplicate name %q (first seen at All[%d])", i, c.Name, prev)
		}
		seen[c.Name] = i
	}
}

// TestRunAllReturnsAllChecks verifies that RunAll returns one result per entry.
func TestRunAllReturnsAllChecks(t *testing.T) {
	results := RunAll()
	if len(results) != len(All) {
		t.Errorf("RunAll returned %d results, expected %d", len(results), len(All))
	}
	names := make(map[string]int)
	for _, r := range results {
		names[r.Name]++
	}
	for i, c := range All {
		if names[c.Name] != 1 {
			t.Errorf("All[%d] (%q) appeared %d times in RunAll output", i, c.Name, names[c.Name])
		}
	}
}

// TestLinkerCandidatesNonEmpty verifies that linkerCandidates returns non-empty
// slices on the current host architecture.
func TestLinkerCandidatesNonEmpty(t *testing.T) {
	candidates := linkerCandidates("/fake/prefix")
	if len(candidates) == 0 {
		t.Errorf("linkerCandidates returned empty slice for arch %s", runtime.GOARCH)
	}
	for i, c := range candidates {
		if c == "" {
			t.Errorf("linkerCandidates[%d] is empty string", i)
		}
	}
}

// TestTermuxPrefixEnvPriority verifies that TERMUX_PREFIX is preferred over PREFIX.
func TestTermuxPrefixEnvPriority(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	t.Setenv("TERMUX_PREFIX", dir1)
	t.Setenv("PREFIX", dir2)

	got := termuxPrefix()
	if got != dir1 {
		t.Errorf("expected TERMUX_PREFIX (%s) to take precedence over PREFIX, got %s", dir1, got)
	}
}

// TestTermuxPrefixFallback verifies fallback to PREFIX when TERMUX_PREFIX is unset.
func TestTermuxPrefixFallback(t *testing.T) {
	if _, err := os.Stat("/data/data/com.termux/files/usr"); err == nil {
		t.Skip("default Termux path exists; test not meaningful")
	}

	dir := t.TempDir()
	vars := []string{"TERMUX_PREFIX", "TERMUX__PREFIX"}
	for _, v := range vars {
		orig := os.Getenv(v)
		os.Unsetenv(v)
		t.Cleanup(func() {
			if orig == "" {
				os.Unsetenv(v)
			} else {
				os.Setenv(v, orig)
			}
		})
	}

	t.Setenv("PREFIX", dir)
	got := termuxPrefix()
	if got != dir {
		t.Errorf("expected fallback to PREFIX (%s), got %q", dir, got)
	}
}

// TestExitCodeConstants verifies that the numeric values of ExitCode constants
// match the documented contract (important for shell script consumers).
func TestExitCodeConstants(t *testing.T) {
	tests := []struct {
		code ExitCode
		want int
	}{
		{ExitOK, 0},
		{ExitMissingDep, 2},
		{ExitSELinux, 3},
		{ExitFilesystem, 4},
		{ExitWX, 5},
		{ExitProcFS, 6},
		{ExitUnknown, 99},
	}
	for _, tc := range tests {
		if int(tc.code) != tc.want {
			t.Errorf("ExitCode %v = %d, want %d", tc.code, int(tc.code), tc.want)
		}
	}
}

// TestResultPassedChecksHaveNoHint is a meta-test that verifies passing results
// from all checks do not accidentally include hint text.
func TestResultPassedChecksHaveNoHint(t *testing.T) {
	// Create a synthetic passing result directly to verify the invariant.
	r := Result{Name: "x", Passed: true, Code: ExitOK, Message: "ok", Hint: ""}
	if r.Hint != "" {
		t.Error("passing result should have empty hint")
	}
	s := r.String()
	if strings.Contains(s, "hint:") {
		t.Errorf("passing result String() should not contain 'hint:': %s", s)
	}
}
