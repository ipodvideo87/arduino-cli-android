# Validation Environment Research

This document records the validation provider landscape for the Android-first
ACL stack.

This is a reference research log. The formal validation scope and reporting
requirements live in [VALIDATION_POLICY.md](VALIDATION_POLICY.md) and
[DIAGNOSTIC_VALIDATION_STANDARD.md](DIAGNOSTIC_VALIDATION_STANDARD.md).

It is intentionally conservative:

- research first
- conclude second
- do not claim Android success from emulation
- keep native Termux as the Android source of truth
- keep real hardware as the final authority for upload, flash, and runtime

## Evidence Legend

- Confirmed device evidence: observed on native Termux or on the Android device.
- Confirmed host evidence: observed in the current Ubuntu/proot development
  environment.
- Inference: a reasoned conclusion that still depends on further proof.

## Validation Provider Hierarchy

The project should use layered validation providers instead of one hardcoded
smoke test.

### 1. Static Analysis

- Strengths: fast, cheap, catches obvious structural issues.
- Limitations: does not execute the code.
- Confidence boundary: useful for code review and CI gating, not behavior proof.

### 2. Unit / Integration Tests

- Strengths: exercise repository logic under controlled conditions.
- Limitations: run on the host runtime, not on Android hardware.
- Confidence boundary: proves repository behavior only.

### 3. Host Smoke Tests

- Strengths: quick end-to-end checks on the development host.
- Limitations: host runtime may differ from Android and Termux.
- Confidence boundary: preflight only.

### 4. ARM64 / QEMU Smoke Tests

- Strengths: useful for ARM64-oriented preflight, packaging checks, and workflow
  orchestration.
- Limitations: still not native Android and still not Termux.
- Confidence boundary: preflight only.

### 5. Native Termux

- Strengths: runs under the Android app sandbox and native Android filesystem,
  loader, and permission rules.
- Limitations: still not real hardware flashing or runtime proof.
- Confidence boundary: authoritative for Android compile and workflow claims.

### 6. Real Hardware

- Strengths: proves upload, flash, boot, and runtime behavior on physical
  devices.
- Limitations: slower and more device-specific.
- Confidence boundary: authoritative for upload, flash, and runtime claims.

### 7. Future GitHub Actions

- Strengths: reproducible automation and regression protection.
- Limitations: still only as strong as the environment it runs in.
- Confidence boundary: useful CI evidence, not a substitute for native Termux or
  hardware.

### 8. Future Desktop Validation

- Strengths: useful for desktop-oriented workflows and regression coverage.
- Limitations: not Android and not Termux.
- Confidence boundary: host-only evidence.

## Current Research Findings

### ARM64 Linux under `qemu-aarch64`

- Useful for ARM64 user-mode smoke tests and workflow preflight.
- It can help validate build logic, JSON reporting, package layout, and command
  orchestration.
- It does not prove Android compatibility.
- It does not exercise the Android app sandbox, Android permissions, Termux
  package layout, or the Android dynamic loader.

### Android Emulator inside `proot-distro`

- The Android Emulator on Linux relies on KVM for hardware acceleration and
  accelerated emulation is documented for x86/x86_64 guest images.
  - Evidence: [Android Emulator hardware acceleration](https://developer.android.com/studio/run/emulator-acceleration)
  - Evidence: [Android Emulator command-line docs](https://developer.android.com/studio/run/emulator-commandline)
- In the current development environment, `/dev/kvm` is absent.
- Inference: a full Android emulator running inside the current `proot-distro`
  environment would fall back to software emulation or fail to meet practical
  performance requirements.
- Inference: even if an emulator could be launched, that would still not be
  native Termux and would not prove Android device behavior.

### Termux-Like Rootfs

- Termux is an Android app and Linux environment that runs directly on Android.
  - Evidence: [Termux site](https://termux.dev/en/)
  - Evidence: [Termux docs](https://termux.dev/en/docs/)
- A generic Linux rootfs can approximate some shell and package-manager behavior.
- It cannot reproduce the Android app sandbox, Android permission model, or the
  Termux-specific execution environment faithfully.
- Inference: useful for tooling preflight, not for Android success claims.

### Android Bionic / Sysroot Compatibility

- A Bionic-oriented sysroot or compatibility test can help catch ABI and libc
  assumptions earlier than native device testing.
- It still does not exercise Android app permissions, USB host mediation, or the
  Termux runtime boundary.
- Inference: useful as an additional provider, not as a replacement for native
  Termux.

### Native Termux Smoke Tests

- Native Termux remains the authoritative Android compile and workflow
  validation provider.
- The commands documented in the native Termux smoke-test workflow remain the
  required evidence path for Android compile claims.
- The validation scripts in `scripts/android/` are therefore preflight helpers,
  not substitutes for on-device validation.

### Real Hardware Validation

- Real hardware remains the only provider that can validate USB acquisition,
  upload, flash, serial monitor behavior, and runtime boot behavior.
- This is a hard confidence boundary. Emulator and host smoke tests do not
  cross it.

## Current Environment Readiness

Observed in the current Ubuntu/proot development environment:

- Architecture: `aarch64`
- `go`: present
- `apt-get`: present
- `dpkg`: present
- `qemu-aarch64`: present
- `qemu-aarch64-static`: absent
- `/dev/kvm`: absent
- `shellcheck`: absent

Interpretation:

- ARM64 user-mode smoke tests are practical here.
- A full Android emulator is not practically validated here because the host
  lacks KVM access in this environment.
- Explicit bootstrap mode can still install missing host packages when the user
  asks for it, but the default action must remain report-only.

## Report Contract

Every validation provider should emit a human-readable summary and a structured
machine-readable report. The canonical reporting fields are defined in
[DIAGNOSTIC_VALIDATION_STANDARD.md](DIAGNOSTIC_VALIDATION_STANDARD.md).

This document keeps the provider landscape and environment-specific findings in
one place so the canonical validation docs do not need to carry the research
history.

## Practical Conclusion

- Emulated ARM64 smoke tests are worth building.
- Android emulator validation is not yet proven practical in the current
  `proot-distro` development environment.
- Termux-like rootfs and Bionic/sysroot providers are useful research and
  preflight aids, not proof of Android behavior.
- Native Termux and real hardware remain the decision points for Android and
  device claims.
