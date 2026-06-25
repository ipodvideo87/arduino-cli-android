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

package android_integration

// This file contains shared test setup for the android integration test
// package.  Test cases live in bootstrap_integration_test.go.
//
// All helpers and fixtures are local to this package; no external test
// infrastructure (e.g., testutil, require) is imported so that this package
// remains runnable with a bare `go test ./...` without extra dependencies.
