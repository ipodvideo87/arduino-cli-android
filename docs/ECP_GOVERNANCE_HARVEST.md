# ECP Governance Harvest — EOS / Project Zero Source Material

> **Status: NON-CANONICAL SOURCE MATERIAL / PRESERVATION WORKSPACE**
>
> This document is a deliberate harvest/staging area for EOS, Project Zero,
> `AGENTS.md`, and repository-governance material that may be useful to the
> Engineering Control Plane (ECP).
>
> **Nothing in this file is adopted ECP policy merely because it appears here.**
> It does not amend EOS, `AGENTS.md`, repository governance, or the ECP Lawset.
> Authoritative wording remains in the original source files until an explicit
> ECP review accepts, adapts, or rejects a concept.
>
> EOS/ECP integration is intentionally deferred until ECP is functional,
> testable, internally coherent, adversarially validated, and piloted. This file
> exists so useful engineering doctrine can be preserved now and evaluated later
> from evidence instead of memory.

## 1. Purpose

The goal of this file is deliberately conservative:

1. preserve wording and decisions that appear potentially reusable by ECP;
2. copy especially valuable source material wholesale where that avoids losing
   context;
3. distinguish universal engineering doctrine from Android/Arduino-specific
   project rules;
4. prevent useful EOS/Project Zero work from being lost while ECP is developed
   independently;
5. defer actual adoption until ECP has a concrete architecture and functioning
   implementation against which the material can be tested.

The intended future classification vocabulary is:

- `ADOPT` — suitable for ECP substantially as written;
- `ADAPT` — principle is useful but wording/scope must change for ECP;
- `ALREADY_COVERED` — ECP already expresses the requirement elsewhere;
- `PROJECT_SPECIFIC` — useful to `arduino-cli-android`, not universal ECP doctrine;
- `DEFER` — worth preserving, but premature for current ECP architecture;
- `REJECT` — intentionally not carried forward;
- `UNASSESSED` — preserved here but not yet adjudicated.

All harvested material starts as `UNASSESSED` unless this document explicitly
states otherwise.

## 2. Source Snapshot

Repository: `ipodvideo87/arduino-cli-android`

Branch: `android-runtime-v2`

Harvest date: 2026-08-13

Primary source set:

- `AGENTS.md`
- `eos.project.json`
- `docs/android/MILESTONE_EOS_PROJECT_ZERO.md`
- `docs/android/ENGINEERING_PRINCIPLES.md`
- `docs/android/ENGINEERING_INVARIANTS.md`
- `docs/android/DECISION_FRAMEWORK.md`
- `docs/android/ENGINEERING_KNOWLEDGE_FRAMEWORK.md`
- `docs/android/ENGINEERING_LIFECYCLE.md`
- `docs/android/CONFIDENCE_MODEL.md`
- `docs/android/UNCERTAINTY_REGISTER.md`
- `docs/android/VALIDATION_POLICY.md`
- `docs/android/DIAGNOSTIC_VALIDATION_STANDARD.md`
- `docs/android/DOCUMENTATION_ARCHITECTURE.md`
- `docs/android/REPOSITORY_GOVERNANCE.md`
- `docs/android/ARCHITECTURE_REVIEW_PROCESS.md`
- `docs/android/INTERFACE_STABILITY_POLICY.md`
- `docs/android/TECHNICAL_DEBT_POLICY.md`
- `docs/android/ENGINEERING_METHODOLOGY.md`
- `docs/android/LESSONS_LEARNED.md`
- `docs/android/ENGINEERING_DECISIONS.md`
- `docs/android/DECISION_LOG.md`
- `docs/android/CODEX_OPERATING_MODEL.md`

Known `AGENTS.md` source blob at harvest time:

`b3a9e5e8f46cc646e102772a7cc8513e7ab28416`

Additional EOS source files may be appended to this harvest later. Their
presence here still must not be interpreted as ECP adoption.

---

# PART I — WHOLESALE SOURCE PRESERVATION

## 3. `AGENTS.md` — Engineering Laws

The following block is preserved wholesale from the current source section.
It is source material, not an ECP Lawset.

