# ACL Scanner

The ACL Scanner inspects ELF binaries and script files for Android/Termux
compatibility and emits structured reports consumed by the patcher, verifier,
and launcher.

## Compatibility categories

| Category | Meaning |
|---|---|
| `native Android compatible` | Targets Android/Bionic; no patching needed |
| `Linux/glibc executable` | Needs interpreter and/or RPATH patching |
| `static ELF` | Statically linked; no dynamic-linker concerns |
| `script` | Shebang-detected file (bash, python, perl, …) |
| `unknown` | Classification failed |
| `unsupported` | Foreign format (Windows `.exe`, etc.) |

## Script shebang validation

When a file is classified as `script`, the scanner also validates the
interpreter declared on the shebang line (`#!`) against:

1. **Literal path** — does the path exist on the current filesystem?
2. **ACL runtime directory** — is the interpreter's basename present under
   `<runtimeDir>/bin/`?
3. **`/usr/bin/env` delegation** — env-style shebangs (`#!/usr/bin/env python3`)
   are resolved by looking up the delegate name (`python3`) in a built-in
   Termux path table and then checking for it under `<prefixDir>/bin/`.
4. **Well-known Termux path table** — common Linux interpreter paths (e.g.
   `/usr/bin/python3`, `/bin/bash`) are mapped to their Termux-relative
   equivalents under `<prefixDir>/`.
5. **Basename fallback** — try `<prefixDir>/bin/<basename>` for any unrecognised
   path.
6. **Missing** — nothing found; a recommendation string explains what to install.

### `interpreter_status` field

Each script entry in the JSON report carries an `interpreter_status` object:

```json
{
  "declared_path": "/usr/bin/python3",
  "args": [],
  "status": "remapped",
  "resolved_path": "/data/data/com.termux/files/usr/bin/python3",
  "recommendation": "Interpreter \"/usr/bin/python3\" is a standard Linux path; Termux equivalent found at \"...\". Update the shebang or use a wrapper."
}
```

`status` is one of:

| Value | Meaning |
|---|---|
| `found` | Interpreter exists at its declared path |
| `remapped` | Interpreter located at a different (Termux-relative) path |
| `missing` | Interpreter not found anywhere; action required |

### Summary counters

The report summary includes per-script interpreter counters:

```json
{
  "script_interpreter_found":    0,
  "script_interpreter_missing":  1,
  "script_interpreter_remapped": 3
}
```

## JSON report schema

```json
{
  "schema_version": "1.0",
  "generated_at": "<RFC3339>",
  "target": "<scanned path>",
  "summary": { ... },
  "entries": [
    {
      "path": "...",
      "category": "script",
      "patch_class": "script-no-elf-patch",
      "recommendation": "Script file; ELF patching is not applicable.",
      "interpreter_status": {
        "declared_path": "/bin/bash",
        "status": "remapped",
        "resolved_path": "/data/data/com.termux/files/usr/bin/bash",
        "recommendation": "..."
      }
    }
  ]
}
```

## API

```go
import "github.com/arduino/arduino-cli/acl/scanner"

// Build a report from a directory scan.
b := scanner.NewReportBuilder(target, prefixDir, runtimeDir)
b.ScanDirectory("/path/to/tools")
report := b.Build()
scanner.WriteJSON(os.Stdout, report)

// Low-level shebang API.
result, err := scanner.ScanShebang("/path/to/script.py", prefixDir, "")
// result.Status is one of InterpreterFound / InterpreterMissing / InterpreterRemapped
```

## Test fixtures

Script fixtures live in `../tests/testdata/scripts/`:

| File | Shebang | Purpose |
|---|---|---|
| `bash_script.sh` | `#!/bin/bash` | Absolute bash path |
| `usr_bin_bash_script.sh` | `#!/usr/bin/bash` | Alternative bash path |
| `env_bash_script.sh` | `#!/usr/bin/env bash` | env-style bash |
| `python3_script.py` | `#!/usr/bin/python3` | Absolute python3 |
| `env_python3_script.py` | `#!/usr/bin/env python3` | env-style python3 |
| `perl_script.pl` | `#!/usr/bin/perl` | Absolute perl |
| `env_perl_script.pl` | `#!/usr/bin/env perl` | env-style perl |
| `env_unknown_script.sh` | `#!/usr/bin/env notarealinterpreter` | Unknown delegate |
| `no_shebang_script.sh` | *(none)* | No shebang line |

Run the Go unit tests:

```bash
go test ./acl/scanner/...
```

Run the fixture integration tests:

```bash
go test ./acl/tests/...
```

Run the shell-based smoke tests (requires `jq`):

```bash
./acl/tests/test-scanner-json.sh
```
