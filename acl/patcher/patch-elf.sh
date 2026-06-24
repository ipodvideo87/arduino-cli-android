#!/usr/bin/env bash
# patch-elf.sh — thin shell shim for the ACL ELF patcher.
#
# This script delegates to the Go-implemented `acl-exec` binary which provides
# the authoritative dry-run and apply logic.  When acl-exec is not in PATH,
# a human-readable fallback message is printed.
#
# Usage:
#   patch-elf.sh [--apply] [--verbose] [--no-color] <elf-file-or-dir> [...]
#
# Flags:
#   (none) / --dry-run   Print the patch plan without modifying files (default).
#   --apply              Apply patches via patchelf after printing the plan.
#   --verbose            Show status for every file, including already-OK entries.
#   --no-color           Disable ANSI colour output.
#   --runtime <dir>      ACL runtime directory (default: $ACL_RUNTIME_DIR or ./runtime).
#   --help               Show this help.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Try to source the common library (not required — fail gracefully).
if [[ -f "$SCRIPT_DIR/../lib/common.sh" ]]; then
    # shellcheck source=../lib/common.sh
    source "$SCRIPT_DIR/../lib/common.sh"
else
    acl::info()  { echo "[INFO]  $*"; }
    acl::warn()  { echo "[WARN]  $*" >&2; }
    acl::die()   { local code="$1"; shift; echo "[ERROR] $*" >&2; exit "$code"; }
fi

usage() {
    cat <<'EOF'
Usage: patch-elf.sh [--apply] [options] <elf-file-or-dir> [...]

Default behavior prints a dry-run patch plan only (no files modified).

Options:
  --apply            Apply patches using patchelf.
  --dry-run          Explicit dry-run (default).
  --verbose          Show status for every file.
  --no-color         Disable ANSI colour.
  --runtime <dir>    ACL runtime directory.
  --help             Show this help and exit.

The patch plan shows --- (current) and +++ (proposed) values for every ELF
field that would be changed (PT_INTERP, RUNPATH, RPATH).

This script delegates to 'acl-exec' when available.  Install or build it with:
  go build -o acl-exec ./acl/cmd/acl-exec/
EOF
}

# ── Parse flags ───────────────────────────────────────────────────────────────
APPLY=0
VERBOSE=0
NO_COLOR=0
RUNTIME=""
TARGETS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --apply)    APPLY=1;          shift ;;
        --dry-run)                    shift ;;  # accepted; default behaviour
        --verbose)  VERBOSE=1;        shift ;;
        --no-color) NO_COLOR=1;       shift ;;
        --runtime)
            if [[ $# -lt 2 ]]; then
                acl::die 2 "--runtime requires an argument"
            fi
            RUNTIME="$2"; shift 2 ;;
        -h|--help)  usage; exit 0 ;;
        --)         shift; TARGETS+=("$@"); break ;;
        -*)         acl::die 2 "unknown option: $1" ;;
        *)          TARGETS+=("$1"); shift ;;
    esac
done

if [[ ${#TARGETS[@]} -eq 0 ]]; then
    usage >&2
    exit 2
fi

# ── Build acl-exec argument list ──────────────────────────────────────────────
ACL_EXEC_ARGS=()

if (( APPLY )); then
    ACL_EXEC_ARGS+=("--apply")
else
    ACL_EXEC_ARGS+=("--dry-run")
fi

if (( VERBOSE ));   then ACL_EXEC_ARGS+=("--verbose");  fi
if (( NO_COLOR ));  then ACL_EXEC_ARGS+=("--no-color"); fi
if [[ -n "$RUNTIME" ]]; then ACL_EXEC_ARGS+=("--runtime" "$RUNTIME"); fi

ACL_EXEC_ARGS+=("${TARGETS[@]}")

# ── Delegate to acl-exec if available ────────────────────────────────────────
if command -v acl-exec >/dev/null 2>&1; then
    exec acl-exec "${ACL_EXEC_ARGS[@]}"
fi

# Look for acl-exec next to this script.
if [[ -x "$SCRIPT_DIR/../../cmd/acl-exec/acl-exec" ]]; then
    exec "$SCRIPT_DIR/../../cmd/acl-exec/acl-exec" "${ACL_EXEC_ARGS[@]}"
fi

# ── Fallback: acl-exec not found ──────────────────────────────────────────────
acl::warn "acl-exec binary not found on PATH."
acl::warn "Build it with: go build -o acl-exec ./acl/cmd/acl-exec/"
acl::info ""
acl::info "Dry-run plan for: ${TARGETS[*]}"
acl::info "  - would inspect PT_INTERP, RUNPATH/RPATH, and DT_NEEDED entries"
acl::info "  - no ELF bytes would be modified (dry-run)"
if (( APPLY )); then
    acl::warn "Apply mode requested but acl-exec is unavailable — no patches applied."
    exit 1
fi
exit 0
