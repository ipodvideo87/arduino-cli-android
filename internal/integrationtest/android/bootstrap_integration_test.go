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

// Package android_integration contains integration tests for the
// internal/android bootstrap package.
//
// These tests exercise the full Bootstrap → ValidateConfig round-trip using
// a realistic (but sandboxed) fake Termux directory layout.  They do NOT
// require a real Android device or Termux installation; all paths are
// redirected to t.TempDir() sandboxes.
package android_integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arduino/arduino-cli/internal/android"
	"gopkg.in/yaml.v3"
)

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// fakeTermux constructs a minimal directory tree that mimics the on-device
// Termux layout inside a t.TempDir() sandbox and returns the fake home and
// prefix paths.
//
//	<root>/data/data/com.termux/files/home   ← fake $HOME
//	<root>/data/data/com.termux/files/usr    ← fake $PREFIX
func fakeTermux(t *testing.T) (home, prefix string) {
	t.Helper()
	root := t.TempDir()
	home = filepath.Join(root, "data", "data", "com.termux", "files", "home")
	prefix = filepath.Join(root, "data", "data", "com.termux", "files", "usr")
	for _, d := range []string{
		home,
		prefix,
		filepath.Join(prefix, "tmp"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("fakeTermux: MkdirAll(%q): %v", d, err)
		}
	}
	return home, prefix
}

// makeCtx returns a BootstrapContext wired to the given fake Termux tree.
func makeCtx(home, prefix string) android.BootstrapContext {
	return android.BootstrapContext{
		HomeDir:      home,
		TermuxPrefix: prefix,
	}
}

// readYAMLMap reads a YAML file into a generic map for inspection without
// importing the full arduino-cli config package.
func readYAMLMap(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readYAMLMap: ReadFile(%q): %v", path, err)
	}
	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("readYAMLMap: yaml.Unmarshal: %v", err)
	}
	return m
}

// getNestedString walks a YAML map using a dot-separated key path such as
// "directories.data".
func getNestedString(m map[string]interface{}, keyPath string) string {
	keys := strings.SplitN(keyPath, ".", 2)
	val, ok := m[keys[0]]
	if !ok {
		return ""
	}
	if len(keys) == 1 {
		if s, ok := val.(string); ok {
			return s
		}
		return ""
	}
	if sub, ok := val.(map[string]interface{}); ok {
		return getNestedString(sub, keys[1])
	}
	return ""
}

// ─────────────────────────────────────────────────────────────────────────────
// Test: Bootstrap creates the config file
// ─────────────────────────────────────────────────────────────────────────────

func TestIntegration_Bootstrap_CreatesValidConfigFile(t *testing.T) {
	home, prefix := fakeTermux(t)
	ctx := makeCtx(home, prefix)

	if err := android.Bootstrap(ctx); err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}

	expectedCfgPath := filepath.Join(home, android.DefaultAndroidDataDirName, "arduino-cli.yaml")
	if _, err := os.Stat(expectedCfgPath); err != nil {
		t.Fatalf("expected config file at %q: %v", expectedCfgPath, err)
	}
	t.Logf("config written to: %s", expectedCfgPath)
}

// ─────────────────────────────────────────────────────────────────────────────
// Test: Bootstrap → ValidateConfig round-trip
// ─────────────────────────────────────────────────────────────────────────────

