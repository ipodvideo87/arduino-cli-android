# Technical Debt Policy

Technical debt is not always a mistake. Sometimes it is a deliberate tradeoff
that allows the repository to move forward while keeping the next step
manageable.

The goal is not perfection. The goal is to keep debt visible, intentional, and
small enough that it does not become a permanent tax on the project.

## Acceptable Technical Debt

Debt is acceptable when it is:

- explicitly documented
- bounded in scope
- temporary or transitional
- understood by the team
- not hiding a broken contract
- not blocking safer work later

Examples include:

- a short-lived compatibility shim
- a temporary fallback while a better interface is validated
- a prototype that is clearly marked as experimental

## Unacceptable Technical Debt

Debt is unacceptable when it:

- obscures ownership
- hides uncertainty as fact
- becomes the default path without review
- creates repeated manual work with no plan to automate it
- duplicates responsibilities unnecessarily
- blocks validation of the real target environment
- makes future maintenance harder in a way the repository has not acknowledged

## Temporary Workarounds

Workarounds are only acceptable when they are clearly labeled and time-bounded.

A workaround should answer:

- why it exists
- what problem it avoids
- what the real fix is
- what condition ends the workaround

If a workaround becomes permanent without review, it is no longer a workaround.

## Documenting Debt

Debt should be documented where future contributors will actually look for it.

Documentation should capture:

- the debt itself
- the reason it exists
- the risk it introduces
- the plan to retire it
- the evidence needed before retirement

## Retiring Debt

Retire debt when the repository has enough evidence or architecture maturity to
replace it safely.

Retirement should be deliberate:

- remove the workaround
- update the docs
- validate the replacement
- preserve historical context only if it is still useful

## Preventing Recurring Debt

If the same problem appears more than once, the repository should not merely
fix it again.

Instead, ask whether the system should evolve so the problem is harder to
repeat. That may mean:

- adding an interface
- clarifying a boundary
- strengthening validation
- improving documentation
- automating a repeated step
- removing an ambiguous path

## Prototypes Versus Production Work

Prototypes are allowed when they help answer a real question.

But a prototype is not production work until it has:

- a clear ownership boundary
- documented limitations
- a validation path
- a retirement or promotion decision

Do not let prototype behavior quietly become a contract.

## When Refactoring Should Occur

Refactor when the current shape makes correct work harder than it should be.

Good refactoring is usually triggered by one of these signs:

- duplicated logic
- unclear ownership
- awkward validation
- repeated bugs in the same area
- a contract that has outgrown its current shape

Refactoring should reduce future maintenance burden. It should not be used to
chase polish for its own sake.

## Debt Management Principle

The repository should prefer visible, bounded debt over hidden, sprawling debt.

Visible debt can be retired. Hidden debt tends to accumulate.
