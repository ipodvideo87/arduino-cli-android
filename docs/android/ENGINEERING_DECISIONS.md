# Engineering Decisions

These decisions use an ADR-style format.

## FirmwarePackage is the canonical build output

- Status: accepted
- Context: The GUI needs one stable package boundary instead of scattered build
  artifacts.
- Decision: `FirmwarePackage` is the canonical compile output for UI consumption.
- Alternatives considered: exposing raw build folders directly; teaching the GUI
  to parse build internals.
- Consequences: package generation becomes a first-class step and raw ELF/MAP files
  remain authoritative inputs, not the primary UI contract.
- Validation/evidence if available: firmware-package tests and compile workflow
  integration.

## ACL Engine orchestrates workflows

- Status: accepted
- Context: The project needs a reusable orchestration layer for compile, package,
  diagnostics, install, and future transport workflows.
- Decision: the ACL Engine owns ordered workflow execution and structured reporting.
- Alternatives considered: wiring UI layers directly to command helpers.
- Consequences: workflows stay testable and reusable across CLI and future GUI
  layers.
- Validation/evidence if available: `acl workflow` command surfaces and engine tests.

## Validation is provider-based and report-driven

- Status: accepted
- Context: The project needs a layered validation ecosystem with explicit
  confidence boundaries instead of a single hardcoded smoke test.
- Decision: each validation provider should emit a human-readable summary and a
  structured machine-readable report, and the common schema should carry
  environment details, tests run, tests skipped, warnings, limitations,
  actions taken, results, and validation level.
- Alternatives considered: ad hoc shell scripts with unstructured console output.
- Consequences: the ACL Engine and future GUI layers can compare provider
  results, preserve institutional memory, and avoid overclaiming success.
- Validation/evidence if available: the new validation-environment research and
  emulated ARM64 smoke-test workflow docs.

## Beginner, Advanced, Professional are UI layers

- Status: accepted
- Context: Users should get different information density without losing capability.
- Decision: the same engine and package model feeds all modes.
- Alternatives considered: separate beginner and pro code paths.
- Consequences: the backend stays single-sourced; the UI chooses disclosure level.
- Validation/evidence if available: workflow report separation in code and tests.

## Metadata-first firmware packaging

- Status: accepted
- Context: ESP32-S3 packages need a reliable way to derive bootloader and flash
  plan entries from actual Arduino build metadata.
- Decision: prefer generated flash arguments first, then build-property patterns,
  then filesystem resolution; fall back to app-only with warning if metadata is
  incomplete.
- Alternatives considered: hardcoding offsets or failing normal compile on missing
  bootloader metadata.
- Consequences: full-flash packages are produced when the core exposes enough data;
  app-only remains available when metadata is incomplete.
- Validation/evidence if available: firmware package tests and native Termux
  workflow behavior.

## QEMU/emulated testing is preflight only

- Status: accepted
- Context: Emulated tests can catch regressions early but do not prove Android-native
  behavior.
- Decision: treat QEMU and emulated ARM64 smoke tests as preflight checks only.
- Alternatives considered: using emulation as a substitute for native validation.
- Consequences: stronger evidence is required before Android success claims.
- Validation/evidence if available: native-vs-proot divergences and project policy.

## Native Termux validation is required for Android compile claims

- Status: accepted
- Context: Android behavior differs from proot and desktop Linux.
- Decision: compile success on Android must be validated in native Termux.
- Alternatives considered: relying on Ubuntu/proot or build success alone.
- Consequences: native Termux remains the source of truth for Android compile claims.
- Validation/evidence if available: project policy and native validation history.

## Explicit installation is opt-in in validation bootstrap mode

- Status: accepted
- Context: validation bootstraps should be safe by default and not mutate the
  host unless the user explicitly asks for it.
- Decision: default bootstrap behavior is report-only; installation only occurs
  when the user requests `bootstrap --install` or sets an equivalent explicit
  installation flag.
- Alternatives considered: implicit package installation during inspection or
  verification.
- Consequences: environment checks remain safe and repeatable for existing
  setups and clean installs alike.
- Validation/evidence if available: validation workflow policy and bootstrap
  script design.

## Real hardware validation is required for upload/flash claims

- Status: accepted
- Context: Upload and flash behavior depends on transport and device state.
- Decision: do not claim upload/flash success until tested on physical hardware.
- Alternatives considered: desktop-only or emulator-only upload verification.
- Consequences: upload milestones remain gated by real hardware proof.
- Validation/evidence if available: current upload milestone policy.

## USB upload must be transport-based and device-agnostic

- Status: accepted
- Context: Android USB access is permission-gated and board-specific upload logic
  does not scale.
- Decision: build a generic transport layer and keep upload workflows device-agnostic.
- Alternatives considered: hardcoding board-specific serial paths or VID/PID logic.
- Consequences: Arduino CLI stays unaware of Android internals and UI layers can
  reuse the same transport abstraction.
- Validation/evidence if available: transport architecture docs and Termux USB findings.

## Android compatibility belongs in reusable ACL infrastructure

- Status: accepted
- Context: Android compatibility work should benefit compile, install, diagnostics,
  and future tools.
- Decision: put Android-specific logic in ACL layers that can be reused and tested.
- Alternatives considered: patching only individual commands.
- Consequences: lower maintenance cost and clearer ownership boundaries.
- Validation/evidence if available: install patch pipeline, compatibility layer,
  and engine work.

