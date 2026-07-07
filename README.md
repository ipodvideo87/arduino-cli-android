# Arduino CLI Android

This repository is an experimental fork of Arduino CLI focused on making the
toolchain feel native on Android while building the Android Compatibility Layer
(ACL) as reusable infrastructure for Linux developer tools on Android.

The project is not just about getting one binary to run. It is about proving a
repeatable Android-native workflow for installing toolchains, validating Linux
ELF dependencies, compiling sketches, and eventually uploading firmware from
Android without depending on a desktop Linux machine.

For the long-term mission and intended end state, see [MISSION.md](MISSION.md).
For the current progress snapshot, see [STATUS.md](STATUS.md).
For future milestone ordering, see [docs/android/ROADMAP.md](docs/android/ROADMAP.md).

## Canonical Working Copy

Use this checkout for active work:

`/data/data/com.termux/files/home/Development/GitHub/arduino-cli-android`

Keep this README aligned with [STATUS.md](STATUS.md) and
[docs/android/DEVELOPMENT_WORKFLOW.md](docs/android/DEVELOPMENT_WORKFLOW.md)
when the current milestone or working rules change.

## Start Here

If you are resuming work or changing the repository, read these first:

1. [AGENTS.md](AGENTS.md)
2. [STATUS.md](STATUS.md)
3. [docs/android/DEVELOPMENT_WORKFLOW.md](docs/android/DEVELOPMENT_WORKFLOW.md)
4. [docs/android/TASK_RECOVERY.md](docs/android/TASK_RECOVERY.md)

## What This Project Is Trying To Do

The end goal is a complete Arduino development workflow that runs directly on
Android.

That means the repository is working toward a future where Android can:

- install Arduino CLI and its support tooling
- analyze Linux ELF-based board tools and runtime requirements
- adapt compatible tools to Android without breaking upstream compatibility
- compile Arduino sketches locally on the device
- upload firmware to real hardware from Android
- keep the workflow understandable, reproducible, and maintainable

The repository is also the proving ground for ACL. ACL is intended to become a
general-purpose compatibility layer for Linux developer tools on Android, not
just a one-off Arduino workaround.

## Development Environments

This repository is developed against two intentionally different Linux
environments:

1. Native Termux on Android is the production target and the final authority for
   Android behavior. It runs under the Android application sandbox, Android
   kernel, Bionic, SELinux, and Android filesystem and permission rules.
2. Ubuntu inside `proot-distro` is a development-only environment for tooling,
   GitHub work, and other tasks that are easier to run under Ubuntu/glibc. It is
   useful, but it does not prove Android compatibility.

When the two environments disagree, native Termux wins.

## Current Validated State

The repository has already proven a substantial amount of Android-side work:

- Arduino CLI builds successfully on Android/Termux.
- ESP32 board indexes and core packages install successfully.
- ACL can scan ELF files and report runtime metadata.
- ACL can inventory installed Arduino package executables and produce
  structured compatibility reports.
- ACL can validate scanned tool compatibility data before compile or execution
  work begins.
- The Android patch pipeline exists for install-time repair and ELF handling.
- The ACL compile workflow exists and can produce firmware packages.
- Native Termux validation has established the Termux USB discovery and
  permission boundary.
- Native Termux validation has also established the fd handoff and stream
  boundary diagnostics used for the Termux USB provider.
- Full-flash bootloader package validation for the native-Termux package path
  has been completed.

Those claims are validated in the native Android/Termux target unless noted
otherwise.

## What Is Still Experimental

Some important parts are not finished yet and should still be treated as
in-progress:

- The ACL runtime is still experimental and carries Termux-origin assumptions.
- ELF patching is still mostly a plan-first workflow; the full rewrite path is
  not finished.
- Runtime verification remains conservative and may fail until the runtime tree
  is fully populated.
- The transport stream foundation remains experimental until native Termux
  proves read/write behavior.
- USB flashing is still unimplemented as a validated Android-native workflow.
- Serial monitor support is still a future milestone.
- ACL should not be treated as production-ready or complete.

## Current Focus

Current work centers on the next narrow milestone rather than broad feature
expansion:

- preserve the native-Termux evidence trail
- keep the canonical repo path explicit in the workflow docs
- keep the evidence collector and CLI diagnostics aligned with current Android
  USB semantics
- validate the read-only claim/release diagnostics boundary before moving into
  any byte-stream or upload work

The authoritative snapshot for current work, blockers, and immediate next steps
is [STATUS.md](STATUS.md).

## Validation Workflow

Use this order when judging a meaningful change:

1. Unit tests
2. Integration tests
3. Build verification
4. Native Termux validation on Android

Passing under Ubuntu/proot is useful development evidence, but it is not the
final success criterion for Android-specific behavior.

## ACL Architecture

ACL is organized as a set of reusable layers rather than a pile of
special-purpose scripts:

- `scanner` inspects ELF files and extracts interpreter and dependency data.
- `scanner` also validates compatibility classifications before ACL attempts
  runtime work.
- `patcher` plans or applies safe ELF edits when the runtime requirements are
  known.
- `verifier` checks runtime layout and file compatibility before launch.
- `launcher` will eventually wrap the runtime and execute tools in a controlled
  environment.
- `builder` packages the runtime and supporting assets.

The design goal is to keep Android-specific behavior isolated while preserving
upstream compatibility where practical.

## Recent Work

Recent repository work has focused on these areas:

- Android runtime compatibility and install-time patching
- ACL scanner, verifier, bootstrap, and patch-preview commands
- the ACL compile workflow and firmware package generation
- the prepare-only upload engine foundation
- Termux USB discovery, permission, fd-handoff, and stream diagnostics
- native Termux evidence collection and stream-boundary closeout documentation
- repository governance and workflow documentation
- canonical working-copy cleanup so active work happens in the Termux-visible
  checkout

## Future Work

The roadmap continues toward the larger Android-native workflow:

- validate the native Termux transport stream boundary
- validate the Termux USB provider and fd-handoff path on native Termux
- validate upload planning on native Termux while keeping the CLI/report
  contract explicit
- implement upload execution
- implement the serial monitor workflow
- add GUI/workspaces and project manager UX
- improve firmware analysis
- expand multi-board support
- add desktop and GitHub Actions validation providers when useful

See [docs/android/ROADMAP.md](docs/android/ROADMAP.md) for the full ordered
list.

## Why The Documentation Is Structured This Way

This repository keeps three different truths separate:

- [MISSION.md](MISSION.md) explains why the project exists and where it is
  headed.
- [STATUS.md](STATUS.md) explains what is currently true.
- [docs/android/ROADMAP.md](docs/android/ROADMAP.md) explains what comes next.

That separation matters because the project moves through validated milestones.
Do not read mission, current state, and future plans as if they are the same
document.

## Working Rules

- Native Termux is the source of truth for Android behavior.
- Evidence comes before claims.
- Validate before completion.
- Prefer reusable infrastructure over one-off fixes.
- Keep the working copy on the canonical Termux-visible checkout.

If you are here to continue work, use the documents above as the source of
truth and keep this README current when the milestone mix changes.
