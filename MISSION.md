# Mission

This repository exists to make Arduino CLI truly native on Android.

Arduino CLI should feel like it was designed for Android from the very beginning.

The mission is larger than getting one binary to start or one helper tool to limp
through execution. This project aims to make Android a practical first-class platform
for Arduino and embedded development, where practical, without requiring a separate
desktop operating system.

## Why This Project Exists

Arduino CLI is already a strong foundation for board management, toolchain installation,
project workflows, and automation. But the normal Arduino ecosystem still assumes that
serious development happens on desktop Linux, Windows, or macOS.

This repository exists because Android devices are now capable enough to be used as real
development machines, yet much of the Arduino toolchain stack does not currently behave
as though Android is a first-class host environment.

The goal of this fork is to close that gap without reducing the work to a collection of
one-off hacks.

## The Problem

Arduino CLI itself is a Go program and can often be built on Android. The harder problem
is the platform tooling that Arduino CLI orchestrates.

Many board packages rely on Linux ELF executables and toolchains that expect:

- glibc
- a conventional Linux filesystem layout
- standard interpreter paths
- conventional loader behavior
- assumptions that do not match Android’s Bionic-based environment

Android uses a different C library, a different filesystem model, different host
expectations, and tighter execution constraints than a conventional Linux desktop.

As a result, “the CLI builds” is not enough. A practical Arduino workflow on Android
also needs the surrounding platform tools to install, validate, and execute reliably.

## The Long-Term Goal

The long-term goal is a complete Arduino development workflow that can run directly on
Android.

The intended workflow is:

- install Arduino CLI
- install board platforms
- install toolchains
- compile Arduino sketches
- execute platform tools
- upload firmware to real hardware
- manage projects entirely on Android

This aims to work without requiring:

- desktop Linux
- Windows
- macOS
- WSL
- Docker
- PRoot
- remote compilation services

Some parts of that workflow are currently implemented, some are tested in limited ways,
and some are still planned. Mission is not the same thing as current status.

## Android Compatibility Layer (ACL)

ACL exists because many Arduino platform tools are Linux ELF binaries expecting glibc
and a conventional Linux filesystem, while Android uses Bionic and a different
filesystem model.

ACL is intended to provide reusable infrastructure for analyzing, validating, adapting,
and, where practical, executing Linux-oriented development tools on Android.

Arduino CLI is the proving ground for ACL, not the end goal.

If ACL matures successfully, it is intended to become reusable infrastructure for other
Linux-based developer tooling on Android beyond Arduino CLI.

## Success Criteria

Success for this repository is not just that code exists. Success means the repository
can reproducibly demonstrate meaningful Android-native development workflows.

The project should eventually be able to show, through reproducible validation:

- Arduino CLI builds and behaves correctly on Android
- board platforms and tool dependencies can be installed on Android
- required platform tools can be analyzed and prepared for Android compatibility
- sketches can be compiled on Android for supported targets
- firmware can be uploaded from Android to real hardware
- the workflow is understandable, maintainable, and not dependent on hidden host state

Until those outcomes are validated, they should be described as implemented, tested,
experimental, or planned as appropriate, not as complete.

## Milestones

The exact milestone order may evolve, but the broad path is:

1. Build Arduino CLI cleanly on Android.
2. Analyze Linux ELF-based platform tools and their runtime requirements.
3. Build and validate ACL as reusable compatibility infrastructure.
4. Safely patch or adapt compatible tools where required.
5. Execute required platform tools on Android where practical.
6. Compile real Arduino sketches on Android.
7. Upload firmware to physical hardware from Android.
8. Support a complete project workflow directly on Android.

Some of these steps are currently in progress. Some remain planned. Status belongs in
`STATUS.md` when that document exists; this file defines mission and direction.

## Non-Goals

This repository does not aim to:

- depend permanently on desktop Linux, Windows, or macOS to complete normal workflows
- treat Docker, PRoot, WSL, or remote compilation as the primary solution
- optimize only for one patched binary while ignoring the broader workflow
- claim Android parity before the full workflow has been reproducibly validated
- turn ACL into a pile of application-specific hacks that cannot be reused

Workarounds may still be useful during investigation, but they are not the intended end
state.

## Engineering Principles

The mission should be pursued with the following engineering principles:

- Android-first design
- reproducible validation before claims
- reusable infrastructure over one-off fixes
- clear separation between implemented, tested, validated, experimental, and planned
- upstream compatibility where practical
- documentation that tracks what the repository currently proves
- incremental milestones that can be demonstrated, not just described

## Relationship to AGENTS.md and STATUS.md

`MISSION.md` is the high-level statement of purpose and long-term direction.

`AGENTS.md` defines repository working standards, contributor expectations, engineering
policy, and documentation discipline.

`STATUS.md`, when present, should be the canonical snapshot of current progress,
completed milestones, work in progress, blockers, and the next engineering milestone.

In short:

- `MISSION.md` explains why this repository exists and where it aims to go
- `AGENTS.md` explains how work should be carried out
- `STATUS.md` explains what is currently true
