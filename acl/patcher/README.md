# ACL Patcher

The **patcher** stage computes and applies ELF edits that make glibc-linked
binaries runnable under the ACL runtime on Android/Termux.

## Two modes of operation

| Mode | Flag | Effect |
|---|---|---|
| **Dry-run** | `--dry-run` (default) | Compute the plan, print it, touch nothing |
| **Apply** | `--apply` | Compute the plan, print it, call `patchelf` |

The dry-run plan is always printed first, even during `--apply`, so operators
can audit every change in CI logs before it is committed to disk.

---

## Dry-run output format

The plan is rendered as a human-readable diff, one block per ELF file that
needs patching:

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

Key conventions:

- `---` lines show the **current** value in the binary.
- `+++` lines show the **proposed** replacement value.
- `<absent>` means the ELF field is not currently set.
- Files that need no patching are hidden in quiet mode; use `--verbose` to
  show them.
- Skipped files (wrong architecture, GCC libexec, non-ELF) are also hidden
  unless `--verbose` is given.

---

## acl-exec — the CLI front-end

```
acl-exec [--dry-run] [options] <file-or-dir> [...]
acl-exec --apply    [options] <file-or-dir> [...]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--dry-run` | yes (default) | Print plan, no file changes |
| `--apply` | — | Print plan, then call patchelf |
| `--runtime <dir>` | `$ACL_RUNTIME_DIR` or `./runtime` | ACL runtime directory |
| `--loader <name>` | `ld-linux-aarch64.so.1` | Loader basename inside `--runtime` |
| `--verbose` | off | Print status for every file |
| `--color` | auto-detect | Force ANSI colour |
| `--no-color` | — | Disable ANSI colour |
| `--help` | — | Show help |

### Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | One or more apply-phase errors |
| 2 | Usage error |

### Examples

```sh
# Preview all patches for a GCC toolchain (nothing modified):
acl-exec --dry-run \
    --runtime /data/acl/runtime \
    --verbose \
    /opt/gcc/

# Apply patches:
acl-exec --apply \
    --runtime /data/acl/runtime \
    /opt/gcc/bin/gcc \
    /opt/gcc/bin/g++

# CI pipeline — fail if any file needs patching:
acl-exec --dry-run --runtime /data/acl/runtime /opt/gcc/
# Check $? or grep the summary line for "0 need patching".
```

---

## Go API

```go
import "github.com/arduino/arduino-cli/acl/patcher"

// Compute plans without touching files.
plans := patcher.ComputePlans(paths, patcher.PlanOptions{
    RuntimeDir: "/data/acl/runtime",
})

// Render as a diff plan to any io.Writer.
patcher.WritePlan(os.Stdout, plans, patcher.FormatOptions{
    Color:   true,
    Verbose: false,
})

// Check the summary programmatically.
summary := patcher.Summarise(plans)
if summary.NeedPatching > 0 {
    // … take action
}

// Apply (calls patchelf).
result, err := patcher.Apply(paths, patcher.ApplyOptions{
    PlanOptions: patcher.PlanOptions{RuntimeDir: "/data/acl/runtime"},
    DryRun:      false,
})
```

### Key types

| Type | Purpose |
|---|---|
| `PatchPlan` | All edits for one ELF file |
| `FieldEdit` | Single field change (Field, Current, Proposed, Reason) |
| `PlanSummary` | Aggregate counts across a set of plans |
| `PlanOptions` | RuntimeDir, LoaderName |
| `DryRunOptions` | PlanOptions + Out, Color, Verbose |
| `ApplyOptions` | PlanOptions + DryRun, Out, Color, Verbose |
| `ApplyResult` | Plans, Applied, Skipped, Errors |

---

## Classification rules

| Binary type | Action |
|---|---|
| Not an ELF file | Skipped |
| Not aarch64 (EM_AARCH64) | Skipped |
| GCC libexec (`/libexec/gcc/`) | Skipped (use wrapper-launch instead) |
| PT_INTERP = Android/Bionic linker | No patch needed |
| PT_INTERP = glibc linker | Set PT_INTERP + RUNPATH |
| No PT_INTERP, glibc DT_NEEDED | Set RUNPATH only |
| Legacy DT_RPATH present | Migrate to RUNPATH via `--set-rpath` |

---

## GCC libexec note

GCC internal executables under `libexec/gcc/` must **not** be rewritten with
`patchelf --set-interpreter`.  The ELF layout change caused by patchelf
crashes them on native Termux.  They are handled by the **wrapper-launch**
strategy: the original binary is moved to `.acl/original/<name>` and replaced
with a shell wrapper that invokes the runtime loader explicitly.

See [`acl/docs/PATCHING.md`](../docs/PATCHING.md) for the full rationale.

---

## Integration tests

```sh
bash acl/tests/test-patcher-dryrun.sh
```

The test script:
1. Builds `acl-exec` from source.
2. Generates minimal ELF fixtures with `go run acl/tests/testdata/gen_test_elfs.go`.
3. Runs 20 assertions covering exit codes, output content, file integrity,
   flags, and environment variable handling.
