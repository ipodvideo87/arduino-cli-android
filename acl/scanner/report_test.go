package scanner_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/arduino/arduino-cli/acl/scanner"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// mustReadFile returns the contents of a testdata file or skips if missing.
func mustReadFile(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Skipf("testdata/%s not found – skipping: %v", name, err)
	}
	return data
}

// ─── Unit: CompatCategory ─────────────────────────────────────────────────────

func TestCompatCategoryConstants(t *testing.T) {
	cases := []struct {
		cat  scanner.CompatCategory
		want string
	}{
		{scanner.CompatNativeAndroid, "native Android compatible"},
		{scanner.CompatLinuxGlibc, "Linux/glibc executable"},
		{scanner.CompatStatic, "static ELF"},
		{scanner.CompatScript, "script"},
		{scanner.CompatUnknown, "unknown"},
		{scanner.CompatUnsupported, "unsupported"},
	}
	for _, tc := range cases {
		if string(tc.cat) != tc.want {
			t.Errorf("CompatCategory constant: got %q, want %q", tc.cat, tc.want)
		}
	}
}

// ─── Unit: PatchAction ───────────────────────────────────────────────────────

func TestPatchActionConstants(t *testing.T) {
	cases := []struct {
		action scanner.PatchAction
		want   string
	}{
		{scanner.PatchNoAction, "no-action"},
		{scanner.PatchRewriteInterpreter, "rewrite-interpreter"},
		{scanner.PatchInjectRpath, "inject-rpath"},
		{scanner.PatchRewriteInterpreterAndRpath, "rewrite-interpreter-and-rpath"},
		{scanner.PatchScriptNoop, "script-no-elf-patch"},
		{scanner.PatchUnsupported, "unsupported"},
	}
	for _, tc := range cases {
		if string(tc.action) != tc.want {
			t.Errorf("PatchAction constant: got %q, want %q", tc.action, tc.want)
		}
	}
}

// ─── Unit: Recommend ─────────────────────────────────────────────────────────

func TestRecommend_NativeAndroid(t *testing.T) {
	rec := scanner.Recommend(scanner.CompatNativeAndroid, nil)
	if rec.Action != scanner.PatchNoAction {
		t.Errorf("NativeAndroid: expected no-action, got %q", rec.Action)
	}
	if rec.Rationale == "" {
		t.Error("NativeAndroid: expected non-empty Rationale")
	}
}

func TestRecommend_Static(t *testing.T) {
	rec := scanner.Recommend(scanner.CompatStatic, &scanner.ELFInfo{IsStatic: true})
	if rec.Action != scanner.PatchNoAction {
		t.Errorf("Static: expected no-action, got %q", rec.Action)
	}
}

func TestRecommend_Script(t *testing.T) {
	rec := scanner.Recommend(scanner.CompatScript, nil)
	if rec.Action != scanner.PatchScriptNoop {
		t.Errorf("Script: expected script-no-elf-patch, got %q", rec.Action)
	}
}

func TestRecommend_Unsupported(t *testing.T) {
	rec := scanner.Recommend(scanner.CompatUnsupported, nil)
	if rec.Action != scanner.PatchUnsupported {
		t.Errorf("Unsupported: expected unsupported, got %q", rec.Action)
	}
}

func TestRecommend_LinuxGlibc_BothNeeded(t *testing.T) {
	info := &scanner.ELFInfo{
		Interpreter: "/lib64/ld-linux-x86-64.so.2",
		Rpath:       "",
		Runpath:     "",
	}
	rec := scanner.Recommend(scanner.CompatLinuxGlibc, info)
	if rec.Action != scanner.PatchRewriteInterpreterAndRpath {
		t.Errorf("LinuxGlibc both: expected rewrite-interpreter-and-rpath, got %q", rec.Action)
	}
	if rec.SuggestedInterpreter == "" {
		t.Error("LinuxGlibc both: expected SuggestedInterpreter to be set")
	}
	if rec.SuggestedRpath == "" {
		t.Error("LinuxGlibc both: expected SuggestedRpath to be set")
	}
}

