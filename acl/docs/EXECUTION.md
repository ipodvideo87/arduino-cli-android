# ACL Execution

ACL execution starts with planning, not launch. `acl-exec` inspects a target ELF binary,
selects the active runtime, and prints the execution plan by default.

## Modes

- dry-run: plan only, no execution
- `--apply`: explicit experimental execution path

## What the Plan Contains

- target path and ELF metadata
- active runtime ID and path
- loader path
- runtime library paths
- argv
- cwd
- environment additions
- warnings, errors, and whether execution is allowed

## Current Status

Execution planning exists, and the first real execution backend now exists behind
`--apply`. This backend is still experimental and intentionally narrow in scope. It
attempts explicit loader-based execution using the selected active runtime rather than
claiming transparent Android compatibility.

A successful dry run does not prove Android-native compatibility.

## Apply Mode

When `--apply` is used, `acl-exec` currently:

- requires a valid active runtime
- validates the target, loader, library search path, and cwd before launch
- invokes the selected runtime loader explicitly
- passes `--library-path` using the ACL runtime library directories
- passes the target executable and argv through the loader command
- captures stdout, stderr, and exit status where practical

The selected runtime must already be self-contained enough to satisfy the loader's
library lookups on Android. A copied Termux loader that still falls back to the
original glibc tree can reach `EACCES` or `invalid ELF header` failures before the
target binary itself ever starts.

This is the first execution backend, not final proof that Linux-oriented tooling works
correctly on Android.

## Validation Guidance

Android execution must be validated from a fresh Termux clone outside proot. The current
container environment is useful for testing the planner, command construction, and
failure handling, but it is not proof of device-native execution.

Successful execution inside proot does not prove Android-native compatibility.