```text
## Engineering Laws

These laws are constitutional. They have highest priority over every other
repository guide, policy, workflow, and convention.

1. Never Wing It
   - Do not guess when evidence is available.
   - Research the repository, relevant documentation, implementation, and
     authoritative sources when necessary before making assumptions,
     conclusions, or implementation choices.
   - If a statement cannot be supported by repository evidence, implementation
     evidence, validation evidence, research, or established engineering facts,
     it must not be presented as fact.
   - If evidence is insufficient, continue researching, name the uncertainty,
     ask clarifying questions when necessary, and delay implementation until the
     decision is informed.
   - If evidence conflicts, treat the conflict itself as evidence and investigate
     it until the discrepancy is understood or explicitly documented.

2. Architecture Before Implementation
   - Protect long-term architecture over short-term convenience.

3. Evidence Before Claims
   - Never claim more than has actually been validated.

4. Validate Before Completion
   - Completion requires appropriate validation, not merely successful builds or
     passing host tests.

5. Preserve Engineering Knowledge
   - Every meaningful task should leave the repository in a more knowledgeable
     state than before.

6. Stop at the Smallest Proven Milestone
   - Prefer small, fully validated milestones over large speculative
     implementations.

7. Build Generic Solutions
   - Prefer reusable, maintainable, interface-driven designs over special-case
     implementations whenever practical.

These laws govern every future engineering task, decision, implementation,
review, validation, and documentation update. If another repository document
conflicts with them, the laws win.
```

### Initial ECP interest

All seven are worth later ECP review. In particular:

- `Never Wing It` strongly overlaps ECP epistemic integrity;
- `Architecture Before Implementation` fits ECP's current design-first phase;
- `Evidence Before Claims` is a concise operational expression of epistemic
  discipline;
- `Validate Before Completion` reinforces separation of implementation, testing,
  validation, authorization, execution, and demonstrated outcome;
- `Preserve Engineering Knowledge` is directly relevant to ECP's durable shared
  state and institutional memory;
- `Stop at the Smallest Proven Milestone` matches the ECP goal of building a
  real, testable, coherent system before speculative expansion;
- `Build Generic Solutions` supports ECP's model-agnostic, interface-driven
  architecture.

**Classification: UNASSESSED.**

## 4. `AGENTS.md` — EOS Project Zero boundary

Preserved wholesale:

```text
## EOS Project Zero

This repository is the first real EOS adoption pilot.

- `eos.project.json` is the canonical project adoption manifest.
- `AGENTS.md` is a project-local overlay, not the canonical EOS contract.
- Universal engineering methodology should defer to the EOS canon when EOS has
  a canonical home for the concept.
- Android-specific implementation guidance, workflow guidance, and repository
  hygiene still belong here when they are project-specific.
```

For ECP, this is historical/source evidence only. The current ECP decision is
**not** to design ECP around EOS before ECP is functional.

**Classification: DEFER for EOS integration; UNASSESSED for reusable boundary ideas.**

## 5. `eos.project.json` — wholesale snapshot

```json
{
  "schema_version": "1",
  "project_name": "arduino-cli-android",
  "eos_version": "0.1.0",
  "compatibility": {
    "core_contracts": [
      "decision-framework",
      "evidence-model",
      "confidence-model",
      "uncertainty-register",
      "validation-standard",
      "recovery-protocol",
      "engineering-lifecycle",
      "engineering-knowledge-framework",
      "quality-model",
      "execution-model",
      "project-adoption-model"
    ],
    "model_adapters": [],
    "execution_adapters": [],
    "status": "compatible"
  },
  "project_extensions": [
    "android-first engineering",
    "native Termux is the source of truth for Android claims",
    "real hardware is required for upload, flash, and runtime claims",
    "transport and upload architecture remain board-agnostic and descriptor-driven",
    "project-local operational memory remains in STATUS.md, ROADMAP.md, TASK_RECOVERY.md, and VALIDATED_FINDINGS.md"
  ],
  "overlay": {
    "agents_md": "AGENTS.md",
    "status": "present"
  },
  "validation": {
    "required_checks": [
      "schema validation",
      "canonical ownership review",
      "documentation boundary review",
      "native Termux validation for Android claims",
      "real hardware validation for upload, flash, and runtime claims"
    ],
    "native_validation_required": true
  }
}
```

