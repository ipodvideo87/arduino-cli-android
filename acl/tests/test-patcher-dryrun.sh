#!/usr/bin/env bash
# test-patcher-dryrun.sh — integration tests for the acl-exec --dry-run feature.
#
# These tests verify that the patcher dry-run mode:
#   1. Prints a diff-style patch plan to stdout without modifying any ELF file.
#   2. Correctly identifies binaries that need patching vs those that are OK.
#   3. Shows the right field names (PT_INTERP, RUNPATH, RPATH) and values.
#   4. Skips GCC libexec and wrong-architecture binaries gracefully.
#   5. Exits with code 0 on success and 2 on usage errors.
#   6. Respects --verbose and --no-color flags.
#   7. Reads ACL_RUNTIME_DIR from the environment.
#
# Usage:
#   bash acl/tests/test-patcher-dryrun.sh [--keep-tmp]
#
# Requirements (resolved at runtime):
#   - go  in PATH (to build acl-exec and the ELF fixture generator)
#
# Flags:
#   --keep-tmp   Do not delete the temporary directory on exit (useful for
#                post-mortem debugging).
#
# Exit: 0 all tests pass, 1 one or more failures.
set -euo pipefail

# ── Colours ───────────────────────────────────────────────────────────────────
RED=$'\e[31m'
GREEN=$'\e[32m'
CYAN=$'\e[36m'
RESET=$'\e[0m'

# ── Argument parsing ──────────────────────────────────────────────────────────
KEEP_TMP=0
for arg in "$@"; do
    case "$arg" in
        --keep-tmp) KEEP_TMP=1 ;;
        *) echo "unknown argument: $arg" >&2; exit 2 ;;
    esac
done

# ── Locate the repo root ──────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# ── Temporary workspace ───────────────────────────────────────────────────────
TMPDIR_WORK="$(mktemp -d)"
if (( KEEP_TMP == 0 )); then
    trap 'rm -rf "$TMPDIR_WORK"' EXIT
fi
echo "${CYAN}work dir: $TMPDIR_WORK${RESET}"

ACL_EXEC_BIN="$TMPDIR_WORK/acl-exec"
FIXTURE_DIR="$TMPDIR_WORK/fixtures"
RUNTIME_DIR="$TMPDIR_WORK/runtime"
mkdir -p "$FIXTURE_DIR" "$RUNTIME_DIR"

# ── Build acl-exec ────────────────────────────────────────────────────────────
echo "${CYAN}── building acl-exec …${RESET}"
(cd "$REPO_ROOT" && go build -o "$ACL_EXEC_BIN" ./acl/cmd/acl-exec/)
echo "${GREEN}   built: $ACL_EXEC_BIN${RESET}"

# ── Generate ELF fixtures ─────────────────────────────────────────────────────
echo "${CYAN}── generating ELF fixtures …${RESET}"
(cd "$REPO_ROOT" && go run ./acl/tests/testdata/gen_test_elfs.go "$FIXTURE_DIR")
echo "${GREEN}   fixtures in: $FIXTURE_DIR${RESET}"

# ── Test helpers ──────────────────────────────────────────────────────────────
PASS=0
FAIL=0

pass() { echo "${GREEN}  PASS${RESET} $1"; PASS=$(( PASS + 1 )); }
fail() { echo "${RED}  FAIL${RESET} $1"; FAIL=$(( FAIL + 1 )); }

# assert_exit <desc> <expected-exit> <cmd...>
assert_exit() {
    local desc="$1" expected="$2"
    shift 2
    local actual=0
    "$@" >/dev/null 2>&1 || actual=$?
    if [[ "$actual" == "$expected" ]]; then
        pass "$desc (exit=$expected)"
    else
        fail "$desc — expected exit $expected, got $actual"
    fi
}

# assert_stdout_contains <desc> <needle> <cmd...>
assert_stdout_contains() {
    local desc="$1" needle="$2"
    shift 2
    local out
    out="$("$@" 2>/dev/null)" || true
    if printf '%s' "$out" | grep -qF -- "$needle"; then
        pass "$desc (contains: $needle)"
    else
        fail "$desc — output does not contain '$needle'"
        printf '    stdout (first 20 lines):\n'
        printf '%s\n' "$out" | head -20 | sed 's/^/      /'
    fi
}

# assert_stdout_not_contains <desc> <needle> <cmd...>
assert_stdout_not_contains() {
    local desc="$1" needle="$2"
    shift 2
    local out
    out="$("$@" 2>/dev/null)" || true
    if printf '%s' "$out" | grep -qF -- "$needle"; then
        fail "$desc — output unexpectedly contains '$needle'"
        printf '    stdout (first 20 lines):\n'
        printf '%s\n' "$out" | head -20 | sed 's/^/      /'
    else
        pass "$desc (absent: $needle)"
    fi
}

