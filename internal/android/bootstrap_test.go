// This file is part of arduino-cli.
//
// Copyright 2024 ARDUINO SA (https://www.arduino.cc/)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package android

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newFakeTermux creates a temporary directory tree that mimics Termux's
// on-device layout and returns the simulated HOME and PREFIX paths.
func newFakeTermux(t *testing.T) (home, prefix string) {
	t.Helper()
	root := t.TempDir()
	home = filepath.Join(root, "home")
	prefix = filepath.Join(root, "usr")
	for _, d := range []string{home, prefix, filepath.Join(prefix, "tmp")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("setup: MkdirAll(%q): %v", d, err)
		}
	}
	return home, prefix
}

// makeContext builds a BootstrapContext wired to the fake Termux tree.
func makeContext(home, prefix string) BootstrapContext {
	return BootstrapContext{
		HomeDir:      home,
		TermuxPrefix: prefix,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// DetectTermuxEnvironment
// ────────────────────────────────────────────────────────────────────────────

func TestDetectTermuxEnvironment_HappyPath(t *testing.T) {
	home, prefix := newFakeTermux(t)
	ctx := makeContext(home, prefix)

	env, err := DetectTermuxEnvironment(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if env.HomeDir != home {
		t.Errorf("HomeDir: got %q, want %q", env.HomeDir, home)
	}
	if env.Prefix != prefix {
		t.Errorf("Prefix: got %q, want %q", env.Prefix, prefix)
	}

	wantDataDir := filepath.Join(home, DefaultAndroidDataDirName)
	if env.DataDir != wantDataDir {
		t.Errorf("DataDir: got %q, want %q", env.DataDir, wantDataDir)
	}

	wantDownloads := filepath.Join(wantDataDir, "staging", "packages")
	if env.DownloadsDir != wantDownloads {
		t.Errorf("DownloadsDir: got %q, want %q", env.DownloadsDir, wantDownloads)
	}

	wantUser := filepath.Join(wantDataDir, "sketchbook")
	if env.UserDir != wantUser {
		t.Errorf("UserDir: got %q, want %q", env.UserDir, wantUser)
	}
}

func TestDetectTermuxEnvironment_MissingPrefix(t *testing.T) {
	home, _ := newFakeTermux(t)
	ctx := BootstrapContext{HomeDir: home} // no prefix, no $PREFIX env var

	// Unset $PREFIX for this test in case the test runner happens to have it.
	t.Setenv("PREFIX", "")

	_, err := DetectTermuxEnvironment(ctx)
	if err == nil {
		t.Fatal("expected error when PREFIX is missing, got nil")
	}
	if !strings.Contains(err.Error(), "PREFIX") {
		t.Errorf("error should mention PREFIX, got: %v", err)
	}
}

func TestDetectTermuxEnvironment_PrefixFromEnv(t *testing.T) {
	home, prefix := newFakeTermux(t)
	t.Setenv("PREFIX", prefix)

	ctx := BootstrapContext{HomeDir: home} // prefix taken from env

	env, err := DetectTermuxEnvironment(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Prefix != prefix {
		t.Errorf("Prefix: got %q, want %q", env.Prefix, prefix)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Bootstrap
// ────────────────────────────────────────────────────────────────────────────

func TestBootstrap_CreatesConfigOnFirstRun(t *testing.T) {
	home, prefix := newFakeTermux(t)
	ctx := makeContext(home, prefix)

	if err := Bootstrap(ctx); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	cfgPath := filepath.Join(home, DefaultAndroidDataDirName, "arduino-cli.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("config file not created at %q: %v", cfgPath, err)
	}

	content := string(data)
	for _, want := range []string{
		"data:",
		"downloads:",
		"user:",
		DefaultAndroidDataDirName,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("config missing %q:\n%s", want, content)
		}
	}
}

func TestBootstrap_IdempotentWithoutForce(t *testing.T) {
	home, prefix := newFakeTermux(t)
	ctx := makeContext(home, prefix)

	if err := Bootstrap(ctx); err != nil {
		t.Fatalf("first Bootstrap failed: %v", err)
	}

	cfgPath := filepath.Join(home, DefaultAndroidDataDirName, "arduino-cli.yaml")
	info1, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("stat after first Bootstrap: %v", err)
	}

	// Second call — should be a no-op.
	if err := Bootstrap(ctx); err != nil {
		t.Fatalf("second Bootstrap failed: %v", err)
	}

	info2, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("stat after second Bootstrap: %v", err)
	}

	if info1.ModTime() != info2.ModTime() {
		t.Error("config file was re-written on second Bootstrap without Force=true")
	}
}

func TestBootstrap_ForceOverwrites(t *testing.T) {
	home, prefix := newFakeTermux(t)
	ctx := makeContext(home, prefix)

	if err := Bootstrap(ctx); err != nil {
		t.Fatalf("first Bootstrap failed: %v", err)
	}

	cfgPath := filepath.Join(home, DefaultAndroidDataDirName, "arduino-cli.yaml")
	// Corrupt the file.
	if err := os.WriteFile(cfgPath, []byte("garbage: true\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ctx.Force = true
	if err := Bootstrap(ctx); err != nil {
		t.Fatalf("Bootstrap with Force failed: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), "garbage") {
		t.Error("config still contains 'garbage' after Force bootstrap")
	}
}

func TestBootstrap_CustomConfigPath(t *testing.T) {
	home, prefix := newFakeTermux(t)
	customCfg := filepath.Join(t.TempDir(), "subdir", "my-cli.yaml")
	ctx := BootstrapContext{
		HomeDir:      home,
		TermuxPrefix: prefix,
		ConfigPath:   customCfg,
	}

	if err := Bootstrap(ctx); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	if _, err := os.Stat(customCfg); err != nil {
		t.Fatalf("expected config at custom path %q: %v", customCfg, err)
	}
}

func TestBootstrap_CreatesRequiredDirectories(t *testing.T) {
	home, prefix := newFakeTermux(t)
	ctx := makeContext(home, prefix)

	if err := Bootstrap(ctx); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	base := filepath.Join(home, DefaultAndroidDataDirName)
	for _, want := range []string{
		base,
		filepath.Join(base, "staging", "packages"),
		filepath.Join(base, "sketchbook"),
		filepath.Join(prefix, "tmp", "arduino-cli"),
	} {
		fi, err := os.Stat(want)
		if err != nil {
			t.Errorf("expected directory %q to exist: %v", want, err)
			continue
		}
		if !fi.IsDir() {
			t.Errorf("expected %q to be a directory", want)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// ValidateConfig
// ────────────────────────────────────────────────────────────────────────────

func TestValidateConfig_ValidConfig(t *testing.T) {
	home, prefix := newFakeTermux(t)
	ctx := makeContext(home, prefix)

	if err := Bootstrap(ctx); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	cfgPath := filepath.Join(home, DefaultAndroidDataDirName, "arduino-cli.yaml")
	problems, err := ValidateConfig(cfgPath)
	if err != nil {
		t.Fatalf("ValidateConfig error: %v", err)
	}
	if len(problems) != 0 {
		t.Errorf("ValidateConfig found problems in a freshly bootstrapped config: %v", problems)
	}
}

func TestValidateConfig_HardcodedUsrPath(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "arduino-cli.yaml")
	content := `
directories:
  data: /usr/share/arduino
  downloads: /usr/local/arduino/downloads
  user: /home/user/Arduino
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	problems, err := ValidateConfig(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) == 0 {
		t.Fatal("expected validation problems for hardcoded /usr paths, got none")
	}

	// Expect problems about hardcoded paths.
	combined := strings.Join(problems, "\n")
	if !strings.Contains(combined, "hardcoded") {
		t.Errorf("expected 'hardcoded' in problems, got:\n%s", combined)
	}
}

func TestValidateConfig_EmptyFields(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "arduino-cli.yaml")
	content := `
directories:
  data: ""
  downloads: ""
  user: ""
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	problems, err := ValidateConfig(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) < 3 {
		t.Errorf("expected at least 3 problems for empty directory fields, got %d: %v", len(problems), problems)
	}
}

func TestValidateConfig_MissingFile(t *testing.T) {
	_, err := ValidateConfig("/nonexistent/path/arduino-cli.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// PatchConfigPaths
// ────────────────────────────────────────────────────────────────────────────

func TestPatchConfigPaths_ReplacesHardcodedPaths(t *testing.T) {
	home, prefix := newFakeTermux(t)
	ctx := makeContext(home, prefix)

	cfgPath := filepath.Join(t.TempDir(), "arduino-cli.yaml")
	// Write a config that might have been generated by upstream arduino-cli on
	// a standard Linux system.
	upstream := `
directories:
  data: /home/user/.arduino15
  downloads: /home/user/.arduino15/staging/packages
  user: /home/user/Arduino
`
	if err := os.WriteFile(cfgPath, []byte(upstream), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := PatchConfigPaths(cfgPath, ctx); err != nil {
		t.Fatalf("PatchConfigPaths failed: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile after patch: %v", err)
	}
	patched := string(data)

	if strings.Contains(patched, "/home/user") {
		t.Errorf("patched config still contains /home/user:\n%s", patched)
	}
	if !strings.Contains(patched, home) {
		t.Errorf("patched config does not contain home dir %q:\n%s", home, patched)
	}
}

func TestPatchConfigPaths_NoOpWhenAlreadyClean(t *testing.T) {
	home, prefix := newFakeTermux(t)
	ctx := makeContext(home, prefix)

	// Bootstrap a clean config first.
	bootCtx := makeContext(home, prefix)
	if err := Bootstrap(bootCtx); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}
	cfgPath := filepath.Join(home, DefaultAndroidDataDirName, "arduino-cli.yaml")

	info1, _ := os.Stat(cfgPath)

	// Patch should be a no-op.
	if err := PatchConfigPaths(cfgPath, ctx); err != nil {
		t.Fatalf("PatchConfigPaths failed: %v", err)
	}

	info2, _ := os.Stat(cfgPath)
	if info1.ModTime() != info2.ModTime() {
		t.Error("PatchConfigPaths modified a file that needed no changes")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// guessUsername (internal helper)
// ────────────────────────────────────────────────────────────────────────────

func TestGuessUsername(t *testing.T) {
	cases := []struct {
		home string
		want string
	}{
		{"/home/alice", "alice"},
		{"/home/bob/", "bob"},
		{"/data/data/com.termux/files/home", ""},
		{"/root", ""},
		{"/Users/charlie", ""},
	}
	for _, tc := range cases {
		got := guessUsername(tc.home)
		if got != tc.want {
			t.Errorf("guessUsername(%q) = %q, want %q", tc.home, got, tc.want)
		}
	}
}
