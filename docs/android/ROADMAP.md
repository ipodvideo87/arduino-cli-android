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
- Transport Stream Foundation
- Transport API Stabilization
- Upload Engine foundation and prepare-only executor

## Next Milestones

- Finish full-flash bootloader package validation on native Termux
- Add emulated ARM64 / Termux-like smoke tests
- Build a generic Android USB transport bridge
- Validate the Termux USB provider and file-descriptor handoff path on native Termux
- Validate the TERMUX_USB_FD probe surface on native Termux
- Validate the transport stream foundation on native Termux
- Validate upload prepare-only planning on native Termux and keep the CLI/report contract explicit
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

## Roadmap Notes

- ESP32-S3 remains the first validation target.
- The architecture should stay board-agnostic so other platforms can be added
  without rewriting the workflow model.