func TestRecommend_LinuxGlibc_InterpOnly(t *testing.T) {
	info := &scanner.ELFInfo{
		Interpreter: "/lib64/ld-linux-x86-64.so.2",
		Runpath:     "/data/data/com.termux/files/usr/lib/acl-runtime/lib",
	}
	rec := scanner.Recommend(scanner.CompatLinuxGlibc, info)
	if rec.Action != scanner.PatchRewriteInterpreter {
		t.Errorf("LinuxGlibc interp-only: expected rewrite-interpreter, got %q", rec.Action)
	}
	if rec.SuggestedInterpreter == "" {
		t.Error("LinuxGlibc interp-only: expected SuggestedInterpreter to be set")
	}
	if rec.SuggestedRpath != "" {
		t.Errorf("LinuxGlibc interp-only: SuggestedRpath should be empty, got %q", rec.SuggestedRpath)
	}
}

func TestRecommend_LinuxGlibc_RpathOnly(t *testing.T) {
	// Interpreter already set to Android-compatible path; rpath missing.
	info := &scanner.ELFInfo{
		Interpreter: "/system/bin/linker64",
		Rpath:       "",
		Runpath:     "",
	}
	rec := scanner.Recommend(scanner.CompatLinuxGlibc, info)
	if rec.Action != scanner.PatchInjectRpath {
		t.Errorf("LinuxGlibc rpath-only: expected inject-rpath, got %q", rec.Action)
	}
	if rec.SuggestedRpath == "" {
		t.Error("LinuxGlibc rpath-only: expected SuggestedRpath to be set")
	}
}

func TestRecommend_LinuxGlibc_NilInfo(t *testing.T) {
	rec := scanner.Recommend(scanner.CompatLinuxGlibc, nil)
	if rec.Action != scanner.PatchUnsupported {
		t.Errorf("LinuxGlibc nil info: expected unsupported, got %q", rec.Action)
	}
}

func TestRecommend_Unknown(t *testing.T) {
	rec := scanner.Recommend(scanner.CompatUnknown, nil)
	if rec.Action != scanner.PatchUnsupported {
		t.Errorf("Unknown: expected unsupported, got %q", rec.Action)
	}
}

// ─── Unit: FindMissingSymbols ────────────────────────────────────────────────

func TestFindMissingSymbols_GlibcOnlyLibs(t *testing.T) {
	info := &scanner.ELFInfo{
		Needed: []string{"libc.so.6", "libpthread.so.0", "libssl.so.1.1"},
	}
	missing := scanner.FindMissingSymbols(info)
	if len(missing) != 2 {
		t.Fatalf("expected 2 missing symbols, got %d: %v", len(missing), missing)
	}
	names := map[string]bool{}
	for _, m := range missing {
		names[m.Name] = true
	}
	if !names["libc.so.6"] {
		t.Error("expected libc.so.6 to be flagged as missing")
	}
	if !names["libpthread.so.0"] {
		t.Error("expected libpthread.so.0 to be flagged as missing")
	}
}

func TestFindMissingSymbols_NoGlibcLibs(t *testing.T) {
	info := &scanner.ELFInfo{
		Needed: []string{"libssl.so.1.1", "libcrypto.so.1.1"},
	}
	missing := scanner.FindMissingSymbols(info)
	if len(missing) != 0 {
		t.Errorf("expected 0 missing symbols, got %d", len(missing))
	}
}

func TestFindMissingSymbols_NilInfo(t *testing.T) {
	missing := scanner.FindMissingSymbols(nil)
	if missing != nil {
		t.Errorf("expected nil for nil info, got %v", missing)
	}
}

// ─── Unit: BuildSummary ───────────────────────────────────────────────────────

func TestBuildSummary(t *testing.T) {
	reports := []scanner.BinaryReport{
		{CompatCategory: scanner.CompatNativeAndroid, Recommendation: scanner.PatchRecommendation{Action: scanner.PatchNoAction}},
		{CompatCategory: scanner.CompatLinuxGlibc, Recommendation: scanner.PatchRecommendation{Action: scanner.PatchRewriteInterpreterAndRpath}},
		{CompatCategory: scanner.CompatLinuxGlibc, Recommendation: scanner.PatchRecommendation{Action: scanner.PatchInjectRpath}},
		{CompatCategory: scanner.CompatStatic, Recommendation: scanner.PatchRecommendation{Action: scanner.PatchNoAction}},
		{CompatCategory: scanner.CompatScript, Recommendation: scanner.PatchRecommendation{Action: scanner.PatchScriptNoop}},
		{CompatCategory: scanner.CompatUnknown, Recommendation: scanner.PatchRecommendation{Action: scanner.PatchUnsupported}},
		{CompatCategory: scanner.CompatUnsupported, Recommendation: scanner.PatchRecommendation{Action: scanner.PatchUnsupported}, Error: "parse failed"},
	}
	s := scanner.BuildSummary(reports)

	assertEqual(t, "Total", 7, s.Total)
	assertEqual(t, "NativeAndroid", 1, s.NativeAndroid)
	assertEqual(t, "LinuxGlibc", 2, s.LinuxGlibc)
	assertEqual(t, "Static", 1, s.Static)
	assertEqual(t, "Script", 1, s.Script)
	assertEqual(t, "Unknown", 1, s.Unknown)
	assertEqual(t, "Unsupported", 1, s.Unsupported)
	assertEqual(t, "Errors", 1, s.Errors)
	assertEqual(t, "NeedsPatch", 2, s.NeedsPatch)
}

