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

// Package android provides bootstrap utilities for running arduino-cli natively
// inside Termux on Android without chroots, PRoot, or a Linux distribution layer.
//
// # Overview
//
// On Android/Termux the traditional Unix directory hierarchy does not exist at the
// expected absolute paths. In particular:
//
//   - There is no writable /home – the user home is $PREFIX/home or $HOME
//     (typically /data/data/com.termux/files/home).
//   - /usr does not exist; Termux packages install under $PREFIX
//     (typically /data/data/com.termux/files/usr).
//   - /tmp is often on a noexec filesystem; use $PREFIX/tmp instead.
//   - SELinux enforcement prevents execution from arbitrary paths.
//
// The bootstrap package detects the Termux environment, builds a valid
// arduino-cli configuration that points all directory keys to writable,
// executable-friendly paths under $HOME/.arduino15-android, and writes
// that configuration to disk if it is not already present.
//
// # Usage
//
//	ctx := android.BootstrapContext{
//	    ConfigPath: "/path/to/arduino-cli.yaml",  // optional override
//	    Force:      false,                          // set true to overwrite
//	}
//	if err := android.Bootstrap(ctx); err != nil {
//	    log.Fatal(err)
//	}
package android

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultAndroidDataDirName is the subdirectory created under $HOME that
// contains all arduino-cli state on Android/Termux.
const DefaultAndroidDataDirName = ".arduino15-android"

// BootstrapContext carries options for the Bootstrap call.
type BootstrapContext struct {
	// ConfigPath is the full path where arduino-cli.yaml will be written.
	// When empty, Bootstrap derives the path from the detected home directory:
	//   $HOME/.arduino15-android/arduino-cli.yaml
	ConfigPath string

	// Force causes Bootstrap to overwrite an existing configuration file.
	// When false (the default), Bootstrap is a no-op if the file already exists.
	Force bool

	// HomeDir overrides the home directory used to construct default paths.
	// When empty, Bootstrap resolves the home directory from the environment.
	HomeDir string

	// TermuxPrefix overrides the Termux $PREFIX used for scratch/tmp paths.
	// When empty, Bootstrap resolves it from the environment.
	TermuxPrefix string
}

// TermuxEnvironment holds resolved Termux-specific path information.
type TermuxEnvironment struct {
	// HomeDir is the writable home directory, e.g. /data/data/com.termux/files/home
	HomeDir string
	// Prefix is the Termux $PREFIX, e.g. /data/data/com.termux/files/usr
	Prefix string
	// DataDir is the base directory for all arduino-cli state files.
	DataDir string
	// DownloadsDir is the directory for downloaded content.
	DownloadsDir string
	// UserDir is the "user" (sketchbook) directory.
	UserDir string
	// TempDir is a suitable temp directory within the Termux hierarchy.
	TempDir string
}

// ArduinoCLIConfig is a minimal structured representation of the arduino-cli
// YAML configuration file. Only the fields relevant to the bootstrap are
// included; the upstream configuration loader tolerates additional keys.
type ArduinoCLIConfig struct {
	Directories struct {
		Data      string `yaml:"data"`
		Downloads string `yaml:"downloads"`
		User      string `yaml:"user"`
	} `yaml:"directories"`
	Board struct {
		AdditionalURLs []string `yaml:"additional_urls,omitempty"`
	} `yaml:"board_manager,omitempty"`
	Logging struct {
		Level  string `yaml:"level,omitempty"`
		Format string `yaml:"format,omitempty"`
	} `yaml:"logging,omitempty"`
}

