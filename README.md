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
For the non-negotiable process invariants, see
[docs/android/ENGINEERING_INVARIANTS.md](docs/android/ENGINEERING_INVARIANTS.md).

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

## Project Status And Roadmap

Current state, active work, blockers, recent validated work, and the next
engineering milestone belong in [STATUS.md](STATUS.md).

Future milestone ordering belongs in
[docs/android/ROADMAP.md](docs/android/ROADMAP.md).

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


## Licensing, Attribution, and Upstream Relationship

This repository is an independent, experimental, modified fork of Arduino CLI.

Arduino CLI and the original upstream source remain copyright their
respective authors and contributors and are licensed under the GNU General
Public License version 3. The complete GPL-3.0 license text is preserved in
[LICENSE.txt](LICENSE.txt).

Original modifications and additions made in this fork remain copyright
their respective authors, to the extent applicable, and are distributed
under GPL-3.0 as part of this modified work.

No ownership is claimed over unmodified Arduino CLI source code,
third-party software, documentation, trademarks, logos, or other materials
belonging to their respective owners.

This repository is still under active development. A more detailed upstream
baseline, modification manifest, authorship and provenance record,
third-party notices inventory, and trademark notice are being prepared and
will be added as the project develops.

Until those records are completed, LICENSE.txt, preserved source-file
headers, Git history, commit records, and existing upstream notices remain
authoritative.

This interim notice does not replace, restrict, expand, or modify the terms
of GPL-3.0 or any applicable third-party license.

Arduino® is a trademark of Arduino S.r.l. This project is independent and
is not affiliated with, endorsed by, sponsored by, certified by, or
officially associated with Arduino S.r.l.