func assertEqual(t *testing.T, field string, want, got int) {
	t.Helper()
	if want != got {
		t.Errorf("Summary.%s: want %d, got %d", field, want, got)
	}
}

// ─── Unit: MarshalReport ─────────────────────────────────────────────────────

func TestMarshalReport_ValidJSON(t *testing.T) {
	report := scanner.ScanReport{
		SchemaVersion: scanner.ReportSchemaVersion,
		GeneratedAt:   "2026-01-01T00:00:00Z",
		Binaries: []scanner.BinaryReport{
			{
				Path:           "/usr/bin/example",
				CompatCategory: scanner.CompatLinuxGlibc,
				ELF: &scanner.ELFInfo{
					Class:       "ELF64",
					Machine:     "EM_AARCH64",
					Interpreter: "/lib/ld-linux-aarch64.so.1",
				},
				Recommendation: scanner.PatchRecommendation{
					Action:               scanner.PatchRewriteInterpreterAndRpath,
					SuggestedInterpreter: "/data/data/com.termux/files/usr/lib/acl-runtime/loader/ld-linux-aarch64.so.1",
					SuggestedRpath:       "/data/data/com.termux/files/usr/lib/acl-runtime/lib",
					Rationale:            "test",
				},
			},
		},
		Summary: scanner.ScanSummary{Total: 1, LinuxGlibc: 1, NeedsPatch: 1},
	}

	data, err := scanner.MarshalReport(report)
	if err != nil {
		t.Fatalf("MarshalReport returned error: %v", err)
	}

	// Confirm it round-trips cleanly.
	var roundTrip scanner.ScanReport
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("Unmarshal of marshalled report failed: %v", err)
	}
	if roundTrip.SchemaVersion != scanner.ReportSchemaVersion {
		t.Errorf("schema_version mismatch: got %q", roundTrip.SchemaVersion)
	}
	if len(roundTrip.Binaries) != 1 {
		t.Errorf("expected 1 binary, got %d", len(roundTrip.Binaries))
	}
}

func TestMarshalReport_SchemaVersion(t *testing.T) {
	report := scanner.ScanReport{SchemaVersion: scanner.ReportSchemaVersion}
	data, err := scanner.MarshalReport(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["schema_version"] != "1.0" {
		t.Errorf("schema_version: got %v, want 1.0", m["schema_version"])
	}
}

// ─── Integration: ScanPaths with real ELF (Linux only) ───────────────────────

// TestScanPaths_RealBinary inspects the test binary itself (or a small ELF
// produced by the Go toolchain) when running on Linux.  This validates the
// end-to-end pipeline without requiring a cross-compiled Android binary.
func TestScanPaths_RealBinary(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("ELF inspection only meaningful on Linux; skipping on " + runtime.GOOS)
	}

	// Locate a suitable ELF binary.
	candidate, err := exec.LookPath("ls")
	if err != nil {
		t.Skip("cannot find 'ls' binary; skipping ELF inspection test")
	}

	report := scanner.ScanPaths([]string{candidate})
	if report.SchemaVersion != scanner.ReportSchemaVersion {
		t.Errorf("schema_version: got %q", report.SchemaVersion)
	}
	if report.Summary.Total != 1 {
		t.Errorf("expected 1 total, got %d", report.Summary.Total)
	}
	bin := report.Binaries[0]
	if bin.Path != candidate {
		t.Errorf("path mismatch: got %q, want %q", bin.Path, candidate)
	}
	if bin.CompatCategory == "" {
		t.Error("expected non-empty compat_category")
	}
	if bin.Recommendation.Action == "" {
		t.Error("expected non-empty recommendation.action")
	}
	if bin.Recommendation.Rationale == "" {
		t.Error("expected non-empty recommendation.rationale")
	}
}

