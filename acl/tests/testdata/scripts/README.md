# Script Fixtures for ACL Scanner Shebang Tests

This directory contains script fixture files used by the ACL scanner's shebang
interpreter validation tests.

Each file is a minimal valid script with a specific shebang line chosen to cover
a distinct code path in `acl/scanner/shebang.go`.

## Fixtures

| File | Shebang | Expected status (fake prefix) | Test scenario |
|---|---|---|---|
| `bash_script.sh` | `#!/bin/bash` | `remapped` | Absolute `/bin/bash` → Termux remap |
| `usr_bin_bash_script.sh` | `#!/usr/bin/bash` | `remapped` | Alternate absolute bash path |
| `env_bash_script.sh` | `#!/usr/bin/env bash` | `remapped` | env-style delegation to bash |
| `python3_script.py` | `#!/usr/bin/python3` | `remapped` | Absolute Python 3 path |
| `env_python3_script.py` | `#!/usr/bin/env python3` | `remapped` | env-style delegation to python3 |
| `perl_script.pl` | `#!/usr/bin/perl` | `remapped` | Absolute Perl path |
| `env_perl_script.pl` | `#!/usr/bin/env perl` | `remapped` | env-style delegation to perl |
| `env_unknown_script.sh` | `#!/usr/bin/env notarealinterpreter` | `missing` | Unknown env delegate not installed |
| `no_shebang_script.sh` | *(none)* | nil result | File with no shebang line |

## Status meanings

| Status | Meaning |
|---|---|
| `found` | Interpreter exists at its declared path |
| `remapped` | Interpreter located at a different (Termux-relative) path |
| `missing` | Interpreter not found; install recommendation provided |
| nil result | File has no shebang; no interpreter validation performed |

## Running the tests

```bash
# Go unit tests (scanner package)
go test ./acl/scanner/...

# Go integration tests (fixtures)
go test ./acl/tests/...

# Shell smoke tests (requires jq)
./acl/tests/test-scanner-json.sh
```
