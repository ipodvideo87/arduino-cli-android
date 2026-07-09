# Closeout Reporting Standard

This document defines the minimum closeout report shape for Codex tasks in this
repository.

The purpose is to keep closeouts auditable. A task may be technically complete
but still too vague if the report does not show what changed, what was checked,
and what remains uncertain.

When a closeout includes a final verdict, confidence statement, recommendation,
or risk assessment, that judgment must also follow
`ENGINEERING_JUDGMENT_STANDARD.md`.

## Required Fields

Every closeout report must include:

- repository path
- branch
- commit hash or `HEAD`
- files changed
- what each changed file now owns or changed
- validation performed
- validator output when applicable
- evidence summary
- known limitations
- deferred work
- whether branch matches origin
- final verdict

## Practical Shape

Keep the report short, but do not omit the required fields.

Recommended order:

1. Repository and branch state
2. Files changed and what each file changed
3. Validation and validator output
4. Evidence summary and limitations
5. Deferred work and branch/origin status
6. Final verdict

## Guidance

- Name the validation level or scope when relevant.
- Do not collapse evidence into a vague success claim.
- If a field does not apply, say why.
- If the branch does not match origin, say so explicitly.
- If the report depends on manual judgment, name the boundary.
- Keep the closeout shape in this document and the judgment contract in
  `ENGINEERING_JUDGMENT_STANDARD.md`; do not duplicate either one here.

## Example Skeleton

```md
- Repository path:
- Branch:
- Commit hash:
- Files changed:
- File ownership or change summary:
- Validation performed:
- Validator output:
- Evidence summary:
- Known limitations:
- Deferred work:
- Branch matches origin:
- Final verdict:
```