# assert_file_unchanged <desc> <path> <before-hash>
assert_file_unchanged() {
    local desc="$1" path="$2" before_hash="$3"
    local after_hash
    # Use sha256sum (Linux) or shasum -a 256 (macOS / Termux).
    if command -v sha256sum >/dev/null 2>&1; then
        after_hash="$(sha256sum "$path" | awk '{print $1}')"
    else
        after_hash="$(shasum -a 256 "$path" | awk '{print $1}')"
    fi
    if [[ "$before_hash" == "$after_hash" ]]; then
        pass "$desc (file unmodified)"
    else
        fail "$desc — file was MODIFIED during dry-run!"
    fi
}

# sha256 <path>  — portable SHA-256 hash helper
sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    else
        shasum -a 256 "$1" | awk '{print $1}'
    fi
}

# ── Fixture paths ─────────────────────────────────────────────────────────────
NEEDS_PATCH="$FIXTURE_DIR/needs-patch.elf"
ALREADY_OK="$FIXTURE_DIR/already-ok.elf"
GLIBC_RPATH="$FIXTURE_DIR/glibc-rpath.elf"
X86="$FIXTURE_DIR/x86_64.elf"
CC1="$FIXTURE_DIR/libexec/gcc/aarch64-linux-gnu/12/cc1"

# ── Tests ─────────────────────────────────────────────────────────────────────

echo ""
echo "${CYAN}── T01: usage error when no arguments given${RESET}"
assert_exit "no-args exits 2" 2 \
    "$ACL_EXEC_BIN"

echo ""
echo "${CYAN}── T02: --help exits 0 and mentions --dry-run, --apply, --runtime${RESET}"
assert_exit              "--help exits 0"           0 "$ACL_EXEC_BIN" --help
assert_stdout_contains   "--help has --dry-run"  "--dry-run"  "$ACL_EXEC_BIN" --help
assert_stdout_contains   "--help has --apply"    "--apply"    "$ACL_EXEC_BIN" --help
assert_stdout_contains   "--help has --runtime"  "--runtime"  "$ACL_EXEC_BIN" --help

echo ""
echo "${CYAN}── T03: dry-run on needs-patch.elf exits 0${RESET}"
assert_exit "dry-run exits 0" 0 \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --no-color "$NEEDS_PATCH"

echo ""
echo "${CYAN}── T04: dry-run shows PT_INTERP diff for needs-patch.elf${RESET}"
assert_stdout_contains "shows --- PT_INTERP" "--- PT_INTERP" \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --no-color "$NEEDS_PATCH"
assert_stdout_contains "shows +++ PT_INTERP" "+++ PT_INTERP" \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --no-color "$NEEDS_PATCH"
assert_stdout_contains "shows current interpreter" "/lib/ld-linux-aarch64.so.1" \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --no-color "$NEEDS_PATCH"
assert_stdout_contains "shows proposed loader path" "$RUNTIME_DIR/ld-linux-aarch64.so.1" \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --no-color "$NEEDS_PATCH"

echo ""
echo "${CYAN}── T05: dry-run shows RUNPATH diff for needs-patch.elf${RESET}"
assert_stdout_contains "shows RUNPATH edit" "RUNPATH" \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --no-color "$NEEDS_PATCH"
assert_stdout_contains "proposed RUNPATH = runtime dir" "$RUNTIME_DIR" \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --no-color "$NEEDS_PATCH"

echo ""
echo "${CYAN}── T06: dry-run summary line is present${RESET}"
assert_stdout_contains "summary line present" "Dry-run summary" \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --no-color "$NEEDS_PATCH"
assert_stdout_contains "summary shows 1 need patching" "1 need patching" \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --no-color "$NEEDS_PATCH"

echo ""
echo "${CYAN}── T07: dry-run does NOT modify the ELF file${RESET}"
BEFORE_HASH="$(sha256 "$NEEDS_PATCH")"
"$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --no-color "$NEEDS_PATCH" >/dev/null 2>&1 || true
assert_file_unchanged "needs-patch.elf unmodified" "$NEEDS_PATCH" "$BEFORE_HASH"

echo ""
echo "${CYAN}── T08: already-ok.elf not shown in quiet mode${RESET}"
assert_stdout_not_contains "quiet omits already-ok header" "=== $ALREADY_OK ===" \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --no-color "$ALREADY_OK"

