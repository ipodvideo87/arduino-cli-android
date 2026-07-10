# Engineering Milestone Summary

## Historical Classification

This document is historical evidence.
It preserves repository state at the time of the milestone.
It is not authoritative for current status.
For the current snapshot, see [STATUS.md](../../STATUS.md).

## Milestone
Native Full-Flash Bootloader Package Validation

## Objective
Prove that the ESP32-S3 sketch package could be produced natively on Termux as a
`full-flash` firmware package, then validate the package boundary and the
prepare-only upload consumer without moving into real upload or flashing.

## Scope
What was intentionally in scope:
- resolve the compile blocker that prevented package generation
- validate package contents and metadata in native Termux
- confirm the package was accepted by the prepare-only upload path
- keep evidence separate from implementation and transport work

What was explicitly out of scope:
- real upload execution
- flashing execution
- USB transport implementation
- serial monitor implementation
- EEL runtime work
- broad refactors unrelated to the package boundary

## Initial State
The repository started with a native-Termux package-validation milestone that was
not yet completed in on-device evidence. The immediate blocker was a compile
failure caused by missing `ESPAsyncWebServer.h` resolution in the active library
set.

## Engineering Journey
The milestone began with dependency investigation because the compile failure
pointed to a missing include, not a sketch logic problem. Native library search
showed the correct provider for `ESPAsyncWebServer.h` and its companion async
TCP dependency, so the smallest safe move was to install the exact libraries
matched by the index rather than guessing at a fork or alternate version.

After the dependency blocker was removed, the compile succeeded in native
Termux. That mattered because Android behavior is the source of truth here, and
the milestone could not be considered complete from host or proot evidence.

With the compile unblocked, package validation became the next focus. The goal
was to prove the package boundary itself: `full-flash` mode, required metadata,
and required artifacts. The package had to be checked before any upload-related
claims could be made.

The final step was prepare-only upload validation. This was used as a dry-run
consumer of the package, not as a transport or flashing exercise. That decision
kept the milestone bounded to the package and report surface while still proving
that the upload planner could consume the generated package safely.

## Evidence Collected
- native Termux compile succeeded
- `ESP Async WebServer 3.11.2` and `Async TCP 3.4.10` resolved the missing
  include blocker
- a `full-flash` firmware package was generated
- `package_mode = full-flash`
- manifest artifacts were present for bootloader, partitions, boot_app0, and
  application
- flash-plan required entries were present
- artifact existence checks passed for the generated package contents
- prepare-only upload dry-run passed
- rebuilding the repo-local `arduino-cli` binary with `go build -o arduino-cli .`
  and regenerating the package produced `target_chip = esp32s3` in the
  manifest, flash plan, and validation report, with validation warnings `null`
- no real upload or flashing was attempted

## What Was Proven
- the sketch compiles successfully in native Termux
- the dependency blocker was caused by missing library resolution, not by the
  sketch itself
- the package generator emits a valid `full-flash` package for this sketch
- the package contains the required bootloader, partitions, boot_app0, and
  application artifacts
- the flash plan matches the required package entries
- the prepare-only upload consumer can load and plan the package without opening
  a real transport stream
- the milestone can distinguish package validation from transport execution

## What Was Not Proven
- real upload
- flashing
- USB transport
- serial monitor
- EEL runtime
- byte-stream readiness
- target device runtime behavior

## Remaining Limitations
No package-level target-chip limitation remains after rebuild/regeneration.

- Severity: none for the package milestone
- Impact: none for the package milestone after binary rebuild and regeneration
- Recommended follow-up: if the warning ever returns, check binary freshness
  before patching source

## Lessons Learned
- dependency failures should be classified at the missing-include or library
  resolution layer before touching code
- native Termux is the deciding evidence source for Android claims
- prepare-only consumers are useful boundary checks when real transport is out
  of scope
- safer shell forms reduce false negatives in validation scripts
- package correctness and transport correctness should stay separate milestones
- warning handling should remain evidence-driven rather than speculative
- binary freshness can matter as much as source correctness when validating
  regenerated package outputs

## Repository Impact
Documentation updated:
- `STATUS.md`
- `docs/android/ROADMAP.md`
- `docs/android/TASK_RECOVERY.md`
- `docs/android/VALIDATED_FINDINGS.md`

Code changes:
- none

## Recommendation
The milestone should be committed.

It is ready for future upload and transport work only as a package boundary
baseline; real upload and flashing remain unproven and should stay separate.

## Next Recommended Milestone
Validate the next smallest evidence-driven step: inspect target-chip metadata
propagation only if that warning must be removed. Otherwise, move to the next
transport milestone without broadening scope.

## Repository Footer
Repository path: `/root/arduino-cli-android`
Branch: `android-runtime-v2`
Commit destination: local git commit on `android-runtime-v2`
Work belongs to: `arduino-cli-android`
Commit recommendation: yes
Repository status: dirty working tree with tracked documentation edits and
untracked EOS adoption files