The named EOS contract families are especially useful as a later review index:

- decision framework;
- evidence model;
- confidence model;
- uncertainty register;
- validation standard;
- recovery protocol;
- engineering lifecycle;
- engineering knowledge framework;
- quality model;
- execution model;
- project adoption model.

Do **not** infer that ECP needs an equivalent one-to-one module for each name.
They are source concepts to compare against real ECP needs later.

**Classification: UNASSESSED.**

## 6. `ENGINEERING_PRINCIPLES.md` — wholesale snapshot

```text
# Engineering Principles

These are the durable engineering principles for this repository.
They elaborate the Engineering Laws at the top of `AGENTS.md`.

## Principles

- Android-first development
  - Treat Android and native Termux as the primary design target.

- Native Termux as source of truth
  - When Android behavior and host behavior differ, native Termux wins.

- Evidence before claims
  - Do not elevate a conclusion above the evidence that supports it.

- Architecture before implementation
  - Understand subsystem boundaries and contracts before adding behavior.

- Research before assumptions
  - Investigate project history, current docs, and target-environment behavior
    before guessing.

- Interfaces before concrete implementations
  - Define stable boundaries first so implementations can evolve behind them.

- Production-quality architecture over quick fixes
  - Prefer durable structure over the fastest local patch.

- Separation of concerns
  - Keep discovery, permission, connection, protocol, upload, monitor,
    diagnostics, and UI responsibilities distinct.

- Single source of truth where practical
  - Canonical definitions, policies, standards, and contracts should have one
    home.

- Local context is allowed for readability
  - Small summaries and reminders are good when they help a reader stay on the
    page.

- Automation over repeated manual steps
  - Convert recurring operational work into reusable workflows, tests, or
    scripts.

- Reduce future maintenance burden
  - Prefer changes that make the next change simpler and safer.

- Stop at the smallest useful milestone
  - Resolve the current engineering uncertainty before adding extra features or
    scope.
  - Do not expand the work just because more functionality could fit nearby.

- If a mistake can happen twice, make it harder to happen a third time
  - Add structure, validation, tests, or documentation that prevents the same
    failure mode from repeating.

- Every task should leave the repository easier to understand, validate, or
  maintain
  - Even small changes should improve clarity, evidence, or boundaries.

- Preserve compatibility when practical
  - Avoid unnecessary breaking changes when additive evolution is possible.

- Prefer board-agnostic and descriptor-driven designs
  - Do not hardcode device-family assumptions when the problem can be modeled
    generically.

- Document what the repository proves, not what it merely hopes to prove
  - Keep planned and experimental behavior labeled as such.
```

The Android/Termux-specific items should remain project-specific unless a later
ECP generalization is justified. The underlying claim-scoping principle may be
reusable: validation authority should match the environment and scope of the
claim.

**Classification: UNASSESSED.**

## 7. `ENGINEERING_KNOWLEDGE_FRAMEWORK.md` — wholesale snapshot

