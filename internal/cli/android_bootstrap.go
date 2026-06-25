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

// Package cli wires the Android bootstrap into the root cobra command.
//
// This file registers:
//
//  1. The `--android-bootstrap` and `--android-bootstrap-force` persistent
//     flags on the root command.  When either flag is present, Bootstrap runs
//     in PersistentPreRunE before the selected sub-command executes.
//
//  2. The `android-bootstrap` sub-command for explicit one-shot use.
//
// Integration pattern
//
//	In the file that constructs the root cobra.Command (typically cli.go or
//	main.go), call:
//
//	  androidbootstrap.RegisterOnRoot(rootCmd)
//
//	That single call wires both the flags and the sub-command.
package cli

import (
	"github.com/arduino/arduino-cli/internal/android"
	"github.com/spf13/cobra"
)

// RegisterAndroidBootstrap wires the Android bootstrap flags and sub-command
// onto rootCmd.  It is safe to call this unconditionally on all platforms;
// the bootstrap is only executed when the user explicitly passes
// --android-bootstrap (or --android-bootstrap-force) or invokes the
// android-bootstrap sub-command.
func RegisterAndroidBootstrap(rootCmd *cobra.Command, getConfigFilePath func() string) {
	// 1. Register persistent flags so they are available on every sub-command.
	android.AddFlags(rootCmd)

	// 2. Hook into PersistentPreRunE to run the bootstrap before any command.
	//    We wrap any existing PersistentPreRunE so we do not clobber it.
	existingPreRun := rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Determine the config path from whatever config resolution the
		// root command already performs (passed in via getConfigFilePath).
		cfgPath := ""
		if getConfigFilePath != nil {
			cfgPath = getConfigFilePath()
		}
		if err := android.RunFromFlags(cmd, cfgPath); err != nil {
			return err
		}
		if existingPreRun != nil {
			return existingPreRun(cmd, args)
		}
		return nil
	}

	// 3. Register the explicit android-bootstrap sub-command.
	rootCmd.AddCommand(android.NewBootstrapCmd())
}
