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
- surface compile workflow summaries before the package details
- surface the flash readiness state
- hide low-level clutter unless a problem needs it
- convert compatibility issues into plain-language next steps

## Advanced

Advanced mode should:

- show the package and its validation state
- reveal compile workflow report details and package location
- reveal flash plans and selected transports
- reveal compatibility decisions in summary form
- keep the common workflow simple

## Professional

Professional mode should expose:

- manifest details
- compile workflow events and report payloads
- compatibility rules and decisions
- flash plan details
- ELF and MAP artifacts
- validation reports
- transport diagnostics
- raw build and flashing metadata

## Backend Rule

The backend does not fork into separate code paths for each workspace.

The backend produces the same firmware package, flash plan, validation report,
diagnostics workflow, and ACL engine report. The UI chooses how much of that
state to show.

The engine is the backend boundary for future workspace screens:

- Beginner consumes the engine with automation and progressive disclosure.
- Advanced consumes the same engine with more of the report exposed.
- Professional consumes the full engine report, events, and artifacts.

## Related Code

- [Firmware Package Architecture](FIRMWARE_PACKAGE_ARCHITECTURE.md)
- [Flash Plan Architecture](FLASH_PLAN_ARCHITECTURE.md)
- [Diagnostics Workflow](DIAGNOSTICS_WORKFLOW.md)
- [ACL Engine Architecture](ACL_ENGINE_ARCHITECTURE.md)