```text
# Engineering Knowledge Framework

This document defines how engineering work becomes permanent repository
knowledge.

The framework is meant to make every completed task answer the same set of
questions in a durable, version-controlled form:

- What question were we trying to answer?
- What evidence was collected?
- What engineering decision was made?
- What uncertainty was eliminated?
- What uncertainty remains?
- What confidence changed?
- What lesson was learned?
- Which documentation was updated?
- How did this influence the roadmap?

## Purpose

The repository should not only accumulate implementation. It should accumulate
repeatable engineering knowledge that makes the next decision easier to make
and harder to repeat incorrectly.

This framework exists to:

- preserve evidence with context
- keep uncertainty visible instead of implicit
- make decisions traceable
- separate validated knowledge from speculation
- promote durable lessons into canonical docs
- reduce repeated investigation

## Canonical Ownership

The engineering knowledge system has multiple canonical homes with distinct
responsibilities:

- `ENGINEERING_DECISIONS.md` owns long-lived ADR-style decisions
- `VALIDATED_FINDINGS.md` owns evidence-backed findings and validation notes
- `DECISION_LOG.md` owns task-level decision provenance
- `UNCERTAINTY_REGISTER.md` owns open questions and closure criteria
- `CONFIDENCE_MODEL.md` owns confidence semantics and change rules
- `LESSONS_LEARNED.md` owns durable lessons distilled from repeated evidence

This document owns the policy that connects those artifacts into one lifecycle.
It does not replace their individual responsibilities.

## Required Closeout Questions

Any substantive engineering task should leave behind enough material to answer
the following:

1. What question were we trying to answer?
2. What evidence did we collect?
3. What decision did we make?
4. What uncertainty was removed?
5. What uncertainty remains?
6. What confidence changed?
7. What lesson did we learn?
8. What documentation changed?
9. What roadmap impact followed?

If the task cannot answer these questions, the task is not fully closed from a
knowledge perspective.

## Artifact Routing

Use the smallest artifact that owns the knowledge:

- short-lived uncertainty belongs in `UNCERTAINTY_REGISTER.md`
- a concrete task decision belongs in `DECISION_LOG.md`
- a durable architecture decision belongs in `ENGINEERING_DECISIONS.md`
- evidence and validation outcome belong in `VALIDATED_FINDINGS.md`
- reusable guidance belongs in `LESSONS_LEARNED.md`
- confidence interpretation belongs in `CONFIDENCE_MODEL.md`

## Promotion Rules

Knowledge should move through the repository deliberately:

1. Capture the active question and the current uncertainty.
2. Gather evidence in the appropriate validation environment.
3. Record the decision and confidence change.
4. Close or narrow the uncertainty.
5. Promote any durable lesson into canonical guidance.
6. Update status and roadmap only when the change affects the project view.

Do not promote speculative reasoning directly into a canonical rule.

## Knowledge Promotion Ladder

Use the following path when engineering knowledge becomes more durable:

1. Observation: raw evidence, command output, or local inspection.
2. Validated Finding: evidence-backed conclusion in `VALIDATED_FINDINGS.md`.
3. Lesson Learned: reusable guidance in `LESSONS_LEARNED.md`.
4. Engineering Invariant: non-negotiable process rule in `ENGINEERING_INVARIANTS.md`.
5. Automation: tooling or validation that enforces the rule repeatedly.

The ladder is directional, not automatic. Promote only when the new layer is
actually warranted by repeated evidence, future reuse, or recurring friction.
Do not skip directly from a one-off observation to an invariant unless the
pattern has already proven durable enough to justify the stronger rule.

## Operating Rule

Every completed engineering task should leave the repository with:

- one more verified fact
- one fewer unresolved question, or a clearly narrowed one
- one explicit decision trail
- one clearer confidence boundary
- one updated doc trail

That is the mechanism by which repository knowledge compounds over time.
```

This source is particularly relevant to ECP because ECP is explicitly intended
to externalize authoritative engineering state and preserve attributable
knowledge across human/AI execution contexts.

**Classification: UNASSESSED.**

---

# PART II — HIGH-VALUE SOURCE MATERIAL TO REVIEW NEXT

## 8. Copy-wholesale candidates

The following source documents are intentionally listed before any ECP rewrite.
When this harvest is expanded, prefer preserving the source text first and only
then writing an ECP adaptation beside it.

### Evidence, confidence, and uncertainty

- `docs/android/CONFIDENCE_MODEL.md`
  - separates validation level from confidence;
  - confidence rises with evidence quality, target relevance, reproducibility,
    independent confirmations, and clearer scope;
  - confidence falls when environments change, contradictions appear, or claims
    broaden beyond evidence.

- `docs/android/UNCERTAINTY_REGISTER.md`
  - keeps unknowns visible rather than allowing them to disappear;
  - gives each uncertainty a question, evidence need, owner, next validation,
    and closure condition.

