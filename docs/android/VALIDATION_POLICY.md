# Validation Policy

This project uses evidence levels to avoid overclaiming.

## Levels

- Level 0: builds locally
- Level 1: unit tests pass
- Level 2: CLI command exists and basic help works
- Level 3: native Termux compile or workflow validation passes
- Level 4: real firmware package validates on device
- Level 5: real hardware upload or flash succeeds
- Level 6: flashed firmware boots and runtime behavior is verified

## Rules

- Emulated ARM64 or QEMU smoke tests are useful preflight checks only.
- Emulated tests do not prove native Android success.
- Native Termux is required for Android compile claims.
- Real hardware is required for upload, flash, and runtime claims.

## Claiming Success

- State the highest achieved validation level explicitly.
- Keep compile, packaging, validation, upload, flash, and runtime claims separate.
- Do not collapse a lower evidence level into a higher one.

