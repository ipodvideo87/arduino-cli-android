# Project Context Document: arduino-cli-android

## Project Overview

**arduino-cli-android** is an experimental fork of the official Arduino CLI, engineered to run natively on Android devices without requiring chroots, PRoot, Docker, virtual machines, or a traditional Linux distribution. It is simultaneously the flagship implementation of the **Android Compatibility Layer (ACL)** — a reusable framework for running Linux developer tools reliably on Android/Termux environments.

The project has two primary objectives:
1. Make Arduino CLI a first-class citizen on Android/Termux (production target).
2. Build ACL as a general-purpose, reusable layer for adapting Linux ELF toolchains to the Android application sandbox.

**Key constraints**: Android userspace (Bionic libc, not glibc), Android kernel, SELinux enforcement, Android filesystem and permission rules, no traditional Linux distro layer.

---

## Development Environments

| Environment | Role | Authority |
|---|---|---|
| **Native Termux on Android** | Production target | **Final authority** — always wins on disagreements |
| **Ubuntu inside `proot-distro`** | Development/CI only | Useful for tooling; does NOT prove Android compatibility |

> **Critical rule**: When native Termux and Ubuntu/proot disagree, native Termux is the source of truth.

---

## Architecture

### High-Level Structure

```
arduino-cli-android/
├── acl/                    # Android Compatibility Layer (ACL) — core framework
│   ├── cmd/                # ACL command-line entry points
│   ├── builder/            # Runtime package assembly
│   ├── scanner/            # ELF inspection and compatibility classification
│   ├── patcher/            # ELF binary patching (planning + future rewrites)
│   ├── verifier/           # Pre-launch runtime assumption checks
│   ├── launcher/           # Execution backend (experimental)
│   ├── runtime/            # Runtime store contract and layout
│   ├── database/           # Starter metadata for compatibility checks
│   ├── tests/              # ACL-specific verification scripts
│   └── tools/              # Utility entry points and helper commands
├── internal/               # Arduino CLI internals (upstream + Android additions)
│   ├── cli/                # CLI command implementations (cobra-based)
│   ├── integrationtest/    # Integration test suite
│   ├── locales/            # i18n data
│   └── android/            # Android-specific runtime assets
│       └── bootstrap/      # Relocatable data-directory bootstrap for Termux PREFIX
├── rpc/                    # gRPC/protobuf definitions and client examples
├── debian/                 # Debian packaging (Dockerfile-based)
├── docs/                   # Documentation, ADRs, Android research
└── .github/workflows/      # CI/CD pipeline definitions
```

### ACL Module Architecture

The ACL is designed as a staged pipeline:

```
[ELF Binary / Arduino Package]
         │
         ▼
    ┌─────────┐
    │ Scanner │  → Inspect ELF (class, machine, SONAME, PT_INTERP,
    └─────────┘    RPATH/RUNPATH, imports, absolute paths)
         │         → Detect shebangs and classify script interpreter compatibility
         │         → Classify compatibility category
         │         → Assign patch class
         │         → Emit human-readable OR machine-readable JSON report
         ▼
    ┌──────────┐
    │ Verifier │  → Check runtime directory exists
    └──────────┘  → Verify expected glibc-style libraries present
         │         → Confirm ELF looks relocatable (not Termux-path-locked)
         │         → Android filesystem pre-flight checks (noexec, SELinux)
         │         → SELinux enforcement state detection
         │         → Writable/executable path validation
         ▼
    ┌─────────┐
    │ Patcher │  → Plan or apply ELF edits (interpreter, RPATH)
    └─────────┘  → Dry-run mode: emit unified diff-style patch plan (no writes)
         │         → Apply mode: invoke patchelf to rewrite ELF fields
         │         → Requires `patchelf` to be available for apply mode
         ▼
    ┌──────────┐
    │ Launcher │  → Bootstrap runtime environment
    └──────────┘  → Execute tools in controlled environment (experimental)
```

### ACL Scanner — Shebang and Script Interpreter Compatibility

The scanner now includes a dedicated **shebang/script interpreter compatibility check** subsystem. When a file is classified as a `script` (via shebang detection), the scanner additionally:

1. **Extracts the shebang interpreter path** from the first line of the file.
2. **Classifies the interpreter** against a known compatibility matrix:
   - Interpreters available natively in Termux (e.g., `/usr/bin/python3`, `/bin/sh`, `/usr/bin/env`) → `compatible`
   - Interpreters with a known Termux alternative path (e.g., `/usr/bin/python` → `/data/data/com.termux/files/usr/bin/python3`) → `remappable`
   - Interpreters with no known Android/Termux equivalent → `unsupported`
   - Interpreters requiring a version check or conditional availability → `conditional`
3. **Emits structured interpreter check results** in both human-readable and JSON output modes.
4. **Produces patch recommendations** for remappable interpreters (analogous to ELF interpreter/RPATH recommendations).

