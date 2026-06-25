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
	"fmt"

	"github.com/spf13/cobra"
)

// NewBootstrapCmd returns a standalone cobra.Command for the android-bootstrap
// subcommand.  This allows users to explicitly run:
//
//	arduino-cli android-bootstrap [--force] [--config-file PATH]
//
// The command is intentionally simple: it delegates entirely to Bootstrap.
// It is registered on the root command in internal/cli/cli.go.
func NewBootstrapCmd() *cobra.Command {
	var (
		force      bool
		configFile string
	)

	cmd := &cobra.Command{
		Use:   "android-bootstrap",
		Short: "Bootstrap arduino-cli for Android/Termux",
		Long: `Bootstrap arduino-cli for Android/Termux.

Detects the Termux environment ($PREFIX, $HOME), creates the state directory
at $HOME/.arduino15-android, and writes a valid arduino-cli.yaml that points
all directory references to Android-writable, executable-friendly paths.

This command is idempotent: running it multiple times without --force is safe
and will not overwrite an existing configuration.

Examples:

  # First-run setup:
  arduino-cli android-bootstrap

  # Force regeneration of an existing config:
  arduino-cli android-bootstrap --force

  # Write config to a custom path:
  arduino-cli android-bootstrap --config-file /data/local/tmp/arduino-cli.yaml
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := BootstrapContext{
				ConfigPath: configFile,
				Force:      force,
			}
			if err := Bootstrap(ctx); err != nil {
				return fmt.Errorf("android bootstrap failed: %w", err)
			}

			// Report the written config path to the user.
			env, err := DetectTermuxEnvironment(ctx)
			if err == nil {
				cfgPath := configFile
				if cfgPath == "" {
					cfgPath = env.DataDir + "/arduino-cli.yaml"
				}
				cmd.Printf("Android bootstrap complete.\n")
				cmd.Printf("  Config:    %s\n", cfgPath)
				cmd.Printf("  Data dir:  %s\n", env.DataDir)
				cmd.Printf("  Downloads: %s\n", env.DownloadsDir)
				cmd.Printf("  Sketchbook:%s\n", env.UserDir)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false,
		"Overwrite an existing configuration file.")
	cmd.Flags().StringVar(&configFile, "config-file", "",
		"Path to write arduino-cli.yaml (default: $HOME/.arduino15-android/arduino-cli.yaml).")

	return cmd
}
