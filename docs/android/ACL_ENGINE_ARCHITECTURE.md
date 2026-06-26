# ACL Engine Architecture

The ACL Engine is the orchestration layer that sits above scanner, verifier,
patch preview, compatibility, firmware packaging, and transport selection.

It exists so future GUI/workspace layers can run workflows without calling
Arduino CLI internals or individual ACL utilities directly.

## Responsibilities

- execute ordered workflows
- publish structured progress events
- collect machine-readable workflow reports
- preserve beginner, advanced, and professional views over one backend model
- stop on critical failures
- allow optional steps to be skipped

## Core Types

- `Engine`
- `Workflow`
- `Job`
- `Step`
- `StepStatus`
- `StepResult`
- `JobResult`
- `WorkflowReport`
- `Event`
- `EventBus`
- `EventSink`

## Workflow Model

Workflows are organized into jobs, and jobs are organized into ordered steps.

The engine does not know about UI state. It only knows about:

- input context
- step execution
- structured output
- events for logs and progress bars

## Current Workflow Skeletons

- `BootstrapWorkflow`
  - scan
  - verify
  - patch-preview
  - bootstrap report
- `CompileWorkflow`
  - preflight
  - compatibility check
  - compile hook
  - firmware package generation hook
  - binary validation hook
  - build report
- `FlashWorkflow`
  - package validation
  - transport selection
  - USB bridge placeholder
  - flash placeholder
  - verify placeholder
- `DiagnosticsWorkflow`
  - scanner
  - verifier
  - Android environment checks
  - toolchain checks
  - diagnostics report

## UI Contract

- Beginner mode consumes the same engine reports as professional mode.
- Beginner mode is professional mode with automation and progressive disclosure.
- Advanced mode is a midpoint view of the same engine output.

## Related Code

- `internal/acl/engine`
- `internal/acl/scanner`
- `internal/acl/verifier`
- `internal/acl/patcher`
- `internal/acl/bootstrap`
