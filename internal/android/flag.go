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

import "github.com/spf13/cobra"

// FlagName is the cobra flag that triggers Android/Termux bootstrap.
const FlagName = "android-bootstrap"

// FlagForce is the cobra flag that forces re-generation of the config even
// when one already exists.
const FlagForce = "android-bootstrap-force"

// AddFlags registers the Android bootstrap flags onto cmd.
// Call this from the root command's init/setup function.
func AddFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().Bool(
		FlagName,
		false,
		"Bootstrap arduino-cli for Android/Termux: auto-detect $PREFIX, "+
			"create $HOME/.arduino15-android, and write a valid arduino-cli.yaml. "+
			"Idempotent — safe to run on every invocation.",
	)
	cmd.PersistentFlags().Bool(
		FlagForce,
		false,
		"Force re-generation of the Android bootstrap configuration even "+
			"if arduino-cli.yaml already exists (implies --"+FlagName+").",
	)
}

// RunFromFlags reads the bootstrap-related flags from cmd and, if enabled,
// executes the Bootstrap operation.  configPath may be empty; the bootstrap
// will then derive the config path from the detected Termux home directory.
//
// RunFromFlags is intended to be called at the top of a cobra PersistentPreRunE
// or from the root command's Execute wrapper.
func RunFromFlags(cmd *cobra.Command, configPath string) error {
	force, _ := cmd.Flags().GetBool(FlagForce)
	enabled, _ := cmd.Flags().GetBool(FlagName)

	if !enabled && !force {
		return nil
	}

	ctx := BootstrapContext{
		ConfigPath: configPath,
		Force:      force,
	}
	return Bootstrap(ctx)
}
