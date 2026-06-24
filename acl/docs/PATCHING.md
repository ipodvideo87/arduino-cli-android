# ACL ELF Patching

This document describes how the Android Compatibility Layer (ACL) patches
ELF binaries to make glibc-linked tools run on Android/Termux without a
chroot or Linux distribution layer.

---

## Why patching is needed

Stock Android ships the **Bionic** C library, not glibc.  Binaries compiled
for Linux distributions (e.g., GCC toolchains distributed via Arduino packages)
contain two glibc-specific ELF fields that prevent them loading on Android:

| ELF field | Typical glibc value | Problem |
|---|---|---|
| `PT_INTERP` | `/lib/ld-linux-aarch64.so.1` | The path does not exist on Android |
| `RUNPATH` / `RPATH` | `/lib:/usr/lib` | Glibc library paths don't exist on Android |

The ACL runtime ships a compatible loader and library set.  The patcher
rewrites these two fields so they point at the ACL runtime directory instead
of system glibc paths.

---

## Dry-run mode — review before you change

> **The patcher's default mode is dry-run.**  Nothing is modified until you
> explicitly pass `--apply`.

### CLI — `acl-exec`

```sh
# Preview every patch that would be applied (default / explicit --dry-run):
acl-exec --dry-run \
    --runtime /data/acl/runtime \
    /opt/gcc/

# Apply after reviewing the plan:
acl-exec --apply \
    --runtime /data/acl/runtime \
    /opt/gcc/
```

### Shell shim — `patch-elf.sh`

```sh
# Dry-run (default):
bash acl/patcher/patch-elf.sh /opt/gcc/bin/gcc

# Apply:
bash acl/patcher/patch-elf.sh --apply /opt/gcc/bin/gcc
```

### Dry-run output format

The plan is formatted as a unified diff to make the before/after state
immediately legible:

```
=== /opt/gcc/bin/gcc ===
--- PT_INTERP   (current)  /lib/ld-linux-aarch64.so.1
+++ PT_INTERP   (proposed) /data/acl/runtime/ld-linux-aarch64.so.1
    reason: binary uses a glibc dynamic linker; replace with ACL runtime loader
--- RUNPATH     (current)  <absent>
+++ RUNPATH     (proposed) /data/acl/runtime
    reason: set RUNPATH to ACL runtime directory so the loader finds glibc-world libraries

── Dry-run summary: 4 file(s) — 1 need patching, 2 already OK, 1 skipped ──
```

Conventions:

- `---` = current value in the binary (`<absent>` if the field is not set).
- `+++` = value that would be written by `patchelf`.
- Files that need no changes are hidden in quiet mode; use `--verbose` to see
  them.
- Skipped files (wrong arch, GCC libexec, non-ELF) are also hidden by default.

---

## Supported patch operations

### 1. `set-interpreter` (PT_INTERP)

Replaces the dynamic linker path hard-coded in the binary.

```
patchelf --set-interpreter /data/acl/runtime/ld-linux-aarch64.so.1 <binary>
```

Applied to: executables with a non-Android `PT_INTERP`.

### 2. `set-rpath` (RUNPATH)

Sets `DT_RUNPATH` to the ACL runtime directory and removes the old `DT_RPATH`.

```
patchelf --set-rpath /data/acl/runtime <binary>
```

Applied to: any binary with glibc-world `DT_NEEDED` entries or a glibc
interpreter.

### 3. `remove-rpath` (RPATH removal)

Removes a legacy `DT_RPATH` when a correct `RUNPATH` is already set.

```
patchelf --remove-rpath <binary>
```

Applied to: binaries that have `DT_RPATH` but not `DT_RUNPATH`.

---

## GCC libexec — wrapper launch instead of patchelf

GCC internal executables under `libexec/gcc/` (e.g., `cc1`, `lto1`,
`collect2`) **must not** be patched with `patchelf --set-interpreter`.

**Reason**: patchelf rewrites ELF segments to fit the new interpreter path.
The expanded segment layout crashes the kernel ELF loader on native Termux
when it tries to map the binary into memory.  This has been confirmed
experimentally; the failure mode is a silent `SIGBUS` at startup.

**Solution — wrapper launch**: the original binary is moved to
`.acl/original/<name>` and a small shell wrapper is written in its place:

```sh
#!/bin/sh
set -eu
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
runtime_dir="$script_dir/../../../../.acl/runtime"
loader="$runtime_dir/ld-linux-aarch64.so.1"
target="$script_dir/.acl/original/cc1"
# Strip environment variables that confuse the glibc runtime:
unset LD_PRELOAD LD_LIBRARY_PATH LD_AUDIT TERMUX_VERSION ...
exec "$loader" --library-path "$runtime_dir" "$target" "$@"
```

The patcher **skips** GCC libexec binaries and marks them with
`SkipReason: "GCC libexec binary: requires wrapper-launch …"`.
The `elf_plan.go` layer in `internal/android/` handles the actual wrapper
creation.

---

## Classification rules

| Condition | Patch action |
|---|---|
| Not an ELF file (script, etc.) | Skip |
| ELF, not aarch64 | Skip |
| ELF, path contains `/libexec/gcc/` | Skip (wrapper-launch) |
| PT_INTERP = Android/Bionic linker | No patch needed |
| PT_INTERP = glibc linker | Set PT_INTERP + RUNPATH |
| No PT_INTERP, glibc DT_NEEDED entries | Set RUNPATH only |
| DT_RPATH present (legacy) | Migrate to RUNPATH (`--set-rpath`) |

---

## Go API

The patcher is implemented as a Go package:

```go
import "github.com/arduino/arduino-cli/acl/patcher"

// 1. Compute plans (read-only, no filesystem changes):
plans := patcher.ComputePlans(paths, patcher.PlanOptions{
    RuntimeDir: "/data/acl/runtime",
    LoaderName: "ld-linux-aarch64.so.1", // optional, this is the default
})

// 2. Inspect plans:
for _, plan := range plans {
    if plan.NeedsPatching() {
        for _, edit := range plan.Edits {
            fmt.Printf("%s: %s: %q → %q (%s)\n",
                plan.Path, edit.Field,
                edit.Current, edit.Proposed, edit.Reason)
        }
    }
}

// 3. Render as human-readable diff (dry-run output):
patcher.WritePlan(os.Stdout, plans, patcher.FormatOptions{
    Color:   true,
    Verbose: false,
})

// 4. Aggregate statistics:
summary := patcher.Summarise(plans)
fmt.Printf("%d need patching, %d OK, %d skipped\n",
    summary.NeedPatching, summary.AlreadyOK, summary.Skipped)

// 5. Apply (calls patchelf):
result, err := patcher.Apply(paths, patcher.ApplyOptions{
    PlanOptions: patcher.PlanOptions{RuntimeDir: "/data/acl/runtime"},
})
```

---

## Requirements

- **Go 1.21+** to build `acl-exec`.
- **patchelf** in `PATH` for apply mode only.  Dry-run mode has no external
  dependencies.

---

## Integration tests

```sh
bash acl/tests/test-patcher-dryrun.sh
```

Runs 20 assertions covering:
- Exit codes (0 for success, 2 for usage errors).
- Output content (correct field names, current/proposed values, reason text).
- File integrity (SHA-256 checked before and after dry-run).
- Flag behaviour (`--verbose`, `--no-color`, `--loader`, `--runtime`).
- Environment variable handling (`ACL_RUNTIME_DIR`).
- Directory expansion (recursive ELF collection).
- GCC libexec and wrong-architecture skipping.
