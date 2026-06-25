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

// Package android provides Android/Termux compatibility bootstrap utilities
// for the arduino-cli-android fork.
//
// # Problem
//
// The upstream arduino-cli was designed for conventional POSIX systems where:
//
//   - A writable home directory exists at /home/<user> or /root.
//   - System libraries live under /usr/lib.
//   - Temporary storage is available at /tmp.
//
// None of these assumptions hold on Android/Termux:
//
//   - The writable home is under /data/data/com.termux/files/home.
//   - The Termux package prefix is /data/data/com.termux/files/usr ($PREFIX).
//   - /tmp is typically on a noexec filesystem; the safe temp dir is $PREFIX/tmp.
//   - SELinux enforcement restricts execution to specific filesystem regions.
//
// # Solution
//
// This package detects the Termux environment at runtime and auto-generates an
// arduino-cli.yaml that points all directory references to valid, writable,
// executable-friendly paths under $HOME/.arduino15-android.
//
// # Entry Points
//
//   - [Bootstrap] — main entry point; performs first-run initialisation.
//   - [DetectTermuxEnvironment] — resolves paths from the runtime environment.
//   - [ValidateConfig] — checks an existing config for Android compatibility.
//   - [PatchConfigPaths] — rewrites hardcoded /usr and /home paths in-place.
//
// # Integration with arduino-cli
//
// The bootstrap is triggered by the --android-bootstrap flag on the root cobra
// command (see internal/cli/cli.go).  It runs before any other command so that
// subsequent sub-commands receive a valid configuration.
//
// # Reference
//
// See docs/android/ANDROID_COMPATIBILITY_RESEARCH.md for background on Android
// filesystem constraints and the ACL project context document for architectural
// decisions.
package android
