# EOS Project Zero

Project Zero is the first real EOS adoption pilot for `arduino-cli-android`.
It exists to test EOS against a production engineering repository and to teach
EOS what this repository can prove, not just what EOS can prescribe.

## Charter

### Mission

Prove that EOS can govern a real production repository without duplicating
policy, collapsing ownership, or losing project-specific engineering needs.

### Scope

- project adoption boundary
- canonical ownership classification
- `eos.project.json`
- `AGENTS.md` overlay role
- adoption validation criteria
- compatibility reporting
- repository-to-EOS learning loop

### Success Criteria

- the repository has one canonical EOS adoption manifest
- `AGENTS.md` is clearly overlay-only
- major concepts have one primary owner
- the adoption boundary is objective and repeatable
- EOS can state what it learned from the repository
- the repository can state what it learned from EOS

### Non-Goals

- runtime code changes
- transport/upload/flash implementation changes
- broad documentation rewrites
- migration of every local doc into EOS
- overfitting EOS to Android-specific implementation details

### Exit Criteria

- the manifest exists and validates
- ownership is classified
- adoption friction is documented
- compatibility is explicit
- the next adoption step is clear and evidence-based

## Adoption Strategy

The safest adoption strategy is staged and review-first:

1. classify concepts by canonical owner
2. pin EOS in `eos.project.json`
3. keep `AGENTS.md` as a thin local overlay
4. document compatibility and local extensions
5. verify the boundary with static review before any migration
6. preserve rollback by keeping EOS adoption additive until the boundary is
   proven stable

### Review Gates

- canonical ownership review
- manifest/schema review
- documentation boundary review
- compatibility review
- evidence review

### Rollback Points

- `AGENTS.md` starts duplicating EOS policy again
- the manifest becomes path-coupled instead of contract-coupled
- validation requirements become vague
- EOS starts absorbing project-specific Android rules that should remain local

### Validation Requirements

- schema validation for the manifest
- document consistency review
- explicit ownership classification
- native Termux validation for Android claims
- real hardware validation for upload, flash, and runtime claims

### Acceptance Criteria

- a new contributor can tell what is EOS-owned versus repo-owned
- a future session can resume without guessing the adoption boundary
- local guidance remains readable without becoming a second constitution

## Required Adoption Artifacts

The minimum evidence-backed artifact set is:

- `eos.project.json`
- `AGENTS.md` overlay notes
- adoption/compatibility review in this milestone doc
- canonical ownership classification
- validation checklist or equivalent validation notes

Additional artifacts are only justified if the pilot proves they are needed.

## Adoption Validation Model

EOS adoption is successful when the following are true:

- the manifest pins EOS explicitly
- supported EOS contracts are named
- project-specific extensions are explicit
- `AGENTS.md` stays local and non-canonical
- repository documentation does not duplicate EOS doctrine
- the adoption boundary is understandable without hidden assumptions
- the repository can explain what it learned from EOS
- EOS can explain what it learned from the repository

This is a static, evidence-based validation model for the pilot. Runtime or
hardware validation is not required for the adoption boundary itself, but the
repository's Android claims still require the normal Android validation path.

## Canonical Ownership Review

| Concept | Canonical Owner | Operational Owner | Historical Owner | Evidence Owner |
| --- | --- | --- | --- | --- |
| Universal engineering methodology | EOS canon | `AGENTS.md` as overlay | `STATUS.md` / milestone docs | EOS decision records |
| Android-specific implementation policy | `AGENTS.md` / Android docs | Android docs | milestone docs | `VALIDATED_FINDINGS.md` |
| Current project state | `STATUS.md` | `STATUS.md` | prior statuses | `STATUS.md` |
| Future ordering | `docs/android/ROADMAP.md` | `docs/android/ROADMAP.md` | prior roadmap entries | roadmap notes |
| Live interruption context | `docs/android/TASK_RECOVERY.md` | `docs/android/TASK_RECOVERY.md` | cleared recovery snapshots | recovery history |
| Evidence-backed findings | `docs/android/VALIDATED_FINDINGS.md` | `VALIDATED_FINDINGS.md` | archived findings | `VALIDATED_FINDINGS.md` |
| Decision provenance | `docs/android/DECISION_LOG.md` | `DECISION_LOG.md` | earlier decisions | `DECISION_LOG.md` |
| Durable architecture decisions | `docs/android/ENGINEERING_DECISIONS.md` | `ENGINEERING_DECISIONS.md` | ADR history | `ENGINEERING_DECISIONS.md` |
| Project adoption contract | `eos.project.json` | `eos.project.json` | adoption history | milestone report |
| Overlay instructions | `AGENTS.md` | `AGENTS.md` | previous AGENTS revisions | AGENTS and status docs |

## Engineering Memory Assessment

What the repository already does well:

- separates current state from roadmap and recovery
- keeps evidence and decision history separate
- preserves native Termux as the source of truth for Android claims
- records durable findings instead of relying on chat memory

What EOS should learn:

- adoption works better as a manifest than as embedded prose
- overlay docs are useful when they stay local
- project memory should be split into state, evidence, decision, and lesson
  artifacts instead of one large catch-all document

## EOS Learning Model

### Concepts validated

- manifest-first adoption
- overlay-only `AGENTS.md`
- explicit version pinning
- explicit compatibility declarations
- keep repository state separate from evidence

### Concepts refined

- whether manifest metadata should include a first-class safety/constraints field
- how much local Android guidance should remain in `AGENTS.md`
- how to classify docs as canonical, operational, historical, or reference

### Concepts rejected

- `AGENTS.md` as the canonical adoption contract
- path-coupled adoption
- duplicating EOS doctrine in project-local policy files

### New canonical concepts

- project adoption manifest
- manifest-first adoption boundary
- overlay-only local instructions

### Project-specific concepts

- Android-first validation
- native Termux as source of truth
- real hardware required for upload/flash/runtime claims
- board-agnostic transport and upload architecture

## Future Adoption Model

Project Zero should become the template for future EOS adoption by proving a
repeatable sequence:

1. classify repository concepts
2. pin EOS in a project manifest
3. declare local extensions and constraints
4. keep AGENTS overlay-only
5. review ownership and compatibility
6. record friction and evidence
7. update EOS only when the learning is reusable

This template must stay repository-agnostic so that future adopters do not have
to resemble `arduino-cli-android`.

## Risks

- overfitting EOS to Android
- letting `AGENTS.md` silently grow back into a second constitution
- under-specifying manifest safety or compatibility fields
- treating review output as adoption success without clear validation criteria

## Remaining Uncertainty

- whether the manifest needs a dedicated safety/constraints field before EOS
  Foundation v0.1.0
- whether future repositories will need mandatory model-adapter declarations
- how much of the current Android operating model should stay local versus move
  into EOS canon

## Confidence

Medium-high.

The boundary is strong enough to pilot now, but the exact manifest shape still
needs evidence from adoption use.

## Recommended First Implementation Slice

The smallest useful slice is:

1. keep `eos.project.json` as the adoption contract
2. keep `AGENTS.md` explicit about being overlay-only
3. keep the milestone doc as the adoption assessment
4. update status and roadmap so the pilot is visible
5. preserve any reusable EOS learning in canonical EOS docs only when the
   evidence supports it