- `docs/android/VALIDATION_POLICY.md`
  - separates evidence levels;
  - requires evidence strength to match the claim being made;
  - explicitly prevents lower validation levels from being collapsed into
    higher ones.

- `docs/android/DIAGNOSTIC_VALIDATION_STANDARD.md`
  - requires claims to identify execution, environment, evidence, supported
    claims, unsupported claims, remaining unknowns, and next validation;
  - separates implemented/integrated/validated/production-ready states.

### Decision and architecture governance

- `docs/android/DECISION_FRAMEWORK.md`
  - considers user intent, mission/status, architecture, evidence, history,
    maintenance cost, validation cost, and extension points;
  - warns against collapsing separate layers into one function;
  - distinguishes user intent from implementation strategy.

- `docs/android/ARCHITECTURE_REVIEW_PROCESS.md`
  - makes architecture decisions deliberate rather than accidental;
  - reviews ownership, dependencies, boundaries, interfaces, duplication,
    validation, testing, and long-term maintenance before implementation.

- `docs/android/REPOSITORY_GOVERNANCE.md`
  - requires deliberate evolution, explicit ownership, evidence, migrations,
    and one canonical home for long-lived decisions;
  - treats repository health as more than green checks.

- `docs/android/INTERFACE_STABILITY_POLICY.md`
  - prefers additive, version-aware evolution;
  - treats serialized data and internal contracts as interfaces;
  - explicitly warns against collapsing distinct responsibilities into opaque
    abstractions.

### Knowledge ownership and historical truth

- `docs/android/DOCUMENTATION_ARCHITECTURE.md`
  - distinguishes canonical, operational, reference, and historical documents;
  - says the goal is not eliminating all overlap but eliminating duplicated
    ownership;
  - preserves historical material without allowing it to compete with current
    canonical truth.

- `docs/android/LESSONS_LEARNED.md`
  - promotes only reusable evidence-backed lessons;
  - keeps lessons from becoming a second policy source;
  - includes the useful pattern: solve recurring friction at the class level.

- `docs/android/ENGINEERING_DECISIONS.md`
  - contains durable ADR-style decisions;
  - includes the EOS manifest-first/overlay-only adoption decision;
  - includes the decision that the Engineering Knowledge Framework is the
    repository knowledge spine.

- `docs/android/DECISION_LOG.md`
  - preserves task-level path from question to evidence to decision;
  - explicitly records alternatives, confidence change, remaining uncertainty,
    docs changed, roadmap impact, and follow-up.

### Lifecycle, methodology, and debt

- `docs/android/ENGINEERING_LIFECYCLE.md`
  - frames question -> inspect evidence -> gather evidence -> compare -> decide ->
    record uncertainty -> update confidence -> capture lesson -> update docs ->
    close task.

- `docs/android/ENGINEERING_METHODOLOGY.md`
  - optimizes for sustained correctness and long-term maintainability rather
    than short-term throughput;
  - treats evidence as an engineering input, not a postscript;
  - treats automation as a maintenance strategy rather than a substitute for
    human judgment.

- `docs/android/TECHNICAL_DEBT_POLICY.md`
  - permits visible, bounded, intentional debt;
  - rejects hidden debt that obscures ownership or uncertainty;
  - prohibits prototypes from quietly becoming production contracts.

### Project Zero source

- `docs/android/MILESTONE_EOS_PROJECT_ZERO.md`
  - should be reviewed substantially or wholesale before any future ECP/EOS
    integration work;
  - preserves the adoption charter, success/non-goals, review gates, rollback
    points, canonical ownership table, EOS learning model, risks, remaining
    uncertainty, and recommended first slice.

**Classification for this entire section: UNASSESSED / PRESERVE FOR REVIEW.**

---

# PART III — INITIAL ECP HARVEST INDEX

## 9. Candidate concepts worth mapping into ECP

