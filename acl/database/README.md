# ACL Database

This directory holds starter metadata files used by ACL compatibility checks and
planning.

Each JSON file is intentionally small in v0.1 so the project has a valid schema anchor while
the runtime and patching behavior is still being filled in.

Tool compatibility reporting currently uses code-level heuristics in the ACL scanner.
Future compatibility rules should be added in a structured way so they can eventually be
stored and versioned here alongside the rest of the ACL metadata.
