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

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arduino/arduino-cli/internal/android"
	"github.com/spf13/cobra"
)

// buildRootCmd constructs a minimal root cobra.Command for testing that has
// the Android bootstrap registered on it.
func buildRootCmd(t *testing.T) (*cobra.Command, string) {
	t.Helper()

	home := t.TempDir()
	prefix := filepath.Join(t.TempDir(), "usr")
	if err := os.MkdirAll(filepath.Join(prefix, "tmp"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Set environment variables so that DetectTermuxEnvironment can resolve them.
	t.Setenv("HOME", home)
	t.Setenv("PREFIX", prefix)

	root := &cobra.Command{
		Use:          "arduino-cli",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	// Register the bootstrap without a config path resolver (derives from env).
	RegisterAndroidBootstrap(root, nil)

	return root, home
}

func TestRegisterAndroidBootstrap_FlagsPresentOnRoot(t *testing.T) {
	root, _ := buildRootCmd(t)

	if root.PersistentFlags().Lookup(android.FlagName) == nil {
		t.Errorf("expected --%s flag on root command", android.FlagName)
	}
	if root.PersistentFlags().Lookup(android.FlagForce) == nil {
		t.Errorf("expected --%s flag on root command", android.FlagForce)
	}
}

func TestRegisterAndroidBootstrap_SubCommandPresent(t *testing.T) {
	root, _ := buildRootCmd(t)

	var found bool
	for _, sub := range root.Commands() {
		if sub.Use == "android-bootstrap" {
			found = true
			break
		}
	}
	if !found {
		t.Error("android-bootstrap sub-command not found on root")
	}
}

func TestRegisterAndroidBootstrap_FlagTriggersBootstrap(t *testing.T) {
	root, home := buildRootCmd(t)

	// Execute the root command with --android-bootstrap.
	root.SetArgs([]string{"--" + android.FlagName})
	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute failed: %v", err)
	}

	cfgPath := filepath.Join(home, android.DefaultAndroidDataDirName, "arduino-cli.yaml")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("expected config at %q after --android-bootstrap: %v", cfgPath, err)
	}
}

func TestRegisterAndroidBootstrap_SubCommandCreatesConfig(t *testing.T) {
	root, home := buildRootCmd(t)

	root.SetArgs([]string{"android-bootstrap"})
	if err := root.Execute(); err != nil {
		t.Fatalf("android-bootstrap sub-command failed: %v", err)
	}

	cfgPath := filepath.Join(home, android.DefaultAndroidDataDirName, "arduino-cli.yaml")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("config not created at %q: %v", cfgPath, err)
	}

	problems, err := android.ValidateConfig(cfgPath)
	if err != nil {
		t.Fatalf("ValidateConfig: %v", err)
	}
	if len(problems) != 0 {
		t.Errorf("config created by sub-command has problems: %v", problems)
	}
}

func TestRegisterAndroidBootstrap_PreservesExistingPersistentPreRunE(t *testing.T) {
	home := t.TempDir()
	prefix := filepath.Join(t.TempDir(), "usr")
	if err := os.MkdirAll(filepath.Join(prefix, "tmp"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PREFIX", prefix)

	preRunCalled := false
	root := &cobra.Command{
		Use:          "arduino-cli",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			preRunCalled = true
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	RegisterAndroidBootstrap(root, nil)

	root.SetArgs([]string{"--" + android.FlagName})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !preRunCalled {
		t.Error("existing PersistentPreRunE was not called after RegisterAndroidBootstrap")
	}
}
