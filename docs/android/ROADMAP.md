# Roadmap

## Completed or In Progress

- Android runtime compatibility
- Android install patch pipeline
- ACL scanner
- ACL verifier
- ACL bootstrap
- ACL patch preview
- Compatibility layer
- ACL Engine
- CompileWorkflow
- FirmwarePackage
- FlashPlan
- BuildManifest
- ValidationReport
- `analysis.json`
- `README_FLASHING.txt`
- Validation-environment research
- Emulated ARM64 smoke-test provider
- USB transport research and architecture docs
- USB transport provider model and upload/monitor previews
- Transport provider skeleton, diagnostics models, and fake-provider tests
- Termux USB transport provider, command traces, and safe CLI diagnostics
- Termux USB fd-handoff probe and transport-stream diagnostics foundation
- Termux USB topology bridge foundation
- Transport Stream Foundation
- Transport API Stabilization
- Upload Engine foundation and prepare-only executor
- Native full-flash bootloader package validation on native Termux
- Governance framework batch 1: Codex operating model, engineering principles,
  decision framework, and documentation architecture
- Governance framework batch 1.5: repository governance, engineering
  methodology, architecture review, technical debt, and interface stability
- Governance framework batch 2: apply ownership and terminology discipline to
  high-risk validation, transport, and workflow preview docs
- Governance framework batch 3: Engineering Knowledge Framework, lifecycle,
  decision log, uncertainty register, confidence model, and lessons learned

## Next Milestones

- Complete Project Zero EOS adoption pilot and validate the manifest-first
  boundary
- Add emulated ARM64 / Termux-like smoke tests
- Research whether any interface-2 transfer diagnostic is safe and meaningful
  before adding live bulk I/O
- Build a generic Android USB transport bridge
- Validate the native Termux transport stream boundary
- Implement the upload workflow
- Implement the serial monitor workflow
- Validate the transport API stabilization boundary before upload depends on it
- Add GUI/workspaces
- Add project manager UX
- Add library/platform manager UX
- Improve firmware analysis
- Expand multi-board support
- Add GitHub Actions validation providers
- Add desktop validation providers
- Add Termux-like rootfs and Bionic/sysroot preflight providers if they prove
  useful
- Implement the first transport provider runtime after the architecture is
  proven stable
- Implement real upload execution after the dry-run upload foundation is
  validated on native Termux and real hardware
- Promote the bounded byte-stream bridge only after native Termux validation
  proves the experimental stream path

## STATUS Sync

`STATUS.md` is the authoritative snapshot of current work, blockers, and the
next engineering milestone.

`ROADMAP.md` is authoritative for future ordering and long-range direction.
docs/android/ROADMAP.md is authoritative for future ordering.

Update both documents when a validated change affects current state and the
future milestone sequence. If they diverge, treat `STATUS.md` as the source of
truth for the present state and `ROADMAP.md` as the source of truth for the
future order.

## Roadmap Notes

- ESP32-S3 remains the first validation target.
- The architecture should stay board-agnostic so other platforms can be added
  without rewriting the workflow model.
- Governance and documentation framework work should stay ahead of further
  upload-execution expansion so later batches build on stable decision and
  ownership rules.
- Repository governance and interface-stability guidance should be treated as
  part of the foundation before more architecture or schema changes are added.
- High-risk doc cleanup should continue to prefer canonical references and
  concise local summaries over duplicated policy text.
