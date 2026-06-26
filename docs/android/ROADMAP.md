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

## Next Milestones

- Finish full-flash bootloader package validation on native Termux
- Add emulated ARM64 / Termux-like smoke tests
- Build a generic Android USB transport bridge
- Implement the upload workflow
- Implement the serial monitor workflow
- Add GUI/workspaces
- Add project manager UX
- Add library/platform manager UX
- Improve firmware analysis
- Expand multi-board support

## Roadmap Notes

- ESP32-S3 remains the first validation target.
- The architecture should stay board-agnostic so other platforms can be added
  without rewriting the workflow model.

