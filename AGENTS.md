# Repository Guidelines

## Mission
This repository aims to become a production-quality Arduino CLI that runs natively on Android without chroots, PRoot, Docker, virtual machines, or a traditional Linux distribution. It is also the flagship implementation of the Android Compatibility Layer (ACL), a reusable framework for running Linux developer tools reliably on Android.

## Core Principles
- Android-first engineering.
- Automation over manual configuration.
- Reusable engineering over one-off fixes.
- Preserve upstream compatibility whenever practical.
- Keep Android-specific code isolated when possible.
- Prefer deterministic, reproducible behavior.

## Project Goals
- Run Arduino CLI natively on Android.
- Build and maintain ACL as a general-purpose compatibility platform.
- Detect Linux ELF executables and analyze their dependencies and runtime requirements.
- Patch compatible executables safely.
- Build portable runtimes when required.
- Support ESP32, ESP8266, RP2040, AVR, STM32, and future platforms.

## Current Priorities
Work in this order unless a higher-priority issue is clearly required:
1. Complete ACL runtime architecture.
2. Implement reliable ELF analysis and compatibility verification.
3. Implement safe ELF patching.
4. Execute `esptool` successfully.
5. Execute the ESP32 compiler toolchain.
6. Compile a Blink sketch.
7. Upload firmware to a physical ESP32-S3.

Do not claim a milestone is complete until it has been verified through real-world testing.

## Engineering Expectations
- Understand the existing architecture before changing it.
- Reuse existing components whenever possible.
- Avoid duplicating logic or creating application-specific hacks.
- Keep changes modular, maintainable, and reviewable.
- Update documentation when behavior changes.
- Never fabricate APIs, functionality, or test results.
- Clearly mark experimental behavior.

## Repository Notes
- ACL work lives under `acl/`.
- Keep generated binaries out of git.
- Shell wrappers, runtime checks, and scanner tooling should stay small and explicit.