| Source concept | Initial status | Possible ECP destination |
| --- | --- | --- |
| Never Wing It | UNASSESSED | Constitution / Engineering Methodology |
| Architecture Before Implementation | UNASSESSED | Engineering Principles |
| Evidence Before Claims | UNASSESSED | Constitution / Assurance; derived from epistemic integrity |
| Validate Before Completion | UNASSESSED | Assurance / State semantics |
| Preserve Engineering Knowledge | UNASSESSED | Knowledge Governance / Audit |
| Stop at the Smallest Proven Milestone | UNASSESSED | Engineering Methodology |
| Build Generic Solutions | UNASSESSED | Architecture Principles |
| Research Before Assumptions | UNASSESSED | Methodology |
| Interfaces Before Concrete Implementations | UNASSESSED | Architecture Principles |
| Production-quality architecture over quick fixes | UNASSESSED | Architecture Principles |
| Separation of concerns | UNASSESSED | Architecture / privileged-operation separation |
| One canonical owner per engineering claim | UNASSESSED | Information Architecture |
| Automation over recurring manual work | UNASSESSED | Methodology / Tooling |
| Make repeated mistakes harder to repeat | UNASSESSED | Root-cause / Regression Policy |
| Distinguish implemented/tested/validated/production-ready | UNASSESSED | State vocabulary / Assurance |
| Keep uncertainty explicit | UNASSESSED | Evidence / Findings / Uncertainty model |
| Confidence separate from validation level | UNASSESSED | Assurance / Epistemic state |
| Question -> evidence -> decision -> confidence -> lesson | UNASSESSED | Engineering lifecycle |
| Observation -> finding -> lesson -> invariant -> automation | UNASSESSED | Knowledge promotion model |
| Canonical / operational / reference / historical separation | UNASSESSED | Information Architecture |
| Deliberate additive interface evolution | UNASSESSED | Interface Governance |
| Visible bounded debt over hidden debt | UNASSESSED | Technical Debt Policy |
| Project memory split by responsibility | UNASSESSED | ECP object model / knowledge governance |
| Manifest-first integration boundary | DEFER | Future EOS/ECP integration |
| Overlay-only local instructions | DEFER | Future integration/adapters |

## 10. Project-specific material that must not be imported blindly

The following are valuable in `arduino-cli-android` but are not automatically
universal ECP rules:

- Android-first engineering;
- native Termux as source of truth;
- Bionic/glibc-specific validation policy;
- Android USB and transport-specific rules;
- board/device implementation details;
- canonical Termux repository paths;
- Arduino CLI/ACL command contracts;
- Codex-specific execution instructions;
- Project Zero's claim that EOS is the universal canonical owner.

A future ECP adaptation may generalize the reasoning behind some of these. For
example, `native Termux is the source of truth for Android claims` may generalize
to a claim-scope rule: **the authoritative validation environment must match the
claim being asserted.** That generalization still requires explicit ECP review.

## 11. ECP integration deferral

Current ECP direction:

```text
Build ECP as a real, testable, coherent system first.
        ↓
Validate its architecture and behavior.
        ↓
Run adversarial tests and a real engineering pilot.
        ↓
Establish what ECP actually proves, needs, and lacks.
        ↓
Only then evaluate EOS integration from evidence.
```

This file therefore exists to preserve future options without introducing
speculative EOS coupling into the ECP core.

## 12. Future harvest procedure

When ECP is ready for a formal EOS/governance comparison:

1. freeze the ECP architecture/version being evaluated;
2. freeze or identify the exact EOS/source revisions being compared;
3. review each preserved source concept independently;
4. classify it `ADOPT`, `ADAPT`, `ALREADY_COVERED`, `PROJECT_SPECIFIC`, `DEFER`,
   or `REJECT`;
5. record the rationale and evidence for that classification;
6. map accepted concepts to the correct ECP layer instead of automatically
   promoting them into Engineering Laws;
7. preserve rejected and deferred decisions so they are not repeatedly
   rediscovered;
8. make integration changes only where a functioning ECP demonstrates a real
   need.

## 13. Guardrail

This harvest deliberately preserves ideas more broadly than ECP will probably
adopt.

That is intentional.

**Preservation is cheap. Adoption is consequential.**

The purpose of this document is to make later selection evidence-based without
forcing today's ECP architecture to predict EOS's eventual role.
