package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// ParseShebang
// ---------------------------------------------------------------------------

func TestParseShebang(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantInterp  string
		wantArgs    []string
	}{
		{
			name:       "bash",
			line:       "#!/bin/bash",
			wantInterp: "/bin/bash",
			wantArgs:   nil,
		},
		{
			name:       "env python3",
			line:       "#!/usr/bin/env python3",
			wantInterp: "/usr/bin/env",
			wantArgs:   []string{"python3"},
		},
		{
			name:       "env with flags",
			line:       "#!/usr/bin/env -S python3 -u",
			wantInterp: "/usr/bin/env",
			wantArgs:   []string{"-S", "python3", "-u"},
		},
		{
			name:       "perl",
			line:       "#!/usr/bin/perl",
			wantInterp: "/usr/bin/perl",
			wantArgs:   nil,
		},
		{
			name:       "python absolute",
			line:       "#!/usr/bin/python3",
			wantInterp: "/usr/bin/python3",
			wantArgs:   nil,
		},
		{
			name:       "no shebang",
			line:       "# normal comment",
			wantInterp: "",
			wantArgs:   nil,
		},
		{
			name:       "empty shebang",
			line:       "#!",
			wantInterp: "",
			wantArgs:   nil,
		},
		{
			name:       "whitespace after hashbang",
			line:       "#!  /usr/bin/python3  ",
			wantInterp: "/usr/bin/python3",
			wantArgs:   nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotInterp, gotArgs := ParseShebang(tc.line)
			if gotInterp != tc.wantInterp {
				t.Errorf("interpreter: got %q, want %q", gotInterp, tc.wantInterp)
			}
			if len(gotArgs) != len(tc.wantArgs) {
				t.Errorf("args len: got %d (%v), want %d (%v)", len(gotArgs), gotArgs, len(tc.wantArgs), tc.wantArgs)
				return
			}
			for i, a := range tc.wantArgs {
				if gotArgs[i] != a {
					t.Errorf("args[%d]: got %q, want %q", i, gotArgs[i], a)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ReadShebang
// ---------------------------------------------------------------------------

func TestReadShebang(t *testing.T) {
	dir := t.TempDir()

	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		return p
	}

	t.Run("script with shebang", func(t *testing.T) {
		p := write("script.sh", "#!/bin/bash\necho hello\n")
		got, err := ReadShebang(p)
		if err != nil {
			t.Fatal(err)
		}
		if got != "#!/bin/bash" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("file without shebang", func(t *testing.T) {
		p := write("plain.txt", "# just a comment\n")
		got, err := ReadShebang(p)
		if err != nil {
			t.Fatal(err)
		}
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		p := write("empty.txt", "")
		got, err := ReadShebang(p)
		if err != nil {
			t.Fatal(err)
		}
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := ReadShebang(filepath.Join(dir, "does-not-exist"))
		if err == nil {
			t.Error("expected error for missing file")
		}
	})
}

// ---------------------------------------------------------------------------
// CheckInterpreter — using a fake PREFIX directory
// ---------------------------------------------------------------------------

// buildFakePrefix creates a minimal fake Termux PREFIX tree with the
// supplied binary names under <prefix>/bin/.
func buildFakePrefix(t *testing.T, binaries ...string) string {
	t.Helper()
	prefix := t.TempDir()
	binDir := filepath.Join(prefix, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	for _, b := range binaries {
		p := filepath.Join(binDir, b)
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("write %s: %v", b, err)
		}
	}
	return prefix
}

func TestCheckInterpreter_Found(t *testing.T) {
	prefix := buildFakePrefix(t, "bash", "python3", "perl", "env")

	// Create a real file at a path that matches the interpreter so the
	// "literal exists" branch fires.
	realBash := filepath.Join(prefix, "bin", "bash")

	res := CheckInterpreter(realBash, nil, prefix, "")
	if res.Status != InterpreterFound {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterFound)
	}
	if res.ResolvedPath != realBash {
		t.Errorf("resolved: got %q, want %q", res.ResolvedPath, realBash)
	}
}

func TestCheckInterpreter_RemappedBash(t *testing.T) {
	prefix := buildFakePrefix(t, "bash")

	res := CheckInterpreter("/bin/bash", nil, prefix, "")
	if res.Status != InterpreterRemapped {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterRemapped)
	}
	want := filepath.Join(prefix, "bin", "bash")
	if res.ResolvedPath != want {
		t.Errorf("resolved: got %q, want %q", res.ResolvedPath, want)
	}
}

func TestCheckInterpreter_RemappedPython3(t *testing.T) {
	prefix := buildFakePrefix(t, "python3")

	res := CheckInterpreter("/usr/bin/python3", nil, prefix, "")
	if res.Status != InterpreterRemapped {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterRemapped)
	}
	want := filepath.Join(prefix, "bin", "python3")
	if res.ResolvedPath != want {
		t.Errorf("resolved: got %q, want %q", res.ResolvedPath, want)
	}
}

func TestCheckInterpreter_RemappedPerl(t *testing.T) {
	prefix := buildFakePrefix(t, "perl")

	res := CheckInterpreter("/usr/bin/perl", nil, prefix, "")
	if res.Status != InterpreterRemapped {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterRemapped)
	}
	want := filepath.Join(prefix, "bin", "perl")
	if res.ResolvedPath != want {
		t.Errorf("resolved: got %q, want %q", res.ResolvedPath, want)
	}
}

func TestCheckInterpreter_EnvBash(t *testing.T) {
	prefix := buildFakePrefix(t, "bash", "env")

	res := CheckInterpreter("/usr/bin/env", []string{"bash"}, prefix, "")
	if res.Status != InterpreterRemapped {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterRemapped)
	}
	want := filepath.Join(prefix, "bin", "bash")
	if res.ResolvedPath != want {
		t.Errorf("resolved: got %q, want %q", res.ResolvedPath, want)
	}
}