// ─── Integration: ScanPaths with a script ────────────────────────────────────

func TestScanPaths_Script(t *testing.T) {
	tmp := t.TempDir()
	script := filepath.Join(tmp, "run.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	report := scanner.ScanPaths([]string{script})
	if report.Summary.Total != 1 {
		t.Fatalf("expected 1 total, got %d", report.Summary.Total)
	}
	bin := report.Binaries[0]
	if bin.CompatCategory != scanner.CompatScript {
		t.Errorf("expected script category, got %q", bin.CompatCategory)
	}
	if bin.Recommendation.Action != scanner.PatchScriptNoop {
		t.Errorf("expected script-no-elf-patch, got %q", bin.Recommendation.Action)
	}
}

// ─── Integration: ScanPaths with a Windows PE (.exe extension) ───────────────

func TestScanPaths_WindowsBinary(t *testing.T) {
	tmp := t.TempDir()
	exe := filepath.Join(tmp, "tool.exe")
	// Minimal MZ header.
	mz := []byte{0x4D, 0x5A, 0x00, 0x00}
	if err := os.WriteFile(exe, mz, 0o644); err != nil {
		t.Fatal(err)
	}
	report := scanner.ScanPaths([]string{exe})
	bin := report.Binaries[0]
	if bin.CompatCategory != scanner.CompatUnsupported {
		t.Errorf("expected unsupported for .exe, got %q", bin.CompatCategory)
	}
	if bin.Recommendation.Action != scanner.PatchUnsupported {
		t.Errorf("expected unsupported action, got %q", bin.Recommendation.Action)
	}
}

// ─── Golden-file tests ───────────────────────────────────────────────────────

// TestGoldenReport_LinuxGlibc verifies the JSON output against a golden file
// stored in testdata/golden/.  Run with UPDATE_GOLDEN=1 to regenerate.
func TestGoldenReport_LinuxGlibc(t *testing.T) {
	report := scanner.ScanReport{
		SchemaVersion: scanner.ReportSchemaVersion,
		GeneratedAt:   "2026-06-19T00:00:00Z", // Fixed for determinism.
		Binaries: []scanner.BinaryReport{
			{
				Path:           "/usr/bin/avr-gcc",
				CompatCategory: scanner.CompatLinuxGlibc,
				ELF: &scanner.ELFInfo{
					Class:       "ELF64",
					Machine:     "EM_X86_64",
					Interpreter: "/lib64/ld-linux-x86-64.so.2",
					Needed:      []string{"libc.so.6", "libpthread.so.0", "libgcc_s.so.1"},
				},
				MissingSymbols: []scanner.MissingSymbol{
					{Name: "libc.so.6", Library: "libc.so.6", Reason: "glibc libc — Bionic provides libc.so, not libc.so.6"},
					{Name: "libpthread.so.0", Library: "libpthread.so.0", Reason: "POSIX threads — folded into libc.so on Bionic/Android ≥ 5.0"},
					{Name: "libgcc_s.so.1", Library: "libgcc_s.so.1", Reason: "GCC runtime — not provided by Android NDK default paths"},
				},
				Recommendation: scanner.PatchRecommendation{
					Action:               scanner.PatchRewriteInterpreterAndRpath,
					SuggestedInterpreter: "/data/data/com.termux/files/usr/lib/acl-runtime/loader/ld-linux-aarch64.so.1",
					SuggestedRpath:       "/data/data/com.termux/files/usr/lib/acl-runtime/lib",
					Rationale: `PT_INTERP "/lib64/ld-linux-x86-64.so.2" is a glibc dynamic linker; must be replaced with ACL loader. ` +
						`No ACL RPATH present; must inject "/data/data/com.termux/files/usr/lib/acl-runtime/lib" so the ACL runtime libraries are found.`,
				},
			},
		},
		Summary: scanner.ScanSummary{
			Total:      1,
			LinuxGlibc: 1,
			NeedsPatch: 1,
		},
	}

	got, err := scanner.MarshalReport(report)
	if err != nil {
		t.Fatalf("MarshalReport: %v", err)
	}

	goldenPath := filepath.Join("testdata", "golden", "linux-glibc-report.json")

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("golden file updated: %s", goldenPath)
		return
	}

	want := mustReadFile(t, filepath.Join("golden", "linux-glibc-report.json"))
	if string(got) != string(want) {
		t.Errorf("golden mismatch for %s\n--- want ---\n%s\n--- got ---\n%s", goldenPath, want, got)
	}
}

