package scanner

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── ReportBuilder basic round-trip ──────────────────────────────────────────

func TestReportBuilder_Empty(t *testing.T) {
	b := NewReportBuilder("/some/target", "", "")
	r := b.Build()
	if r.SchemaVersion != "1.0" {
		t.Errorf("schema_version: got %q", r.SchemaVersion)
	}
	if r.Summary.Total != 0 {
		t.Errorf("total: got %d", r.Summary.Total)
	}
	if len(r.Entries) != 0 {
		t.Errorf("entries: got %d", len(r.Entries))
	}
}

func TestReportBuilder_ELFEntry(t *testing.T) {
	b := NewReportBuilder("/bin", "", "")
	b.AddELFEntry(
		"/bin/avrdude",
		CategoryNativeAndroid,
		PatchClassNone,
		"/system/bin/linker64", "",
		"Binary already targets Android/Bionic; no patching required.",
		nil,
	)
	r := b.Build()
	if r.Summary.Total != 1 {
		t.Fatalf("total: got %d", r.Summary.Total)
	}
	if r.Summary.NativeAndroid != 1 {
		t.Errorf("native_android: got %d", r.Summary.NativeAndroid)
	}
	if r.Summary.NeedsPatch != 0 {
		t.Errorf("needs_patch: got %d", r.Summary.NeedsPatch)
	}
	e := r.Entries[0]
	if e.InterpreterStatus != nil {
		t.Error("ELF entry should not have InterpreterStatus")
	}
}

func TestReportBuilder_LinuxGlibcCounted(t *testing.T) {
	b := NewReportBuilder("/tools", "", "")
	b.AddELFEntry(
		"/tools/cc1plus",
		CategoryLinuxGlibc,
		PatchClassLoaderAndRpath,
		"/lib/ld-linux-aarch64.so.1", "",
		"Replace interpreter and RPATH.",
		[]PatchAction{
			{Action: "set-interpreter", Field: "interpreter",
				CurrentValue: "/lib/ld-linux-aarch64.so.1",
				RecommendedValue: "/data/data/com.termux/files/usr/glibc/lib/ld-linux-aarch64.so.1",
				Reason: "Android does not have /lib"},
		},
	)
	r := b.Build()
	if r.Summary.NeedsPatch != 1 {
		t.Errorf("needs_patch: got %d", r.Summary.NeedsPatch)
	}
	if r.Summary.LinuxGlibc != 1 {
		t.Errorf("linux_glibc: got %d", r.Summary.LinuxGlibc)
	}
}

// ─── Script entry with shebang validation ────────────────────────────────────