echo ""
echo "${CYAN}── T09: already-ok.elf shown in --verbose mode${RESET}"
assert_stdout_contains "verbose shows already-ok header" "=== $ALREADY_OK ===" \
    "$ACL_EXEC_BIN" --dry-run --verbose --runtime "$RUNTIME_DIR" --no-color "$ALREADY_OK"
assert_stdout_contains "verbose shows [ok]" "[ok]" \
    "$ACL_EXEC_BIN" --dry-run --verbose --runtime "$RUNTIME_DIR" --no-color "$ALREADY_OK"

echo ""
echo "${CYAN}── T10: x86_64.elf is skipped in --verbose mode${RESET}"
assert_stdout_contains "x86 shows [skipped]" "[skipped]" \
    "$ACL_EXEC_BIN" --dry-run --verbose --runtime "$RUNTIME_DIR" --no-color "$X86"

echo ""
echo "${CYAN}── T11: GCC libexec cc1 skipped in --verbose mode${RESET}"
assert_stdout_contains "cc1 shows [skipped]" "[skipped]" \
    "$ACL_EXEC_BIN" --dry-run --verbose --runtime "$RUNTIME_DIR" --no-color "$CC1"

echo ""
echo "${CYAN}── T12: glibc-rpath.elf shows RPATH field in plan${RESET}"
assert_stdout_contains "RPATH field present" "RPATH" \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --no-color "$GLIBC_RPATH"

echo ""
echo "${CYAN}── T13: directory target expands to all ELF files${RESET}"
assert_exit "directory target exits 0" 0 \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --no-color "$FIXTURE_DIR"
assert_stdout_contains "directory scan shows summary" "Dry-run summary" \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --no-color "$FIXTURE_DIR"

echo ""
echo "${CYAN}── T14: --no-color suppresses ANSI escape sequences${RESET}"
out="$("$ACL_EXEC_BIN" --dry-run --no-color --verbose --runtime "$RUNTIME_DIR" "$FIXTURE_DIR" 2>/dev/null)" || true
# Use printf + od to detect ESC (0x1b) bytes without relying on grep -P
if printf '%s' "$out" | od -An -tx1 | tr ' ' '\n' | grep -q '^1b$'; then
    fail "--no-color: ESC (0x1b) byte found in output"
else
    pass "--no-color: no ESC bytes in output"
fi

echo ""
echo "${CYAN}── T15: ACL_RUNTIME_DIR environment variable is honoured${RESET}"
out="$(ACL_RUNTIME_DIR="$RUNTIME_DIR" \
    "$ACL_EXEC_BIN" --dry-run --no-color "$NEEDS_PATCH" 2>/dev/null)" || true
if printf '%s' "$out" | grep -qF "$RUNTIME_DIR"; then
    pass "ACL_RUNTIME_DIR reflected in plan output"
else
    fail "ACL_RUNTIME_DIR not reflected in plan output"
    printf '    stdout (first 10 lines):\n'
    printf '%s\n' "$out" | head -10 | sed 's/^/      /'
fi

echo ""
echo "${CYAN}── T16: missing file argument exits 2${RESET}"
assert_exit "missing file exits 2" 2 \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" /does/not/exist/binary

echo ""
echo "${CYAN}── T17: --loader flag overrides default loader basename${RESET}"
assert_stdout_contains "custom loader in proposed interp" "ld-custom.so.1" \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --loader ld-custom.so.1 --no-color "$NEEDS_PATCH"

echo ""
echo "${CYAN}── T18: multiple file arguments${RESET}"
assert_exit "multi-file exits 0" 0 \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --no-color \
    "$NEEDS_PATCH" "$ALREADY_OK" "$X86"

echo ""
echo "${CYAN}── T19: default mode (no --dry-run, no --apply) behaves like dry-run${RESET}"
assert_stdout_contains "default shows summary" "Dry-run summary" \
    "$ACL_EXEC_BIN" --runtime "$RUNTIME_DIR" --no-color "$NEEDS_PATCH"

echo ""
echo "${CYAN}── T20: reason text present for each planned edit${RESET}"
assert_stdout_contains "reason: line present" "reason:" \
    "$ACL_EXEC_BIN" --dry-run --runtime "$RUNTIME_DIR" --no-color "$NEEDS_PATCH"

# ── Final report ──────────────────────────────────────────────────────────────
echo ""
echo "─────────────────────────────────────────────────"
if (( FAIL == 0 )); then
    echo "${GREEN}ALL $PASS TEST(S) PASSED${RESET}"
    exit 0
else
    echo "${RED}$FAIL TEST(S) FAILED${RESET} (${GREEN}$PASS passed${RESET})"
    exit 1
fi
