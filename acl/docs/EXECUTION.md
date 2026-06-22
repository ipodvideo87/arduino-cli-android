# ACL Execution

ACL execution starts with planning, not launch. `acl-exec` classifies the target ELF
and then chooses between direct execution and an explicit ACL loader strategy.

## Modes

- dry-run: plan only, no execution
- `--apply`: explicit experimental execution path

## Decision Matrix

`acl-exec` now makes the target class explicit:

- `android-native-elf` and `linux-direct-elf` use direct kernel execution
- `rust-launcher` uses direct kernel execution so the wrapper keeps its own identity
- `patched-linux-elf` uses the explicit ACL loader strategy

The direct path is the default for Rust launcher wrappers and Android-native binaries.
Only Linux ELF binaries that actually need the copied ACL runtime go through the loader.

## What the Plan Contains

- target path and ELF metadata
- target class and launch mode
- active runtime data when the explicit-loader path is selected
- loader path and runtime library paths when required
- argv
- cwd
- environment additions
- warnings, errors, and whether execution is allowed

## Diagnostics

`acl-exec` prints a structured diagnostic report for both dry-run and `--apply`.
That report is meant to support:

- `native Termux detected` evidence when the session is running in Termux with Termux paths and no proot/chroot markers
- `PRoot/proot-distro detected` evidence when proot-style markers or rootfs evidence are present
- `environment unknown` when the session does not provide enough evidence to classify it

The diagnostic report includes the target, selected strategy, runtime information,
sanitized environment audit, execution result, and likely-cause hints. It is designed
to make native failures actionable without mixing ad hoc prints into the execution code.

## Current Status

Execution planning exists, and the first real execution backend now exists behind
`--apply`. The backend is intentionally narrow in scope. It now executes Rust launcher
wrappers and Android-native ELF binaries directly, while reserving the ACL loader for
patched Linux ELF binaries that still require it.

A successful dry run does not prove Android-native compatibility.

## Apply Mode

When `--apply` is used, `acl-exec` currently:

- keeps the selected runtime only when the explicit-loader path is needed
- validates the target, runtime, loader, library search path, and cwd before launch
- executes direct-target binaries without forcing them through the loader
- captures stdout, stderr, and exit status where practical

This strategy avoids the `/proc/self/exe` identity problem that broke the ESP32 Rust
launcher wrappers under explicit loader invocation.

The selected runtime still must be self-contained enough to satisfy the loader's
library lookups on Android when the loader strategy is chosen. A copied Termux loader
that still falls back to the original glibc tree can still fail before the target binary
itself starts, but that is now limited to explicit-loader launches.

This is the first execution backend, not final proof that Linux-oriented tooling works
correctly on Android.

Native diagnostics also showed that some launcher failures were not execution-planner
bugs at all. The packaging path can still strip execute bits when it rewrites source
assets, so ACL now preserves source file modes during runtime packaging and runtime
installation, and validate-compat reports executable or script candidates that arrive
without `+x`.

## Validation Guidance

Android execution must be validated from a fresh Termux clone outside proot. The current
container environment is useful for testing the planner, command construction, and
failure handling, but it is not proof of device-native execution.

Successful execution inside proot does not prove Android-native compatibility.
