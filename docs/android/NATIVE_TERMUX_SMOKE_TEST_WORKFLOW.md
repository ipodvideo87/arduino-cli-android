# Native Termux Smoke Test Workflow

This workflow is the native Termux validation path for the Android-first ACL
stack.

It is documentation and validation guidance, not a claim that upload is done.

## Goals

- verify the native Termux toolchain still compiles
- verify firmware package generation still works
- verify the Android post-install pipeline still repairs installed executables
- avoid claiming upload success until a real USB bridge exists

## Workflow

1. Clean the relevant Arduino and ACL caches if needed.
2. Run `arduino-cli acl bootstrap --details` to confirm the Android runtime,
   scanner, verifier, patch preview, and install-pipeline wiring are healthy.
3. Run `arduino-cli acl scan --details` against the installed packages tree.
4. Run `arduino-cli acl verify --details` against the same tree and, when
   relevant, a known executable target.
5. Run `arduino-cli acl patch-preview --details` to confirm the dry-run report
   includes interpreter, RPATH, wrapper, and permission changes.
6. Compile Blink on native Termux.
7. Compile a larger multi-file sketch when available.
8. Confirm a `FirmwarePackage` is generated with:
   - application binary
   - bootloader binary
   - partition table binary
   - ELF
   - MAP when available
   - build manifest
   - flash plan
   - validation report
9. Inspect the package JSON output if a failure occurs.
10. Verify installed tools and platforms still launch after Android patching.
11. Record any permission or runtime repair evidence in `STATUS.md` and the
   Android research log.

## Explicit Non-Goals

- Do not claim USB flashing is complete.
- Do not use the workflow as proof of upload success.
- Do not skip the firmware package check.

## Current Evidence

- Native Termux compilation of Blink is already validated in this workstream.
- Native Termux compilation of a larger ESP32-S3 project has also been reported
  as successful in this workstream.
- Firmware package generation is now wired into the compile path.
- The ACL CLI now exposes scanner, verifier, patch-preview, and bootstrap
  commands as the native validation entry points for this workflow.