func TestCheckInterpreter_EnvPython3(t *testing.T) {
	prefix := buildFakePrefix(t, "python3", "env")

	res := CheckInterpreter("/usr/bin/env", []string{"python3"}, prefix, "")
	if res.Status != InterpreterRemapped {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterRemapped)
	}
	want := filepath.Join(prefix, "bin", "python3")
	if res.ResolvedPath != want {
		t.Errorf("resolved: got %q, want %q", res.ResolvedPath, want)
	}
}

func TestCheckInterpreter_EnvPerl(t *testing.T) {
	prefix := buildFakePrefix(t, "perl", "env")

	res := CheckInterpreter("/usr/bin/env", []string{"perl"}, prefix, "")
	if res.Status != InterpreterRemapped {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterRemapped)
	}
	want := filepath.Join(prefix, "bin", "perl")
	if res.ResolvedPath != want {
		t.Errorf("resolved: got %q, want %q", res.ResolvedPath, want)
	}
}

func TestCheckInterpreter_Missing(t *testing.T) {
	prefix := buildFakePrefix(t /* no binaries */)

	res := CheckInterpreter("/usr/bin/python3", nil, prefix, "")
	if res.Status != InterpreterMissing {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterMissing)
	}
	if res.ResolvedPath != "" {
		t.Errorf("resolved should be empty, got %q", res.ResolvedPath)
	}
	if res.Recommendation == "" {
		t.Error("expected non-empty recommendation for missing interpreter")
	}
}

func TestCheckInterpreter_EnvMissingDelegate(t *testing.T) {
	prefix := buildFakePrefix(t /* env present, but not python3 */)

	res := CheckInterpreter("/usr/bin/env", []string{"python3"}, prefix, "")
	if res.Status != InterpreterMissing {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterMissing)
	}
}

func TestCheckInterpreter_UnknownEnvDelegate_Fallback(t *testing.T) {
	// "myspecialtool" is not in envDelegates but is physically present.
	prefix := buildFakePrefix(t, "myspecialtool")

	res := CheckInterpreter("/usr/bin/env", []string{"myspecialtool"}, prefix, "")
	if res.Status != InterpreterRemapped {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterRemapped)
	}
	want := filepath.Join(prefix, "bin", "myspecialtool")
	if res.ResolvedPath != want {
		t.Errorf("resolved: got %q, want %q", res.ResolvedPath, want)
	}
}

