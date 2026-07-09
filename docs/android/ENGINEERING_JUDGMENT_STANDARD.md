# Engineering Judgment Standard

This document defines the minimum reporting contract for engineering judgments
in this repository.

It exists to keep judgments traceable, evidence-backed, auditable, and
reproducible without turning every conclusion into a long report.

This document does not define validation scope, confidence semantics, closeout
shape, or workflow order. Those responsibilities remain with:

- `VALIDATION_POLICY.md`
- `CONFIDENCE_MODEL.md`
- `CLOSEOUT_REPORTING_STANDARD.md`
- `DEVELOPMENT_WORKFLOW.md`

## What Counts As A Judgment

A judgment is a conclusion about an engineering question that others may need
to rely on.

Common judgment types include:

- approval or rejection
- recommendation
- risk assessment
- completeness assessment
- confidence assessment
- architecture assessment

## Required Reporting Contract

Every consequential engineering judgment must include:

- Judgment
- Evidence
- Reasoning
- Assumptions
- Confidence
- Remaining uncertainty

## Field Guidance

### Judgment

State the conclusion directly. Do not hide the conclusion inside vague success
language.

### Evidence

Name the concrete evidence the judgment depends on. Evidence may come from
validation, inspection, repository state, or other documented artifacts.

### Reasoning

Explain why that evidence supports the judgment. Keep the reasoning concise and
explicit.

### Assumptions

State any assumptions that must remain true for the judgment to hold.

### Confidence

State the confidence level and explain why that level is appropriate.

Confidence must say more than `High` or `Medium`. It should describe the
evidence quality, the relevant scope, and what evidence would increase or
decrease confidence when the judgment is consequential.

### Remaining Uncertainty

State what is still not proven or not fully closed.

## Alternatives Considered

`Alternatives Considered` is optional.

Use it when:

- multiple reasonable engineering approaches existed
- an architectural decision was made
- workflow or long-term maintenance tradeoffs mattered

Omit it for routine factual judgments or when no meaningful alternative was
available.

## Relationship To Other Standards

- Use `VALIDATION_POLICY.md` to decide how strong the evidence is.
- Use `CONFIDENCE_MODEL.md` to interpret confidence levels.
- Use `CLOSEOUT_REPORTING_STANDARD.md` for the overall closeout report shape.
- Use `DEVELOPMENT_WORKFLOW.md` for when the judgment standard must be applied
  during task execution and closeout.

## Minimal Example Shape

```md
- Judgment:
- Evidence:
- Reasoning:
- Assumptions:
- Confidence:
- Remaining uncertainty:
- Alternatives considered: optional
```
