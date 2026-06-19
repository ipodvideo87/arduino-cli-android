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

Execution planning exists. Real execution is still experimental and intentionally
backend-limited in this sprint. A successful dry run does not prove Android-native
compatibility.

## Validation Guidance

Android execution must be validated from a fresh Termux clone outside proot. The current
container environment is useful for testing the planner and failure handling, but it is
not proof of device-native execution.