func TestReportBuilder_ScriptEntry_BashRemapped(t *testing.T) {
	prefix := buildFakePrefix(t, "bash")
	dir := t.TempDir()
	script := filepath.Join(dir, "run.sh")
	if err := os.WriteFile(script, []byte("#!/bin/bash\necho hello\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	b := NewReportBuilder(dir, prefix, "")
	if err := b.AddScriptEntry(script); err != nil {
		t.Fatal(err)
	}
	r := b.Build()

	if r.Summary.Script != 1 {
		t.Fatalf("script count: got %d", r.Summary.Script)
	}
	if r.Summary.ScriptRemapped != 1 {
		t.Errorf("script_interpreter_remapped: got %d", r.Summary.ScriptRemapped)
	}
	e := r.Entries[0]
	if e.Category != CategoryScript {
		t.Errorf("category: got %q", e.Category)
	}
	if e.InterpreterStatus == nil {
		t.Fatal("InterpreterStatus is nil")
	}
	if e.InterpreterStatus.Status != InterpreterRemapped {
		t.Errorf("status: got %q", e.InterpreterStatus.Status)
	}
	if e.InterpreterStatus.DeclaredPath != "/bin/bash" {
		t.Errorf("declared_path: got %q", e.InterpreterStatus.DeclaredPath)
	}
}

func TestReportBuilder_ScriptEntry_EnvPython3Remapped(t *testing.T) {
	prefix := buildFakePrefix(t, "python3", "env")
	dir := t.TempDir()
	script := filepath.Join(dir, "hello.py")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env python3\nprint('hello')\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := NewReportBuilder(dir, prefix, "")
	if err := b.AddScriptEntry(script); err != nil {
		t.Fatal(err)
	}
	r := b.Build()
	e := r.Entries[0]
	if e.InterpreterStatus == nil {
		t.Fatal("nil InterpreterStatus")
	}
	if e.InterpreterStatus.Status != InterpreterRemapped {
		t.Errorf("status: got %q", e.InterpreterStatus.Status)
	}
	if e.InterpreterStatus.DeclaredPath != "/usr/bin/env" {
		t.Errorf("declared_path: got %q", e.InterpreterStatus.DeclaredPath)
	}
	if len(e.InterpreterStatus.Args) == 0 || e.InterpreterStatus.Args[0] != "python3" {
		t.Errorf("args: got %v", e.InterpreterStatus.Args)
	}
}

func TestReportBuilder_ScriptEntry_Missing(t *testing.T) {
	prefix := buildFakePrefix(t /* nothing installed */)
	dir := t.TempDir()
	script := filepath.Join(dir, "script.py")
	if err := os.WriteFile(script, []byte("#!/usr/bin/python3\npass\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := NewReportBuilder(dir, prefix, "")
	if err := b.AddScriptEntry(script); err != nil {
		t.Fatal(err)
	}
	r := b.Build()
	if r.Summary.ScriptMissing != 1 {
		t.Errorf("script_interpreter_missing: got %d", r.Summary.ScriptMissing)
	}
	e := r.Entries[0]
	if e.InterpreterStatus.Status != InterpreterMissing {
		t.Errorf("status: got %q", e.InterpreterStatus.Status)
	}
}

func TestReportBuilder_ScriptEntry_NoShebang(t *testing.T) {
	prefix := buildFakePrefix(t, "bash")
	dir := t.TempDir()
	script := filepath.Join(dir, "noShebang.sh")
	if err := os.WriteFile(script, []byte("echo hello\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	b := NewReportBuilder(dir, prefix, "")
	if err := b.AddScriptEntry(script); err != nil {
		t.Fatal(err)
	}
	r := b.Build()
	e := r.Entries[0]
	// No shebang → InterpreterStatus should be nil.
	if e.InterpreterStatus != nil {
		t.Errorf("expected nil InterpreterStatus for file without shebang, got %+v", e.InterpreterStatus)
	}
}

// ─── Summary counters ────────────────────────────────────────────────────────

func TestReportSummary_MixedEntries(t *testing.T) {
	prefix := buildFakePrefix(t, "bash", "python3")
	dir := t.TempDir()

	bashScript := filepath.Join(dir, "run.sh")
	if err := os.WriteFile(bashScript, []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	pyScript := filepath.Join(dir, "hello.py")
	if err := os.WriteFile(pyScript, []byte("#!/usr/bin/python3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	missingScript := filepath.Join(dir, "perl.pl")
	if err := os.WriteFile(missingScript, []byte("#!/usr/bin/perl\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := NewReportBuilder(dir, prefix, "")
	for _, s := range []string{bashScript, pyScript, missingScript} {
		if err := b.AddScriptEntry(s); err != nil {
			t.Fatalf("AddScriptEntry(%s): %v", s, err)
		}
	}
	// Also add an ELF entry.
	b.AddELFEntry("/bin/tool", CategoryNativeAndroid, PatchClassNone, "/system/bin/linker64", "", "ok", nil)

	r := b.Build()

	if r.Summary.Total != 4 {
		t.Errorf("total: got %d, want 4", r.Summary.Total)
	}
	if r.Summary.Script != 3 {
		t.Errorf("script: got %d, want 3", r.Summary.Script)
	}
	if r.Summary.ScriptRemapped != 2 {
		t.Errorf("script_remapped: got %d, want 2", r.Summary.ScriptRemapped)
	}
	if r.Summary.ScriptMissing != 1 {
		t.Errorf("script_missing: got %d, want 1", r.Summary.ScriptMissing)
	}
	if r.Summary.NativeAndroid != 1 {
		t.Errorf("native_android: got %d, want 1", r.Summary.NativeAndroid)
	}
}

// ─── JSON round-trip ─────────────────────────────────────────────────────────

func TestWriteJSON_RoundTrip(t *testing.T) {
	b := NewReportBuilder("/target", "", "")
	b.AddELFEntry("/bin/foo", CategoryNativeAndroid, PatchClassNone, "/system/bin/linker64", "", "ok", nil)

	r := b.Build()

	var buf bytes.Buffer
	if err := WriteJSON(&buf, r); err != nil {
		t.Fatal(err)
	}

	var decoded Report
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v — JSON:\n%s", err, buf.String())
	}
	if decoded.SchemaVersion != "1.0" {
		t.Errorf("schema_version: got %q", decoded.SchemaVersion)
	}
	if len(decoded.Entries) != 1 {
		t.Errorf("entries: got %d", len(decoded.Entries))
	}
}

func TestWriteJSON_ScriptInterpreterStatusField(t *testing.T) {
	prefix := buildFakePrefix(t, "bash")
	dir := t.TempDir()
	script := filepath.Join(dir, "run.sh")
	if err := os.WriteFile(script, []byte("#!/bin/bash\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	b := NewReportBuilder(dir, prefix, "")
	if err := b.AddScriptEntry(script); err != nil {
		t.Fatal(err)
	}
	r := b.Build()

	var buf bytes.Buffer
	if err := WriteJSON(&buf, r); err != nil {
		t.Fatal(err)
	}

	raw := buf.String()
	// The JSON output must contain the interpreter_status key.
	if !strings.Contains(raw, `"interpreter_status"`) {
		t.Errorf("JSON missing interpreter_status field:\n%s", raw)
	}
	if !strings.Contains(raw, `"status"`) {
		t.Errorf("JSON missing status field:\n%s", raw)
	}
	if !strings.Contains(raw, `"declared_path"`) {
		t.Errorf("JSON missing declared_path field:\n%s", raw)
	}
}

// ─── WriteText sanity check ───────────────────────────────────────────────────

func TestWriteText_ScriptEntry(t *testing.T) {
	prefix := buildFakePrefix(t, "python3")
	dir := t.TempDir()
	script := filepath.Join(dir, "hello.py")
	if err := os.WriteFile(script, []byte("#!/usr/bin/python3\npass\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := NewReportBuilder(dir, prefix, "")
	if err := b.AddScriptEntry(script); err != nil {
		t.Fatal(err)
	}
	r := b.Build()

	var buf bytes.Buffer
	if err := WriteText(&buf, r); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.Contains(out, "interp_status") {
		t.Errorf("text output missing interp_status:\n%s", out)
	}
	if !strings.Contains(out, "/usr/bin/python3") {
		t.Errorf("text output missing declared interpreter:\n%s", out)
	}
}

// ─── ScanFile dispatch ────────────────────────────────────────────────────────

func TestScanFile_DetectsScript(t *testing.T) {
	prefix := buildFakePrefix(t, "bash")
	dir := t.TempDir()
	script := filepath.Join(dir, "run.sh")
	if err := os.WriteFile(script, []byte("#!/bin/bash\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	b := NewReportBuilder(dir, prefix, "")
	if err := b.ScanFile(script); err != nil {
		t.Fatal(err)
	}
	r := b.Build()
	if r.Summary.Script != 1 {
		t.Errorf("script count: got %d", r.Summary.Script)
	}
	e := r.Entries[0]
	if e.Category != CategoryScript {
		t.Errorf("category: got %q", e.Category)
	}
	if e.InterpreterStatus == nil {
		t.Error("InterpreterStatus should not be nil for shebang script")
	}
}

func TestScanFile_UnknownFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "data.bin")
	if err := os.WriteFile(p, []byte{0x00, 0x01, 0x02, 0x03}, 0o644); err != nil {
		t.Fatal(err)
	}

	b := NewReportBuilder(dir, "", "")
	if err := b.ScanFile(p); err != nil {
		t.Fatal(err)
	}
	r := b.Build()
	if r.Summary.Unknown != 1 {
		t.Errorf("unknown count: got %d", r.Summary.Unknown)
	}
}
