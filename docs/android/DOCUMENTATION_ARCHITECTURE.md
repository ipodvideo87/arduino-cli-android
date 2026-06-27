# Documentation Architecture

This document defines document ownership and how repository docs should relate
to each other.

The goal is not to eliminate all overlap. The goal is to eliminate duplicated
ownership.

## Canonical Information

Formal definitions, policies, standards, procedures, validation levels,
architecture contracts, interfaces, and long-lived technical decisions should
have exactly one canonical home.

Other documents may summarize those topics for readability, but they should
reference the canonical section instead of maintaining an independent copy.

## Document Categories

### Canonical

Canonical documents own the durable rules, contracts, and decisions that other
docs should defer to.

Examples:

- mission
- status
- validation policy
- diagnostic validation standard
- architecture contracts
- engineering decisions

### Operational

Operational documents describe current work, procedures, milestones, checklists,
and status.

They may summarize canonical rules in a small amount of local context, but they
should not redefine those rules.

### Reference

Reference documents explain systems, concepts, design intent, or supporting
knowledge.

They can help readers understand the topic on the current page without forcing
them to jump to another document every few paragraphs.

### Historical

Historical documents preserve prior decisions, research logs, retired branch
reviews, and evidence from past work.

They should remain readable, but they should not compete with current canonical
sources of truth.

## When To Reference Instead Of Repeat

Use references when the content is:

- a formal definition
- a policy
- a checklist
- a specification
- an interface contract
- a validation standard
- something likely to evolve independently

## When To Repeat

It is acceptable to repeat:

- brief summaries
- small reminders
- short contextual explanations
- one or two paragraph introductions

provided they do not redefine or become a competing source of truth.

## Local Context

Small amounts of local context are encouraged when they improve readability.

Readers should be able to understand a document on its own, but they should not
expect every page to restate the canonical definitions in full.

## Avoid Documentation Mazes

Documentation should never become a chain of references where every page only
says `see another document`.

Each document should:

- state its own responsibility
- summarize enough context to be readable
- defer detailed definitions to the canonical source when appropriate

## Ownership Rules

Each document should have a clear responsibility.

Examples:

- `STATUS.md` reports project status.
- `ROADMAP.md` describes future work.
- `VALIDATION_POLICY.md` defines validation scope.
- `DIAGNOSTIC_VALIDATION_STANDARD.md` defines reporting requirements.
- architecture documents describe architecture.
- mission documents explain project goals.

Those documents may summarize each other where helpful, but they should not
duplicate ownership of another document's responsibility.

## Classifying New Documents

When adding a new document, decide whether it is:

- canonical
- operational
- reference
- historical

Then write the document so it clearly owns that responsibility and does not
quietly take ownership of another document's job.

If a new document needs to mention a canonical concept, it should link to the
canonical source rather than rewriting the definition.

## Preserving Historical Docs

Historical docs should be preserved when they capture evidence, decisions, or
prior architecture that may still be useful later.

Keep them explicitly historical so they remain available without becoming
competing sources of truth.

If a historical document is still useful, that is a reason to keep it readable,
not a reason to let it redefine current policy.
