# ACL Tests

This directory contains integration and verification scripts for the Android
Compatibility Layer (ACL).

## Test Scripts

| Script | Purpose |
|---|---|
| `fresh-clone-verify.sh` | Full ACL fresh-clone verification (branch-level) |
| `test-esptool.sh` | esptool compatibility on Android/Termux |
| `test-launcher.sh` | ACL launcher backend smoke tests |
| `test-patcher.sh` | ELF patcher plan/apply safety checks |
| `test-runtime.sh` | ACL runtime install/list/validate lifecycle |
| `test-scanner-json.sh` | **JSON report output** from `acl-scan --output json` |

## test-scanner-json.sh

Validates that `acl-scan` produces correct machine-readable JSON reports.

### Dependencies

- `acl-scan` binary (build: `go build -o bin/acl-scan ./acl/cmd/acl-scan/`)
- `jq` (JSON query tool)
- `bash` ≥ 4

### Usage

```bash
# Run all JSON scanner tests
bash acl/tests/test-scanner-json.sh

# Regenerate golden files (only when intentionally changing the report schema)
bash acl/tests/test-scanner-json.sh --update-golden
```

### Tests covered

| ID | Fixture | Assertion |
|---|---|---|
| T01 | Shell script (`#!/bin/sh`) | category=script, action=script-no-elf-patch, exit 0 |
| T02 | Windows PE (MZ header + `.exe`) | category=unsupported, action=unsupported, exit 0 |
| T03 | `compat-json` sub-command | always emits valid JSON regardless of `--output` |
| T04 | `validate-compat-json` sub-command | emits `{valid, report}` JSON; valid=true for script |
| T05 | Multi-file (script + exe) | summary.total=2, correct per-category counts |
| T06 | Any file | Required top-level fields present (`schema_version`, `generated_at`, `binaries`, `summary`) |
| T07 | Any file | Required per-binary fields present (`path`, `compat_category`, `recommendation.{action,rationale}`) |
| T08 | `--output=json` (equals form) | Valid JSON produced |
| T09 | `compat --output json` (flag after sub-command) | Valid JSON produced |
| T10 | Unknown binary (random bytes) | category=unknown, exit 1 |
| T11 | `validate-compat` + unknown binary | exit 1 (unknown → invalid) |
| T12 | No arguments | exit 2 (usage error) |

### Golden files

Golden files live in `acl/tests/testdata/golden/`.  They store the expected
JSON output (with `generated_at` stripped for determinism) for fixtures whose
output should be stable across versions.

> **Note**: Golden file paths contain `__FIXTURE_PATH__` placeholders because
> the actual fixture paths are temp-dir paths that differ per run.  The shell
> test uses `jq 'del(.generated_at)'` normalisation and field-level assertions
> rather than full-document diff for path-containing fields.

## Running all ACL tests

```bash
# Go unit tests
go test ./acl/...

# Shell integration tests
bash acl/tests/test-scanner-json.sh
bash acl/tests/test-runtime.sh
bash acl/tests/test-patcher.sh
bash acl/tests/test-launcher.sh
```

## Adding new tests

1. Add a new script following the `test-<component>.sh` naming convention.
2. Source `acl/lib/common.sh` for `acl::info`, `acl::warn`, `acl::die`.
3. Document the script in this README.
4. If the test produces JSON, add golden files under `testdata/golden/` and
   use `--update-golden` to seed them.
