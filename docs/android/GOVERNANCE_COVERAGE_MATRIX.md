# Governance Coverage Matrix

This matrix maps engineering rules to the current coverage surface so Codex can
see what is already enforced, what is only documented, and what still needs a
follow-up.

The matrix is intentionally lightweight. It is a planning and review aid, not a
second policy system.

## How To Read This Matrix

- Covered means a policy, validator, or repeatable test already exists.
- Partial means the rule is documented, but enforcement is manual or narrow.
- Missing means the repository still depends on human judgment or memory.
- Future means the capability is intentionally deferred.

## Coverage Matrix

| Engineering rule | Current coverage | Evidence or tooling | Gap | Recommended next check |
| --- | --- | --- | --- | --- |
| Canonical repo path and preflight reporting | Covered | `AGENTS.md`, `docs/android/DEVELOPMENT_WORKFLOW.md`, repo preflight report | No automatic reminder outside workflow | Ask for the canonical preflight at session start |
| Current state vs future ordering stay separate | Covered | `STATUS.md`, `docs/android/ROADMAP.md`, workflow ownership rules | Manual review still required for drift | Run governance validation before closeout when docs changed |
| Evidence before claims | Partial | `docs/android/VALIDATION_POLICY.md`, `docs/android/VALIDATED_FINDINGS.md` | No dedicated evidence-vs-claim audit yet | Recommend an evidence review when claims or validation summaries change |
| Recurring problem becomes process debt | Partial | `docs/android/LESSONS_LEARNED.md`, `docs/android/TECHNICAL_DEBT_POLICY.md`, this framework | No canonical debt register entry format before Phase 6A | Record the recurrence in the debt register or knowledge docs |
| Docs and evidence must not disagree silently | Partial | `docs/android/VALIDATION_POLICY.md`, `docs/android/ENGINEERING_KNOWLEDGE_FRAMEWORK.md` | Manual cross-check only | Mark the doc stale and note the contradiction in the appropriate artifact |
| Human-memory-dependent rules should not stay implicit | Partial | `docs/android/ENGINEERING_PRINCIPLES.md`, `docs/android/REPOSITORY_GOVERNANCE.md` | No formal coverage map for memory-only rules | Promote the rule into docs, a validator, or explicit manual boundary |
| New validator or tool should be mapped to ownership | Partial | `docs/android/DEVELOPMENT_WORKFLOW.md`, `docs/android/ARCHITECTURE_REVIEW_PROCESS.md` | No single place to see coverage impact | Update this matrix whenever a validator or tool is added |
| Project health summary | Future | None yet | No `acl doctor` surface in Phase 6A | Keep manual review only for now |
| Evidence audit | Future | None yet | No `acl evidence audit` surface in Phase 6A | Use manual evidence review and findings docs |

## Notes

- The matrix should stay synchronized with new validators and recurring
  lessons.
- If a rule becomes repeatedly relevant, promote it into the coverage matrix
  or the debt register instead of leaving it as a private working habit.
- This document should not grow into a large taxonomy. If a rule needs more
  than a short row, it probably belongs in a dedicated policy doc.