#### Shebang Patch Actions

Script entries in the JSON report may now include shebang-specific patch actions:

| Action | Field | Meaning |
|---|---|---|
| `set-interpreter` | `interpreter` | Replace shebang interpreter path with Termux-compatible path |
| `none` | — | Shebang interpreter is already compatible or no change needed |

#### Script Compatibility Categories (Interpreter-Level)

| Category | Meaning |
|---|---|
| `compatible` | Interpreter available natively in Termux at the same path |
| `remappable` | Interpreter has a known Termux alternative; shebang rewrite recommended |
| `unsupported` | No known Android/Termux equivalent for this interpreter |
| `conditional` | Interpreter availability depends on installed Termux packages |
| `unknown` | Interpreter not in the compatibility matrix; manual review needed |

### ACL Scanner Output Modes

The scanner supports two output modes, both implemented as subcommands of `acl-scan`:

| Subcommand | Output Format | Primary Use |
|---|---|---|
| `acl-scan compat` | Human-readable text report | Interactive inspection, debugging |
| `acl-scan compat-json` | Machine-readable JSON report | Tooling, CI, patcher/launcher integration |
| `acl-scan validate-compat` | Human-readable validation results | Manual validation |
| `acl-scan validate-compat-json` | Machine-readable validation results | Automated validation |

#### Machine-Readable JSON Report Schema

The JSON report emitted by `acl-scan compat-json` follows a versioned schema:

```json
{
  "schema_version": "1.0",
  "generated_at": "<RFC3339 timestamp>",
  "target": "<scanned path>",
  "summary": {
    "total": 0,
    "by_category": {},
    "by_patch_class": {},
    "needs_patching": 0,
    "native_compatible": 0,
    "unsupported": 0
  },
  "entries": [
    {
      "path": "<relative or absolute path>",
      "category": "<compatibility category>",
      "patch_class": "<patch class>",
      "interpreter": "<PT_INTERP value or shebang interpreter path, or empty>",
      "rpath": "<RPATH/RUNPATH value or empty>",
      "recommendation": "<human-readable patch recommendation>",
      "interpreter_compat": {
        "interpreter_path": "<extracted shebang interpreter path>",
        "category": "<compatible|remappable|unsupported|conditional|unknown>",
        "recommended_path": "<Termux-compatible interpreter path, if remappable>",
        "reason": "<explanation>"
      },
      "patch_actions": [
        {
          "action": "<set-interpreter|set-rpath|none>",
          "field": "<interpreter|rpath>",
          "current_value": "<current ELF field or shebang value>",
          "recommended_value": "<suggested replacement value>",
          "reason": "<explanation>"
        }
      ]
    }
  ]
}
```

**Key design decisions for the JSON report:**
- `schema_version` is always present to allow consumers to detect breaking changes
- `patch_actions` is a structured list — consumers (patcher, launcher, CI) should iterate this list rather than parse free-text recommendations
- `recommendation` is a human-readable summary kept alongside `patch_actions` for debuggability
- Empty/nil `patch_actions` for entries that need no patching (e.g., `patch_class: none`, `static ELF`, `script` with compatible interpreter)
- Summary counters (`needs_patching`, `native_compatible`, `unsupported`) are pre-computed for quick pipeline decisions
- `interpreter_compat` is present only for `script` category entries; omitted for ELF entries
- `interpreter_compat.recommended_path` is only populated for `remappable` category interpreters

### ACL Builder Pipeline

```
[Loader + Libraries (inputs)]
         │
         ▼
    ┌─────────┐
    │ Builder │  → Assemble runtime package
    └─────────┘  → Write manifest.json, metadata.json, checksums.json
         │         → Verify package integrity
         ▼
    ┌──────────────┐
    │ Runtime Mgr  │  → Install, validate, select active runtime
    └──────────────┘  → Manage runtimes/<id>/ layout
```

### Android Bootstrap Package (Relocatable Data-Directory)

A key addition to `internal/android/bootstrap/` implements relocatable data-directory bootstrapping for Termux PREFIX environments. This addresses a fundamental Android constraint: Arduino CLI expects certain data files (board definitions, platform indexes, tool metadata) at well-known paths, but on Termux those paths must be resolved at runtime relative to `$PREFIX` (typically `/data/data/com.termux/files/usr`) rather than hardcoded to `/usr` or `/home`.

#### Bootstrap Design Principles

- **Relocatable-first**: All data-directory paths are derived from a single root (the Termux PREFIX or an explicitly provided override), never hardcoded to glibc/Linux defaults.
- **Lazy initialization**: The bootstrap runs once on first launch and is a no-op on subsequent launches if the layout is already correct.
- **Idempotent**: Re-running bootstrap on an already-bootstrapped installation must be safe and produce no duplicate entries or corrupted state.
- **Graceful degradation**: On non-Android/non-Termux hosts (Linux dev machines, CI), the bootstrap path logic falls back to standard XDG or OS-default paths, so the same binary works everywhere.