// DetectTermuxEnvironment inspects environment variables and the filesystem to
// determine Termux-specific paths.  It returns an error when called outside a
// Termux environment and ctx.TermuxPrefix is empty.
func DetectTermuxEnvironment(ctx BootstrapContext) (TermuxEnvironment, error) {
	env := TermuxEnvironment{}

	// --- Resolve home directory ---
	if ctx.HomeDir != "" {
		env.HomeDir = ctx.HomeDir
	} else {
		home, err := resolveHomeDir()
		if err != nil {
			return env, fmt.Errorf("android bootstrap: cannot resolve home directory: %w", err)
		}
		env.HomeDir = home
	}

	// --- Resolve Termux prefix ---
	if ctx.TermuxPrefix != "" {
		env.Prefix = ctx.TermuxPrefix
	} else {
		prefix := os.Getenv("PREFIX")
		if prefix == "" {
			// Fallback: check the canonical Termux prefix path directly.
			candidate := "/data/data/com.termux/files/usr"
			if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
				prefix = candidate
			}
		}
		if prefix == "" {
			return env, errors.New(
				"android bootstrap: $PREFIX is not set and the default Termux prefix " +
					"(/data/data/com.termux/files/usr) does not exist; " +
					"are you running inside Termux?",
			)
		}
		env.Prefix = prefix
	}

	// --- Build arduino-cli specific sub-paths ---
	base := filepath.Join(env.HomeDir, DefaultAndroidDataDirName)
	env.DataDir = base
	env.DownloadsDir = filepath.Join(base, "staging", "packages")
	env.UserDir = filepath.Join(base, "sketchbook")
	env.TempDir = filepath.Join(env.Prefix, "tmp", "arduino-cli")

	return env, nil
}

// Bootstrap performs the full first-run initialisation:
//  1. Detects the Termux environment.
//  2. Derives default paths.
//  3. Creates required directories.
//  4. Writes arduino-cli.yaml unless it already exists (or Force is set).
//
// Bootstrap is idempotent: calling it multiple times with the same arguments
// and Force==false is safe and produces no side effects after the first call.
func Bootstrap(ctx BootstrapContext) error {
	env, err := DetectTermuxEnvironment(ctx)
	if err != nil {
		return err
	}

	// Determine config file path.
	cfgPath := ctx.ConfigPath
	if cfgPath == "" {
		cfgPath = filepath.Join(env.DataDir, "arduino-cli.yaml")
	}

	// Short-circuit if the config already exists and Force is not set.
	if !ctx.Force {
		if _, statErr := os.Stat(cfgPath); statErr == nil {
			// File exists; nothing to do.
			return nil
		}
	}

	// Create required directories.
	dirsToCreate := []string{
		env.DataDir,
		env.DownloadsDir,
		env.UserDir,
		env.TempDir,
		filepath.Dir(cfgPath),
	}
	for _, d := range dirsToCreate {
		if mkErr := os.MkdirAll(d, 0o755); mkErr != nil {
			return fmt.Errorf("android bootstrap: failed to create directory %q: %w", d, mkErr)
		}
	}

	// Build the configuration struct.
	cfg := buildConfig(env)

	// Serialise to YAML.
	data, err := marshalConfig(cfg)
	if err != nil {
		return fmt.Errorf("android bootstrap: failed to serialise config: %w", err)
	}

	// Write atomically (write to temp file, then rename).
	if err := writeFileAtomic(cfgPath, data, 0o644); err != nil {
		return fmt.Errorf("android bootstrap: failed to write config to %q: %w", cfgPath, err)
	}

	return nil
}

// ValidateConfig reads an arduino-cli.yaml file and checks that:
//  1. All three directory keys are present and non-empty.
//  2. None of the path values contain hardcoded /usr or /home prefixes that
//     would be invalid on Android.
//
// It returns a slice of human-readable validation errors; an empty slice means
// the configuration is valid.
func ValidateConfig(cfgPath string) ([]string, error) {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("android bootstrap: cannot read config %q: %w", cfgPath, err)
	}

	var cfg ArduinoCLIConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("android bootstrap: cannot parse config %q: %w", cfgPath, err)
	}

	var problems []string

	if cfg.Directories.Data == "" {
		problems = append(problems, "directories.data is empty")
	}
	if cfg.Directories.Downloads == "" {
		problems = append(problems, "directories.downloads is empty")
	}
	if cfg.Directories.User == "" {
		problems = append(problems, "directories.user is empty")
	}

	// Check for hardcoded paths that are invalid on Android.
	invalidPrefixes := []string{"/usr", "/home", "/etc", "/opt", "/var"}
	for _, field := range []struct {
		name  string
		value string
	}{
		{"directories.data", cfg.Directories.Data},
		{"directories.downloads", cfg.Directories.Downloads},
		{"directories.user", cfg.Directories.User},
	} {
		for _, bad := range invalidPrefixes {
			if strings.HasPrefix(field.value, bad) {
				problems = append(problems,
					fmt.Sprintf("%s references a hardcoded path (%q) that does not exist on Android",
						field.name, field.value),
				)
				break
			}
		}
	}

	return problems, nil
}