func TestCheckInterpreter_RuntimeDirFallback(t *testing.T) {
	prefix := buildFakePrefix(t /* nothing in prefix */)
	runtime := t.TempDir()
	runtimeBin := filepath.Join(runtime, "bin")
	if err := os.MkdirAll(runtimeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runtimeBin, "ruby"), []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}

	res := CheckInterpreter("/usr/bin/ruby", nil, prefix, runtime)
	if res.Status != InterpreterRemapped {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterRemapped)
	}
	want := filepath.Join(runtime, "bin", "ruby")
	if res.ResolvedPath != want {
		t.Errorf("resolved: got %q, want %q", res.ResolvedPath, want)
	}
}

func TestCheckInterpreter_BasenameFallback(t *testing.T) {
	// An unknown path but the base name happens to be in PREFIX/bin.
	prefix := buildFakePrefix(t, "mytool")

	res := CheckInterpreter("/opt/local/bin/mytool", nil, prefix, "")
	if res.Status != InterpreterRemapped {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterRemapped)
	}
	want := filepath.Join(prefix, "bin", "mytool")
	if res.ResolvedPath != want {
		t.Errorf("resolved: got %q, want %q", res.ResolvedPath, want)
	}
}

func TestCheckInterpreter_EmptyPrefix(t *testing.T) {
	// With no prefix and no runtime, everything that doesn't exist literally
	// is missing.
	res := CheckInterpreter("/usr/bin/python3", nil, "", "")
	if res.Status != InterpreterMissing {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterMissing)
	}
}

// ---------------------------------------------------------------------------
// ScanShebang integration
// ---------------------------------------------------------------------------

func TestScanShebang_BashScript(t *testing.T) {
	prefix := buildFakePrefix(t, "bash")
	dir := t.TempDir()
	script := filepath.Join(dir, "run.sh")
	if err := os.WriteFile(script, []byte("#!/bin/bash\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.Status != InterpreterRemapped {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterRemapped)
	}
	if res.InterpreterPath != "/bin/bash" {
		t.Errorf("interpreter: got %q", res.InterpreterPath)
	}
}

func TestScanShebang_EnvPython3(t *testing.T) {
	prefix := buildFakePrefix(t, "python3", "env")
	dir := t.TempDir()
	script := filepath.Join(dir, "script.py")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env python3\nprint('hello')\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.Status != InterpreterRemapped {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterRemapped)
	}
	if res.InterpreterPath != "/usr/bin/env" {
		t.Errorf("interpreter: got %q", res.InterpreterPath)
	}
	if len(res.InterpreterArgs) == 0 || res.InterpreterArgs[0] != "python3" {
		t.Errorf("args: got %v", res.InterpreterArgs)
	}
}

func TestScanShebang_NoShebang(t *testing.T) {
	prefix := buildFakePrefix(t, "bash")
	dir := t.TempDir()
	p := filepath.Join(dir, "data.txt")
	if err := os.WriteFile(p, []byte("just data\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := ScanShebang(p, prefix, "")
	if err != nil {
		t.Fatal(err)
	}
	if res != nil {
		t.Errorf("expected nil for non-script, got %+v", res)
	}
}

func TestScanShebang_PerlScript(t *testing.T) {
	prefix := buildFakePrefix(t, "perl")
	dir := t.TempDir()
	script := filepath.Join(dir, "script.pl")
	if err := os.WriteFile(script, []byte("#!/usr/bin/perl\nprint 'hi';\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.Status != InterpreterRemapped {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterRemapped)
	}
}

func TestScanShebang_EnvPerl(t *testing.T) {
	prefix := buildFakePrefix(t, "perl", "env")
	dir := t.TempDir()
	script := filepath.Join(dir, "script.pl")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env perl\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := ScanShebang(script, prefix, "")
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.Status != InterpreterRemapped {
		t.Errorf("status: got %q, want %q", res.Status, InterpreterRemapped)
	}
}
