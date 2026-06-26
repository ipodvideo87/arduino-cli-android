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

## Validation Providers

Use a layered provider model instead of one hardcoded smoke test.

- Static analysis
- Unit / integration tests
- Host smoke tests
- ARM64 / QEMU smoke tests
- Native Termux
- Real hardware

Future providers may include:

- GitHub Actions
- desktop validation
- Android emulator validation if it can be proven practical
- Termux-like rootfs checks
- Android Bionic / sysroot compatibility checks

Every provider should emit:

- environment details
- architecture
- tool versions
- tests executed
- tests skipped
- results
- warnings
- limitations
- validation level or confidence boundary

## Rules

- Emulated ARM64 or QEMU smoke tests are useful preflight checks only.
- Emulated tests do not prove native Android success.
- Native Termux is required for Android compile claims.
- Real hardware is required for upload, flash, and runtime claims.
- Default bootstrap mode should not modify the host. Installation must be
  explicit.

## Claiming Success

- State the highest achieved validation level explicitly.
- Keep compile, packaging, validation, upload, flash, and runtime claims separate.
- Do not collapse a lower evidence level into a higher one.
- Label emulated validation as preflight only.