// PatchConfigPaths rewrites an existing arduino-cli.yaml, replacing any
// hardcoded /usr or /home path prefixes with Android-compatible equivalents
// derived from the detected Termux environment.
//
// This is intended for situations where a configuration was created by upstream
// arduino-cli tooling (e.g., `arduino-cli config init`) on a conventional Linux
// system and then copied to Android.
func PatchConfigPaths(cfgPath string, ctx BootstrapContext) error {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("android bootstrap: cannot read config for patching %q: %w", cfgPath, err)
	}

	env, err := DetectTermuxEnvironment(ctx)
	if err != nil {
		return err
	}

	content := string(data)

	// Build a replacement map: old prefix → new prefix.
	// Order matters: more specific entries first.
	replacements := []struct{ from, to string }{
		{"/home/" + guessUsername(env.HomeDir), env.HomeDir},
		{"/home", env.HomeDir},
		{"/usr/local", env.Prefix},
		{"/usr", env.Prefix},
	}

	for _, r := range replacements {
		content = strings.ReplaceAll(content, r.from, r.to)
	}

	if content == string(data) {
		// No changes required.
		return nil
	}

	return writeFileAtomic(cfgPath, []byte(content), 0o644)
}

// ---- internal helpers -------------------------------------------------------

func resolveHomeDir() (string, error) {
	// $HOME is set by Termux and is the canonical home.
	if h := os.Getenv("HOME"); h != "" {
		return h, nil
	}
	// os.UserHomeDir() falls back to /etc/passwd on Linux, which does not
	// exist on Android.  We call it anyway as a last resort.
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home, nil
}

func buildConfig(env TermuxEnvironment) ArduinoCLIConfig {
	var cfg ArduinoCLIConfig
	cfg.Directories.Data = env.DataDir
	cfg.Directories.Downloads = env.DownloadsDir
	cfg.Directories.User = env.UserDir
	cfg.Logging.Level = "warn"
	cfg.Logging.Format = "text"
	return cfg
}

func marshalConfig(cfg ArduinoCLIConfig) ([]byte, error) {
	var sb strings.Builder
	sb.WriteString("# arduino-cli configuration — generated by android bootstrap\n")
	sb.WriteString("# Do not add hardcoded /usr or /home paths; they do not exist on Android.\n")
	sb.WriteString("#\n")

	body, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	sb.Write(body)
	return []byte(sb.String()), nil
}

// writeFileAtomic writes data to path by first writing a temporary sibling
// file and then renaming it into place.  This prevents a partially-written
// file from being observed by concurrent readers.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".arduino-cli-bootstrap-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	// Ensure the temp file is removed on any error path.
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	ok = true
	return nil
}

// guessUsername extracts a probable POSIX username from a home directory path
// such as /data/data/com.termux/files/home or /home/alice → "alice".
// Returns an empty string when the pattern is unrecognised.
func guessUsername(homeDir string) string {
	// Strip trailing slash.
	homeDir = strings.TrimRight(homeDir, "/")
	// Common patterns:
	//   /home/<user>
	//   /data/data/com.termux/files/home  (Termux — no username component)
	parts := strings.Split(homeDir, "/")
	if len(parts) >= 2 {
		last := parts[len(parts)-1]
		secondLast := parts[len(parts)-2]
		if secondLast == "home" && last != "home" {
			return last
		}
	}
	return ""
}