func TestGoldenReport_NativeAndroid(t *testing.T) {
	report := scanner.ScanReport{
		SchemaVersion: scanner.ReportSchemaVersion,
		GeneratedAt:   "2026-06-19T00:00:00Z",
		Binaries: []scanner.BinaryReport{
			{
				Path:           "/data/data/com.termux/files/usr/bin/avrdude",
				CompatCategory: scanner.CompatNativeAndroid,
				ELF: &scanner.ELFInfo{
					Class:       "ELF64",
					Machine:     "EM_AARCH64",
					Interpreter: "/system/bin/linker64",
					Needed:      []string{"libusb-1.0.so", "libhidapi.so", "libc.so"},
				},
				Recommendation: scanner.PatchRecommendation{
					Action:    scanner.PatchNoAction,
					Rationale: "Binary already targets Android/Bionic; no patching required.",
				},
			},
		},
		Summary: scanner.ScanSummary{
			Total:         1,
			NativeAndroid: 1,
		},
	}

	got, err := scanner.MarshalReport(report)
	if err != nil {
		t.Fatalf("MarshalReport: %v", err)
	}

	goldenPath := filepath.Join("testdata", "golden", "native-android-report.json")

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("golden file updated: %s", goldenPath)
		return
	}

	want := mustReadFile(t, filepath.Join("golden", "native-android-report.json"))
	if string(got) != string(want) {
		t.Errorf("golden mismatch for %s\n--- want ---\n%s\n--- got ---\n%s", goldenPath, want, got)
	}
}

func TestGoldenReport_Mixed(t *testing.T) {
	report := scanner.ScanReport{
		SchemaVersion: scanner.ReportSchemaVersion,
		GeneratedAt:   "2026-06-19T00:00:00Z",
		Binaries: []scanner.BinaryReport{
			{
				Path:           "/tools/esptool.py",
				CompatCategory: scanner.CompatScript,
				Recommendation: scanner.PatchRecommendation{
					Action:    scanner.PatchScriptNoop,
					Rationale: "Script file; ELF patching is not applicable.",
				},
			},
			{
				Path:           "/tools/xtensa-esp32-elf-gcc",
				CompatCategory: scanner.CompatLinuxGlibc,
				ELF: &scanner.ELFInfo{
					Class:       "ELF64",
					Machine:     "EM_X86_64",
					Interpreter: "/lib64/ld-linux-x86-64.so.2",
					Needed:      []string{"libc.so.6"},
				},
				MissingSymbols: []scanner.MissingSymbol{
					{Name: "libc.so.6", Library: "libc.so.6", Reason: "glibc libc — Bionic provides libc.so, not libc.so.6"},
				},
				Recommendation: scanner.PatchRecommendation{
					Action:               scanner.PatchRewriteInterpreterAndRpath,
					SuggestedInterpreter: "/data/data/com.termux/files/usr/lib/acl-runtime/loader/ld-linux-aarch64.so.1",
					SuggestedRpath:       "/data/data/com.termux/files/usr/lib/acl-runtime/lib",
					Rationale: `PT_INTERP "/lib64/ld-linux-x86-64.so.2" is a glibc dynamic linker; must be replaced with ACL loader. ` +
						`No ACL RPATH present; must inject "/data/data/com.termux/files/usr/lib/acl-runtime/lib" so the ACL runtime libraries are found.`,
				},
			},
			{
				Path:           "/tools/flash.exe",
				CompatCategory: scanner.CompatUnsupported,
				Recommendation: scanner.PatchRecommendation{
					Action:    scanner.PatchUnsupported,
					Rationale: "Binary format not supported on Android (e.g. Windows PE); cannot patch.",
				},
			},
		},
		Summary: scanner.ScanSummary{
			Total:       3,
			Script:      1,
			LinuxGlibc:  1,
			Unsupported: 1,
			NeedsPatch:  1,
		},
	}

	got, err := scanner.MarshalReport(report)
	if err != nil {
		t.Fatalf("MarshalReport: %v", err)
	}

	goldenPath := filepath.Join("testdata", "golden", "mixed-report.json")

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("golden file updated: %s", goldenPath)
		return
	}

	want := mustReadFile(t, filepath.Join("golden", "mixed-report.json"))
	if string(got) != string(want) {
		t.Errorf("golden mismatch for %s\n--- want ---\n%s\n--- got ---\n%s", goldenPath, want, got)
	}
}