func TestIntegration_Bootstrap_ThenValidate_NoProblems(t *testing.T) {
	home, prefix := fakeTermux(t)
	ctx := makeCtx(home, prefix)

	if err := android.Bootstrap(ctx); err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}

	cfgPath := filepath.Join(home, android.DefaultAndroidDataDirName, "arduino-cli.yaml")
	problems, err := android.ValidateConfig(cfgPath)
	if err != nil {
		t.Fatalf("ValidateConfig returned error: %v", err)
	}
	if len(problems) != 0 {
		t.Errorf("ValidateConfig found %d problem(s) in freshly bootstrapped config:", len(problems))
		for _, p := range problems {
			t.Errorf("  • %s", p)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test: Config YAML content correctness
// ─────────────────────────────────────────────────────────────────────────────

func TestIntegration_Bootstrap_ConfigDirectoriesPointToAndroidPaths(t *testing.T) {
	home, prefix := fakeTermux(t)
	ctx := makeCtx(home, prefix)

	if err := android.Bootstrap(ctx); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	cfgPath := filepath.Join(home, android.DefaultAndroidDataDirName, "arduino-cli.yaml")
	m := readYAMLMap(t, cfgPath)

	dirData := getNestedString(m, "directories.data")
	dirDownloads := getNestedString(m, "directories.downloads")
	dirUser := getNestedString(m, "directories.user")

	// All three directory values must be present.
	if dirData == "" {
		t.Error("directories.data is empty in bootstrapped config")
	}
	if dirDownloads == "" {
		t.Error("directories.downloads is empty in bootstrapped config")
	}
	if dirUser == "" {
		t.Error("directories.user is empty in bootstrapped config")
	}

	// None of the paths should reference /usr or /home.
	for name, val := range map[string]string{
		"directories.data":      dirData,
		"directories.downloads": dirDownloads,
		"directories.user":      dirUser,
	} {
		for _, bad := range []string{"/usr", "/home"} {
			if strings.HasPrefix(val, bad) {
				t.Errorf("%s = %q starts with disallowed prefix %q", name, val, bad)
			}
		}
	}

	// Values must be rooted under the fake home.
	for name, val := range map[string]string{
		"directories.data":      dirData,
		"directories.downloads": dirDownloads,
		"directories.user":      dirUser,
	} {
		if !strings.HasPrefix(val, home) {
			t.Errorf("%s = %q does not start with home dir %q", name, val, home)
		}
	}

	// All paths must reference the Android-specific sub-directory name.
	for name, val := range map[string]string{
		"directories.data":      dirData,
		"directories.downloads": dirDownloads,
		"directories.user":      dirUser,
	} {
		if !strings.Contains(val, android.DefaultAndroidDataDirName) {
			t.Errorf("%s = %q does not contain %q", name, val, android.DefaultAndroidDataDirName)
		}
	}

	t.Logf("data=%s downloads=%s user=%s prefix_used=%s", dirData, dirDownloads, dirUser, prefix)
}

// ─────────────────────────────────────────────────────────────────────────────
// Test: Required directories are created on disk
// ─────────────────────────────────────────────────────────────────────────────

func TestIntegration_Bootstrap_CreatesAllRequiredDirectories(t *testing.T) {
	home, prefix := fakeTermux(t)
	ctx := makeCtx(home, prefix)

	if err := android.Bootstrap(ctx); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	base := filepath.Join(home, android.DefaultAndroidDataDirName)
	expectedDirs := []string{
		base,
		filepath.Join(base, "staging", "packages"),
		filepath.Join(base, "sketchbook"),
		filepath.Join(prefix, "tmp", "arduino-cli"),
	}

	for _, dir := range expectedDirs {
		fi, err := os.Stat(dir)
		if err != nil {
			t.Errorf("expected directory %q after Bootstrap: %v", dir, err)
			continue
		}
		if !fi.IsDir() {
			t.Errorf("expected %q to be a directory, got mode %v", dir, fi.Mode())
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test: Bootstrap is idempotent
// ─────────────────────────────────────────────────────────────────────────────

func TestIntegration_Bootstrap_IsIdempotent(t *testing.T) {
	home, prefix := fakeTermux(t)
	ctx := makeCtx(home, prefix)

	for i := 0; i < 3; i++ {
		if err := android.Bootstrap(ctx); err != nil {
			t.Fatalf("Bootstrap iteration %d failed: %v", i+1, err)
		}
	}

	cfgPath := filepath.Join(home, android.DefaultAndroidDataDirName, "arduino-cli.yaml")
	problems, err := android.ValidateConfig(cfgPath)
	if err != nil {
		t.Fatalf("ValidateConfig after 3 Bootstrap calls: %v", err)
	}
	if len(problems) != 0 {
		t.Errorf("config invalid after repeated Bootstrap calls: %v", problems)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test: Force overwrite restores a corrupted config
// ─────────────────────────────────────────────────────────────────────────────

func TestIntegration_Bootstrap_ForceOverwritesCorruptedConfig(t *testing.T) {
	home, prefix := fakeTermux(t)
	ctx := makeCtx(home, prefix)

	// Normal first bootstrap.
	if err := android.Bootstrap(ctx); err != nil {
		t.Fatalf("initial Bootstrap failed: %v", err)
	}

	cfgPath := filepath.Join(home, android.DefaultAndroidDataDirName, "arduino-cli.yaml")

	// Corrupt the config with hardcoded paths.
	corrupt := `
directories:
  data: /usr/share/arduino
  downloads: /usr/local/share/arduino/staging
  user: /home/user/Arduino
logging:
  level: debug
`
	if err := os.WriteFile(cfgPath, []byte(corrupt), 0o644); err != nil {
		t.Fatalf("WriteFile corrupt config: %v", err)
	}

	// Confirm it is now invalid.
	problemsBefore, err := android.ValidateConfig(cfgPath)
	if err != nil {
		t.Fatalf("ValidateConfig on corrupt config: %v", err)
	}
	if len(problemsBefore) == 0 {
		t.Fatal("expected validation problems on corrupt config")
	}

	// Force re-bootstrap.
	ctx.Force = true
	if err := android.Bootstrap(ctx); err != nil {
		t.Fatalf("forced Bootstrap failed: %v", err)
	}

	// Should now be valid.
	problemsAfter, err := android.ValidateConfig(cfgPath)
	if err != nil {
		t.Fatalf("ValidateConfig after forced Bootstrap: %v", err)
	}
	if len(problemsAfter) != 0 {
		t.Errorf("config still invalid after forced Bootstrap: %v", problemsAfter)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test: PatchConfigPaths makes an upstream Linux config valid on Android
// ─────────────────────────────────────────────────────────────────────────────

func TestIntegration_PatchConfigPaths_MakesUpstreamConfigValid(t *testing.T) {
	home, prefix := fakeTermux(t)

	// Simulate a config generated by `arduino-cli config init` on a Linux PC
	// and then copied to an Android device.
	cfgPath := filepath.Join(t.TempDir(), "arduino-cli.yaml")
	upstreamConfig := `
directories:
  data: /home/developer/.arduino15
  downloads: /home/developer/.arduino15/staging/packages
  user: /home/developer/Arduino
board_manager:
  additional_urls: []
logging:
  level: warn
  format: text
`
	if err := os.WriteFile(cfgPath, []byte(upstreamConfig), 0o644); err != nil {
		t.Fatalf("WriteFile upstream config: %v", err)
	}

	ctx := android.BootstrapContext{
		HomeDir:      home,
		TermuxPrefix: prefix,
	}

	// Validate before patch — should have problems.
	problemsBefore, err := android.ValidateConfig(cfgPath)
	if err != nil {
		t.Fatalf("ValidateConfig before patch: %v", err)
	}
	if len(problemsBefore) == 0 {
		t.Fatal("expected validation problems on upstream Linux config before patching")
	}
	t.Logf("problems before patch (%d): %v", len(problemsBefore), problemsBefore)

	// Patch.
	if err := android.PatchConfigPaths(cfgPath, ctx); err != nil {
		t.Fatalf("PatchConfigPaths failed: %v", err)
	}

	// Read back and confirm.
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile after patch: %v", err)
	}
	patched := string(data)

	if strings.Contains(patched, "/home/developer") {
		t.Errorf("patched config still contains /home/developer:\n%s", patched)
	}
	if !strings.Contains(patched, home) {
		t.Errorf("patched config does not reference Termux home %q:\n%s", home, patched)
	}

	t.Logf("patched config:\n%s", patched)
}

// ─────────────────────────────────────────────────────────────────────────────
// Test: Custom config path
// ─────────────────────────────────────────────────────────────────────────────

func TestIntegration_Bootstrap_CustomConfigPath(t *testing.T) {
	home, prefix := fakeTermux(t)
	customDir := filepath.Join(t.TempDir(), "custom", "config", "dir")
	customCfg := filepath.Join(customDir, "my-arduino-cli.yaml")

	ctx := android.BootstrapContext{
		HomeDir:      home,
		TermuxPrefix: prefix,
		ConfigPath:   customCfg,
	}

	if err := android.Bootstrap(ctx); err != nil {
		t.Fatalf("Bootstrap with custom path failed: %v", err)
	}

	if _, err := os.Stat(customCfg); err != nil {
		t.Fatalf("config not found at custom path %q: %v", customCfg, err)
	}

	problems, err := android.ValidateConfig(customCfg)
	if err != nil {
		t.Fatalf("ValidateConfig: %v", err)
	}
	if len(problems) != 0 {
		t.Errorf("custom-path config has validation problems: %v", problems)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test: Without Termux, Bootstrap returns a clear error
// ─────────────────────────────────────────────────────────────────────────────

func TestIntegration_Bootstrap_FailsGracefullyOutsideTermux(t *testing.T) {
	// Unset $PREFIX so DetectTermuxEnvironment has no fallback.
	t.Setenv("PREFIX", "")

	ctx := android.BootstrapContext{
		HomeDir: t.TempDir(),
		// TermuxPrefix intentionally left empty.
	}

	err := android.Bootstrap(ctx)
	if err == nil {
		t.Fatal("expected an error when running outside Termux (no PREFIX), got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "prefix") {
		t.Errorf("expected error message to mention PREFIX, got: %v", err)
	}
	t.Logf("got expected error: %v", err)
}
