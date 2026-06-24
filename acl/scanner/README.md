# ACL Scanner

The scanner module inspects ELF binaries and produces structured compatibility
reports for the Android Compatibility Layer (ACL) pipeline.

## Pipeline

```
InspectFile → ClassifyFile → FindMissingSymbols → Recommend → ScanPaths
```

| Stage | Function | Output |
|---|---|---|
| Inspect | `InspectFile(path)` | `*ELFInfo` (class, machine, PT_INTERP, RPATH/RUNPATH, DT_NEEDED) |
| Classify | `ClassifyFile(path)` | `CompatCategory` + `*ELFInfo` |
| Symbols | `FindMissingSymbols(info)` | `[]MissingSymbol` (glibc-only DT_NEEDED entries) |
| Recommend | `Recommend(cat, info)` | `PatchRecommendation` (concrete patch action + suggested values) |
| Scan | `ScanPaths(paths)` | `ScanReport` (full JSON document) |

## Compatibility Categories

| Category | Meaning |
|---|---|
| `native Android compatible` | Runs as-is; Bionic/Termux linker detected |
| `Linux/glibc executable` | Needs patching; glibc `ld-linux` detected |
| `static ELF` | No dynamic linking; no patching required |
| `script` | Shebang detected; ELF patching not applicable |
| `unknown` | Could not classify |
| `unsupported` | Windows PE or other non-Android-patchable format |

## Patch Actions

| Action | When applied |
|---|---|
| `no-action` | Binary is already Android-compatible |
| `rewrite-interpreter` | PT_INTERP is a glibc linker; RPATH already OK |
| `inject-rpath` | Interpreter OK; RPATH missing ACL runtime path |
| `rewrite-interpreter-and-rpath` | Both PT_INTERP and RPATH need updating |
| `script-no-elf-patch` | Script; ELF patching not applicable |
| `unsupported` | Cannot be patched for Android use |

## JSON Report Schema (v1.0)

```json
{
  "schema_version": "1.0",
  "generated_at": "<RFC3339>",
  "binaries": [
    {
      "path": "/path/to/binary",
      "compat_category": "Linux/glibc executable",
      "elf": {
        "class": "ELF64",
        "machine": "EM_AARCH64",
        "interpreter": "/lib/ld-linux-aarch64.so.1",
        "rpath": "",
        "runpath": "",
        "needed": ["libc.so.6", "libpthread.so.0"],
        "is_static": false
      },
      "missing_symbols": [
        {
          "name": "libc.so.6",
          "library": "libc.so.6",
          "reason": "glibc libc — Bionic provides libc.so, not libc.so.6"
        }
      ],
      "recommendation": {
        "action": "rewrite-interpreter-and-rpath",
        "suggested_interpreter": "/data/data/com.termux/files/usr/lib/acl-runtime/loader/ld-linux-aarch64.so.1",
        "suggested_rpath": "/data/data/com.termux/files/usr/lib/acl-runtime/lib",
        "rationale": "..."
      },
      "error": ""
    }
  ],
  "summary": {
    "total": 1,
    "native_android": 0,
    "linux_glibc": 1,
    "static": 0,
    "script": 0,
    "unknown": 0,
    "unsupported": 0,
    "errors": 0,
    "needs_patch": 1
  }
}
```

## Usage via acl-scan

```bash
# Human-readable text report (default)
acl-scan compat /path/to/binary [...]

# Machine-readable JSON report
acl-scan --output json compat /path/to/binary [...]
acl-scan compat-json /path/to/binary [...]          # always JSON

# Validate (exit 1 if any binary needs patching or is unknown)
acl-scan validate-compat /path/to/binary [...]
acl-scan validate-compat-json /path/to/binary [...]  # always JSON

# The --output flag may appear before or after the sub-command name:
acl-scan --output json compat /path/to/binary
acl-scan compat --output json /path/to/binary
acl-scan --output=json compat /path/to/binary
```

## Exit Codes

| Code | Meaning |
|---|---|
| 0 | All binaries are Android-compatible (no patching needed) |
| 1 | One or more binaries need patching, are unknown, or had errors |
| 2 | Usage / argument error |
| 3 | Internal error (I/O, JSON marshal failure) |

## Using the Go Package

```go
import "github.com/arduino/arduino-cli/acl/scanner"

// Inspect a single file.
info, err := scanner.InspectFile("/usr/bin/avr-gcc")

// Classify a single file.
cat, info, err := scanner.ClassifyFile("/usr/bin/avr-gcc")

// Find glibc-only DT_NEEDED entries that Bionic cannot satisfy.
missing := scanner.FindMissingSymbols(info)

// Get a concrete patch recommendation.
rec := scanner.Recommend(cat, info)

// Full pipeline over a list of files.
report := scanner.ScanPaths([]string{"/usr/bin/avr-gcc", "/usr/bin/esptool.py"})

// Serialize to indented JSON.
data, err := scanner.MarshalReport(report)
```

## Testing

```bash
# Go unit and integration tests
go test ./acl/scanner/...

# Regenerate golden files
UPDATE_GOLDEN=1 go test ./acl/scanner/...

# Shell integration tests (requires acl-scan binary and jq)
bash acl/tests/test-scanner-json.sh

# Regenerate shell golden files
bash acl/tests/test-scanner-json.sh --update-golden
```
