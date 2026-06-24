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
         │         → Classify compatibility category
         │         → Assign patch class
         │         → Emit human-readable OR machine-readable JSON report
         ▼
    ┌──────────┐
    │ Verifier │  → Check runtime directory exists
    └──────────┘  → Verify expected glibc-style libraries present
         │         → Confirm ELF looks relocatable (not Termux-path-locked)
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
      "interpreter": "<PT_INTERP value or empty>",
      "rpath": "<RPATH/RUNPATH value or empty>",
      "recommendation": "<human-readable patch recommendation>",
      "patch_actions": [
        {
          "action": "<set-interpreter|set-rpath|none>",
          "field": "<interpreter|rpath>",
          "current_value": "<current ELF field value>",
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
- Empty/nil `patch_actions` for entries that need no patching (e.g., `patch_class: none`, `static ELF`, `script`)
- Summary counters (`needs_patching`, `native_compatible`, `unsupported`) are pre-computed for quick pipeline decisions

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
| `acl/patcher/` | ELF patcher implementation (dry-run plan + apply mode) |
| `acl/database/android.json` | Android compatibility metadata |
| `acl/database/bionic.json` | Bionic ABI metadata |
| `acl/database/glibc.json` | glibc compatibility metadata |
| `acl/database/libraries.json` | Library compatibility index |
| `acl/database/toolchains.json` | Toolchain metadata |

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
| `google.golang.org/protobuf` | Protocol Buffers |
| `github.com/go-git/go-git/v5` | Git operations |
| `github.com/sirupsen/logrus` | Structured logging |
| `github.com/leonelquinteros/gotext` | i18n/l10n |
| `go.bug.st/relaxed-semver` | Semantic versioning |
| `go.bug.st/downloader/v3` | File downloading |
| `github.com/mailru/easyjson` | Fast JSON serialization |
| `github.com/codeclysm/extract/v4` | Archive extraction |
| `github.com/ulikunitz/xz` | XZ compression |
| `github.com/klauspost/compress` | Additional compression (zstd) |
| `github.com/fatih/color` | Terminal color output |
| `dario.cat/mergo` | Struct/map merging |
| `go.bug.st/serial` | Serial port communication |

### Build & Tooling
- **Task** (Taskfile.yml) — primary build task runner
- **Python** (pyproject.toml) — documentation tooling (MkDocs via Poetry)
- **Node.js** (package.json) — Prettier formatting, markdown tooling
- **Protocol Buffers / gRPC** — RPC interface definitions
- **Docker** — Debian packaging

### External Android Tools
- **patchelf** — Required for ELF patching operations (apply mode only; dry-run requires no external tools)
- **Termux** — Android terminal environment (production target)
- **proot-distro** — Ubuntu development environment (dev only)

---

## ACL Compatibility Classification System

### Compatibility Categories
- `native Android compatible` — runs as-is on Android
- `Linux/glibc executable` — needs patching or runtime wrapping
- `static ELF` — no dynamic linking issues
- `script` — shell, Python, etc. (shebang-detected)
- `unknown` — classification failed
- `unsupported` — Windows `.exe` or other foreign binaries

### Patch Classes
- `none` — no patching needed
- `loader-and-rpath` — both PT_INTERP and RPATH need updating
- `rpath-only` — only RPATH needs updating
- `runtime-dependency-only` — treat as runtime dep, not executable
- `script-no-elf-patch` — script, no ELF patching applicable
- `unsupported` — cannot be patched for Android use

### Patch Actions (JSON report)

Each scanner JSON entry may contain structured `patch_actions`. Known action types:

| Action | Field | Meaning |
|---|---|---|
| `set-interpreter` | `interpreter` | Replace PT_INTERP with recommended Android/Termux loader path |
| `set-rpath` | `rpath` | Replace RPATH/RUNPATH with recommended runtime library path |
| `none` | — | No ELF edit required for this entry |

Consumers of the JSON report (patcher, launcher, CI scripts) must iterate `patch_actions` rather than parse free-text `recommendation` strings.

### ELF Patcher: Dry-Run and Apply Modes

The patcher operates in two explicit modes:

| Mode | Behavior | External Tools Required |
|---|---|---|
| **Dry-run** (default) | Emits a unified diff-style patch plan showing what would change; no files written | None |
| **Apply** | Invokes `patchelf` to rewrite PT_INTERP and/or RPATH/RUNPATH fields in-place | `patchelf` must be on PATH |

**Dry-run output conventions:**
- Output resembles a unified diff: each file that would be patched is shown with `---` (current value) and `+++` (proposed value) lines for each ELF field to be changed
- Files requiring no changes are summarized but not shown in diff output
- Exit code 0 even when patches are planned (dry-run never fails due to planned changes)
- Suitable for piping into review workflows or CI gate checks

**Apply mode conventions:**
- Requires explicit opt-in flag (e.g., `--apply`) — dry-run is the safe default
- Operates on scanner JSON report (`patch_actions`) as its authoritative input, not free-text recommendations
- Each `patch_action` entry is executed in order: `set-interpreter` before `set-rpath`
- Failures on individual files are reported but do not abort the full batch (continue-on-error within a run)
- Apply mode should be run only after verifier confirms the runtime is present and valid

**Design principle:** The patcher is a consumer of scanner JSON output — it reads `patch_actions` arrays and does not re-classify ELF files itself.

### ACL Runtime Package Format
```
<package>/
├── manifest.json      # Runtime contract (schema_version, runtime_id, arch, ABIs, loader, libraries, hashes, compatibility_level, created_at, build, extensions)
├── metadata.json      # Builder metadata and package format version
├── checksums.json     # File integrity map
├── version            # Package format version marker
├── loader/            # Dynamic linker/loader
└── lib/               # Runtime shared libraries
```

### Installed Runtime Store Layout
```
<runtime-root>/
├── active.json
└── runtimes/
    └── <runtime-id>/
        ├── manifest.json
        ├── loader/
        ├── lib/
        └── metadata/
```

---

## Coding Conventions

### General Go Conventions
- Standard Go project layout with `internal/` for private packages
- Cobra for all CLI commands with subcommand structure
- Viper for configuration with support for YAML, TOML, JSON, dotenv
- Logrus for structured logging throughout
- gRPC/protobuf for the RPC layer between CLI client and daemon

### ACL-Specific Conventions
- **Android-specific code is isolated** — keep Android compatibility concerns separate from upstream Arduino