#### Bootstrap Resolution Order

The data directory is resolved in this priority order:

1. Explicit flag (`--data-dir` CLI flag or equivalent config key)
2. Environment variable (`ARDUINO_DATA_DIR` or `ARDUINO_DIRECTORIES_DATA`)
3. Termux PREFIX detection (`$PREFIX` env var, or heuristic scan of known Termux mount points)
4. XDG / OS default (standard Arduino CLI behavior for non-Android hosts)

#### Key Files

| Path | Purpose |
|---|---|
| `internal/android/bootstrap/bootstrap.go` | Core bootstrap logic: PREFIX detection, path resolution, directory layout initialization |
| `internal/android/bootstrap/bootstrap_test.go` | Unit tests for bootstrap path resolution and idempotency |
| `internal/android/bootstrap/paths.go` | Path constant definitions and helper functions for Termux-relative paths |
| `internal/android/bootstrap/detect.go` | Termux/Android environment detection heuristics |

---

## Key Files and Directories

### ACL Core

| Path | Purpose |
|---|---|
| `acl/cmd/acl-scan/main.go` | Scanner CLI entry point |
| `acl/cmd/acl-exec/main.go` | Execution planner CLI entry point |
| `acl/cmd/acl-runtime/main.go` | Runtime manager CLI entry point |
| `acl/cmd/acl-build-runtime/main.go` | Runtime package builder CLI entry point |
| `acl/scanner/` | ELF scanner implementation (classification, JSON report, patch recommendations) |
| `acl/scanner/shebang.go` | Shebang extraction and script interpreter compatibility classification |
| `acl/scanner/shebang_compat.go` | Known interpreter compatibility matrix (Termux path mappings) |
| `acl/scanner/shebang_test.go` | Unit tests for shebang detection and interpreter classification |
| `acl/patcher/` | ELF patcher implementation (dry-run plan + apply mode) |
| `acl/verifier/` | Pre-launch verifier implementation (runtime checks + Android filesystem/SELinux pre-flight) |
| `acl/database/android.json` | Android compatibility metadata |
| `acl/database/bionic.json` | Bionic ABI metadata |
| `acl/database/glibc.json` | glibc compatibility metadata |
| `acl/database/libraries.json` | Library compatibility index |
| `acl/database/toolchains.json` | Toolchain metadata |

### Android Bootstrap

| Path | Purpose |
|---|---|
| `internal/android/bootstrap/bootstrap.go` | Core bootstrap: PREFIX detection, path resolution, layout init |
| `internal/android/bootstrap/bootstrap_test.go` | Unit tests for path resolution and idempotency |
| `internal/android/bootstrap/paths.go` | Path constants and Termux-relative path helpers |
| `internal/android/bootstrap/detect.go` | Termux/Android environment detection heuristics |

### Documentation & Research

| Path | Purpose |
|---|---|
| `AGENTS.md` | AI agent guidelines and development rules |
| `MISSION.md` | High-level project mission and long-term direction |
| `STATUS.md` | Current progress snapshot |
| `acl/docs/ARCHITECTURE.md` | ACL architectural overview |
| `acl/docs/ADR-0001-acl-architecture-decision.md` | Architecture Decision Record #1 |
| `acl/docs/ADR-0002-runtime-manager.md` | Architecture Decision Record #2 |
| `acl/docs/AI_CONTEXT.md` | AI assistant context for ACL |
| `acl/docs/DESIGN.md` | ACL design document |
| `acl/docs/EXECUTION.md` | Execution model documentation |
| `acl/docs/LAYERING.md` | Layer separation documentation |
| `acl/docs/PATCHING.md` | ELF patching documentation |
| `acl/docs/RUNTIME.md` | Runtime documentation |
| `docs/android/ANDROID_COMPATIBILITY_RESEARCH.md` | **Must read before Android changes** |

### Arduino CLI Internals

| Path | Purpose |
|---|---|
| `internal/cli/config/` | Configuration management commands |
| `internal/cli/core/core.go` | Core board management commands |
| `internal/locales/` | i18n locale data |

### CI/CD

| Path | Purpose |
|---|---|
| `.github/workflows/build-android-cli.yml` | Android CLI build workflow |
| `.github/workflows/test-go-task.yml` | Go test runner |
| `.github/workflows/check-go-task.yml` | Go linting/checking |
| `.github/workflows/release-go-task.yml` | Release workflow |
| `Taskfile.yml` | Task runner definitions (used in place of Makefile) |

---

## Tech Stack

### Core Language
- **Go** (primary implementation language, 1,637 files, 3,456 functions)

### Key Go Dependencies

| Dependency | Purpose |
|---|---|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/spf13/viper` | Configuration management |
| `google.golang.org/grpc` | gRPC server/client |
| `google.golang.org/protobuf` | Protocol