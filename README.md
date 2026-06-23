# Arduino CLI Android

An experimental fork of Arduino CLI with Android and Termux compatibility work.

For the long-term project mission and intended end state, see [MISSION.md](MISSION.md).
For the current progress snapshot, see [STATUS.md](STATUS.md).

## Development Environments

This repository is developed against two intentionally different Linux environments:

1. Native Termux on Android. This is the production target and the final authority for behavior. It uses Android userspace, Bionic, the Android kernel, Android filesystem rules, SELinux, and the Android app sandbox.
2. Ubuntu inside `proot-distro`. This is a development-only environment for tooling, GitHub work, and other tasks that are easier to run under Ubuntu/glibc. It is useful, but it does not prove Android compatibility.

When the two environments disagree, native Termux wins.

## Project Goal

Run Arduino CLI and its toolchain-dependent helpers directly on Android devices while keeping
the workflow understandable, testable, and as upstream-friendly as possible.

## Current Working Features

- Arduino CLI builds successfully on Android/Termux.
- ESP32 board indexes and core packages install successfully.
- Linux ARM64-compatible host handling is in place for the current toolchain flow.
- ACL v0.1 can scan ELF files and report basic runtime metadata.
- ACL can inventory installed Arduino package executables and produce structured compatibility reports.
- ACL can validate scanned tool compatibility data before compile or execution work begins.

These claims are validated in the native Android/Termux target unless noted otherwise.

## Current Broken or Experimental Features

- The ACL runtime is still experimental and carries Termux-origin assumptions.
- The Android runtime bundle currently vendors the Termux glibc runtime set in
  `internal/android/runtime`; any future runtime library that is not bundled
  there must come from native Termux `gcc-libs-glibc` / `glibc-repo`.
- ELF patching is mostly a plan-first workflow; the actual rewrite path is not finished.
- Runtime verification is conservative and may fail until the runtime tree is fully populated.
- ACL should not be treated as production-ready or complete.

## Validation Workflow

Use this order when judging a meaningful change:

1. Unit tests.
2. Integration tests.
3. Build verification.
4. Native Termux validation on Android.

Passing under Ubuntu/proot is useful development evidence, but it is not the final success criterion for Android-specific behavior.

## ACL Architecture

- `scanner` inspects ELF files and extracts interpreter and dependency data.
- `scanner` also validates compatibility classifications before ACL attempts runtime work.
- `patcher` plans or applies safe ELF edits when the runtime requirements are known.
- `verifier` checks runtime layout and file compatibility before launch.
- `launcher` will eventually wrap the runtime and execute tools in a controlled environment.
- `builder` packages the runtime and supporting assets.

## Known Blocker

The portable runtime is still the main blocker. Without a complete runtime tree and loader
story, ACL can analyze binaries and plan changes, but it cannot yet promise transparent
execution of arbitrary Linux tools on Android.

## Next Milestone

The immediate milestone is a simple pipeline:

`scan -> patch plan -> verify -> compile sketches`

That sequence should stay the focus until the runtime path is stable enough for broader use.
