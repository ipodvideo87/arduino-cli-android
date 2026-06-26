# Emulated ARM64 Smoke Test Workflow

This workflow describes the best practical preflight path for ARM64-oriented
validation when native Termux is unavailable or when a quick environment check
is useful before device testing.

It is not Android validation.
It is not Termux validation.
It is not real hardware validation.

## Purpose

- detect whether the host environment is ready for automated smoke tests
- run focused repository checks on an ARM64-capable Linux environment
- build and exercise the CLI surface
- optionally compile a sketch when the required sketch and ESP32 toolchain are
  available
- produce a structured report that future workflows can compare

## Provider Modes

The bootstrap provider supports three modes:

- `inspect`
  - default
  - detect and report only
  - make no changes
- `bootstrap`
  - install missing dependencies only when explicitly requested
  - use `--install` or `ANDROID_VALIDATION_INSTALL=1`
  - still report what was installed and what remains missing
- `verify`
  - confirm the environment is correctly configured
  - make no changes
  - return a non-zero status when required capabilities are missing

Explicit installation is opt-in.
Do not install anything by default.

## Scripts

- `scripts/android/emulated-arm64-bootstrap.sh`
- `scripts/android/emulated-arm64-smoke-test.sh`

## How To Run

Inspect only:

```bash
sh scripts/android/emulated-arm64-bootstrap.sh
```

Verify without changes:

```bash
sh scripts/android/emulated-arm64-bootstrap.sh verify
```

Explicit bootstrap with installation:

```bash
sh scripts/android/emulated-arm64-bootstrap.sh bootstrap --install
```

Run the smoke test:

```bash
sh scripts/android/emulated-arm64-smoke-test.sh
```

Optional sketch selection for compile checks:

```bash
ANDROID_SMOKE_SKETCHES="$HOME/Development/Sketches/Blink:$HOME/Development/Sketches/esp32ultimate" \
  sh scripts/android/emulated-arm64-smoke-test.sh
```

## What The Smoke Test Runs

The smoke-test script should, when the prerequisites are available:

1. Collect environment readiness.
2. Build `arduino-cli`.
3. Run focused ACL and CLI tests.
4. Check command availability with `acl --help` and `acl workflow --help`.
5. Run ACL bootstrap and diagnostics checks.
6. Optionally run ACL workflow compile when a usable sketch and ESP32 core are
   available.
7. Verify the firmware package outputs when compile succeeds.

## Package Checks

When a compile succeeds, the smoke test should verify that the firmware package
contains the expected files for the mode that was produced.

At minimum, it should check for:

- `manifest.json`
- `flash-plan.json`
- `validation-report.json`
- `analysis.json`
- `README_FLASHING.txt`
- `artifacts/application.bin`
- `artifacts/firmware.elf`
- `artifacts/firmware.map`

If metadata supports full-flash packaging, also verify:

- `artifacts/bootloader.bin`

## Report Output

Every run should generate:

- a human-readable summary
- a structured JSON report

Suggested report fields:

- provider name
- mode
- detected environment
- architecture
- tool versions
- tests executed
- tests skipped
- results
- warnings
- limitations
- actions taken
- remaining recommendations
- confidence or validation level

Suggested result labels:

- `Emulated preflight only`
- `Not native Termux validation`
- `Not Android emulator validation unless proven`
- `Not real hardware validation`

## Validation Boundaries

- ARM64 / QEMU smoke tests are preflight only.
- They are useful for catching regressions in workflow orchestration, CLI
  shape, package creation, and basic build health.
- They do not prove Android behavior.
- They do not prove Termux behavior.
- They do not prove upload, flash, serial monitor, or runtime behavior.

## Expected Provider Hierarchy

This workflow should be treated as one provider in a broader validation system.
The broader hierarchy is:

1. static analysis
2. unit / integration tests
3. host smoke tests
4. ARM64 / QEMU smoke tests
5. native Termux
6. real hardware

Future providers may include:

- GitHub Actions
- desktop validation
- Android emulator validation if it becomes practical and can be proven useful
- Termux-like rootfs checks
- Bionic/sysroot compatibility checks

## Practical Notes

- Keep the scripts POSIX `sh`.
- Prefer capability detection over hardcoded distribution assumptions.
- Avoid attempting to manage `proot-distro` from inside the script; the current
  environment already provides it.
- Use explicit install only when the user asks for `bootstrap --install` or sets
  `ANDROID_VALIDATION_INSTALL=1`.

