# UI Workspaces

The backend is shared. Only the presentation changes.

This repository should treat the UI as three workspaces:

- Beginner
- Advanced
- Professional

## Principle

Beginner is not a reduced IDE.

It is Professional with automation and progressive disclosure.

## Beginner

Beginner mode should:

- show the build result as a package
- surface the flash readiness state
- hide low-level clutter unless a problem needs it
- convert compatibility issues into plain-language next steps

## Advanced

Advanced mode should:

- show the package and its validation state
- reveal flash plans and selected transports
- reveal compatibility decisions in summary form
- keep the common workflow simple

## Professional

Professional mode should expose:

- manifest details
- compatibility rules and decisions
- flash plan details
- ELF and MAP artifacts
- validation reports
- transport diagnostics
- raw build and flashing metadata

## Backend Rule

The backend does not fork into separate code paths for each workspace.

The backend produces the same firmware package, flash plan, validation report,
and diagnostics workflow. The UI chooses how much of that state to show.

## Related Code

- [Firmware Package Architecture](FIRMWARE_PACKAGE_ARCHITECTURE.md)
- [Flash Plan Architecture](FLASH_PLAN_ARCHITECTURE.md)
- [Diagnostics Workflow](DIAGNOSTICS_WORKFLOW.md)
