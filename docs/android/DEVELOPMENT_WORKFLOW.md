# Development Workflow

## Core Rules

- Plan before coding when architecture is involved.
- Implement in small milestones.
- Run focused tests first.
- Run native Termux validation when Android behavior is affected.
- Run real hardware validation before claiming upload or flash success.
- Update docs and `STATUS.md` when behavior or evidence changes.
- Push to GitHub after completed validated change sets unless instructed otherwise.

## Working Practices

- Prefer reusable ACL infrastructure over one-off fixes.
- Keep architecture decisions documented as they land.
- When unsure, document the assumption and the open question.
- Treat queued branches as reference material unless reviewed and intentionally
  merged.

## Git Practices

- Do not use `git add .` blindly when untracked or WIP files exist.
- Stage only the files in scope for the change.
- Keep commits small enough that the evidence and intent are readable later.
- Push after a completion so the remote branch reflects the current validated state.

## Validation Expectations

- Do not merge to `main` until real Android validation is complete.
- Do not claim success beyond the validation level actually achieved.
- Keep preflight, emulated, native, and hardware evidence distinct.

