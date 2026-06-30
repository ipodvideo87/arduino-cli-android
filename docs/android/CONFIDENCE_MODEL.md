# Confidence Model

This document defines how the repository expresses confidence in an engineering
claim or decision.

Confidence is not the same as validation level.

- Validation level says how strong and where the evidence is.
- Confidence says how strongly the repository may rely on the conclusion.

## Confidence Levels

### Low

- The evidence is incomplete or mostly inferred.
- The repository should not treat the conclusion as a contract.

### Medium

- The evidence and architecture point in the same direction.
- The claim is plausible and useful, but still needs target-environment proof
  or broader validation.

### High

- The evidence strongly supports the conclusion.
- The repository can act on the conclusion, while still labeling any target
  validation gaps explicitly.

### Confirmed

- The relevant behavior has been demonstrated in the environment that matters
  for the claim being made.
- This is the strongest confidence state and should be used carefully.

## How Confidence Changes

Confidence should increase only when one or more of the following improve:

- evidence quality
- evidence relevance to the target environment
- reproducibility
- number of independent confirmations
- clarity of the boundary being claimed

Confidence should decrease when:

- the environment changes
- a later result contradicts an earlier assumption
- a claim is broadened beyond the evidence
- the validation scope turns out to be narrower than expected

## Relationship To Validation

Validation levels drive confidence, but they do not automatically determine it.

Examples:

- unit tests can raise confidence in logic
- host validation can raise confidence in build and local behavior
- native Termux validation is required for Android claims
- real hardware is required for upload, flash, and runtime claims

## Reporting Rule

Any engineering summary that states confidence should also state:

- what evidence changed it
- what remains unproven
- what validation level was achieved