## analysis.json is the GUI-facing build analysis source

- Status: accepted
- Context: The GUI should not need to parse raw ELF or map files.
- Decision: emit a structured `analysis.json` that carries the machine-readable
  build analysis surface, while `firmware.elf` and `firmware.map` remain the
  authoritative raw artifacts.
- Alternatives considered: making the GUI parse ELF/MAP directly.
- Consequences: future analysis work can evolve behind a versioned schema without
  breaking the UI contract.
- Validation/evidence if available: firmware package schema and analysis placeholder
  implementation.

## Transport API is stabilizing before upload depends on it

- Status: accepted
- Context: Upload and monitor workflows will depend on the transport provider,
  manager, session, and stream contracts, so those contracts need to settle
  before the next layer is built.
- Decision: treat the current transport API as stabilizing; keep changes
  additive when possible and preserve compatibility aliases such as
  `ByteStreamSession`.
- Alternatives considered: freezing the API now with no further refinements;
  leaving the contract fully experimental until upload is complete.
- Consequences: upload work can build on a clear contract boundary while the
  stream implementation remains experimentally validated.
- Validation/evidence if available: transport provider tests, stream wrapper
  tests, and native Termux fd-handoff diagnostics.

## Upload Engine foundation is dry-run only and transport-neutral

- Status: accepted
- Context: The project needs an upload planning layer that can consume firmware
  packages and flash plans before real transport execution exists.
- Decision: implement the Upload Engine foundation as a dry-run planner that
  validates firmware packages, derives upload steps, and reports diagnostics
  without opening real transport streams or sending bytes.
- Alternatives considered: coupling the foundation directly to transport
  execution; delaying the upload layer until flashing is ready.
- Consequences: CLI and workflow layers can surface upload planning now, while
  real flashing remains a later transport-specific milestone.
- Validation/evidence if available: upload engine package tests and ACL workflow
  upload-dry-run integration tests.

## ACL workflow upload is positional and dry-run only

- Status: accepted
- Context: Native Termux validation showed the upload workflow is useful as a
  dry-run planner, but GUI and CLI consumers need the contract to be explicit.
- Decision: expose the workflow as `acl workflow upload <firmware-package>` and
  keep it dry-run only; do not add `--dry-run` or `--package` flags unless the
  architecture and validation docs are updated together.
- Alternatives considered: adding explicit dry-run/package flags; leaving the
  workflow contract implicit.
- Consequences: the command stays simple and predictable, and GUI layers can
  depend on the positional package contract without guessing about hidden flags.
- Validation/evidence if available: native Termux help output and dry-run
  validation from the current milestone.

## EOS adoption should be manifest-first and AGENTS should be overlay-only

- Status: accepted
- Context: The first real EOS adoption pilot showed that a repository can state
  its EOS relationship more cleanly through a project manifest than by
  duplicating EOS doctrine in `AGENTS.md`.
- Decision: adopt EOS through `eos.project.json` as the canonical contract and
  keep `AGENTS.md` as a project-local overlay that points back to EOS canon.
- Alternatives considered: treating `AGENTS.md` as the canonical adoption
  contract; adding a separate lock file; encoding the adoption boundary only in
  prose.
- Consequences: version pinning, compatibility, and project extensions become
  explicit and machine-readable, while local guidance stays readable without
  becoming a second constitution.
- Validation/evidence if available: repository review of `AGENTS.md`, `STATUS.md`,
  `ROADMAP.md`, `TASK_RECOVERY.md`, and the EOS adoption docs.

## Engineering Knowledge Framework is the repository knowledge spine

- Status: accepted
- Context: The repository needs a durable way to turn each engineering task into
  persistent knowledge instead of isolated implementation history.
- Decision: add a canonical engineering knowledge framework with dedicated
  lifecycle, decision log, uncertainty register, confidence model, and lessons
  learned documents, and treat it as the connective tissue between existing
  decisions, findings, validation policy, and roadmap/status reporting.
- Alternatives considered: leaving task closeout knowledge in chat history;
  embedding the same responsibilities into existing ADR and findings docs
  without separate knowledge artifacts.
- Consequences: future tasks can answer the same closeout questions
  consistently, and the repository gains a single route for question -> evidence
  -> decision -> confidence -> lesson -> roadmap impact.
- Validation/evidence if available: documentation review and the current
  governance framework integration.

## Canonical documentation changes should require intent review and current-state synchronization

- Status: accepted
- Context: The Phase 6 after-state audit showed that governance validation can
  pass while `STATUS.md` remains stale and a historical milestone summary still
  looks current unless the edit path explicitly checks behavior preservation and
  current-state synchronization.
- Decision: keep the existing canonical ownership model, require a concise
  intent review for important canonical-document edits, require an explicit
  current-state synchronization determination at closeout, and extend
  governance validation only with bounded read-only checks for required
  documents, routing, ownership declarations, historical labels, and stale-path
  hygiene.
- Alternatives considered: adding a new constitution; relying on manual review
  only; letting the governance validator infer semantic equivalence; auto-editing
  canonical documents.
- Consequences: canonical docs remain the source of truth, but edits now have a
  defined review gate and deterministic guardrail against the specific drift
  class that occurred after Phase 6.
- Validation/evidence if available: Phase 6 after-state audit, governance
  validation output, and the current canonical docs set.
