# ACL Tests

These scripts verify ACL-specific repository state and targeted workflows.

Fresh-clone verification:

```bash
ACL_VERIFY_BRANCH=android-runtime-v2 bash acl/tests/fresh-clone-verify.sh
```

`fresh-clone-verify.sh` verifies repository reproducibility only. It clones a fresh
checkout, rebuilds `arduino-cli` plus ACL command binaries present in that branch, runs
ACL Go tests, and checks `git diff --check`.

The development workflow intentionally keeps the verification clone at
`~/acl-verification` after the script completes. During active development, that makes
it easier to inspect the exact fresh-clone repository that was built and tested, instead
of losing it to automatic cleanup.

If `~/acl-verification` already exists, the script prompts before deleting it. It never
removes that directory without explicit confirmation.

It does not:

- install Arduino cores
- remove Arduino state
- patch ESP32 tools
- execute Linux ELF binaries through ACL
- compile sketches
- test real Android execution

Planned and targeted tests:

- ELF parsing
- Runtime verification
- Dependency resolution
- Android compatibility
