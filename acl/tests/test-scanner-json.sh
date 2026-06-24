#!/usr/bin/env bash
# acl/tests/test-scanner-json.sh
#
# Golden-file / integration tests for `acl-scan --output json`.
#
# These tests run acl-scan against small synthetic fixtures (scripts, fake ELFs,
# fake Windows PE files) and validate:
#   1. Valid JSON is produced.
#   2. Required top-level fields are present (schema_version, generated_at,
#      binaries, summary).
#   3. Per-binary fields are present and have expected values.
#   4. Exit codes are correct.
#
# Dependencies: acl-scan (built from acl/cmd/acl-scan), jq, bash ≥ 4.
#
# Usage:
#   bash acl/tests/test-scanner-json.sh [--update-golden]
#
# With --update-golden, regenerates golden files from current acl-scan output
# instead of comparing against them.  Only use this when intentionally changing
# the report schema.
#
# Exit codes:
#   0  all tests passed
#   1  one or more tests failed
#   2  dependency missing (acl-scan, jq)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# ─── Colour helpers ───────────────────────────────────────────────────────────
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

pass() { printf "${GREEN}[PASS]${NC} %s\n" "$*"; }
fail() { printf "${RED}[FAIL]${NC} %s\n" "$*" >&2; FAILURES=$((FAILURES + 1)); }
info() { printf "${YELLOW}[INFO]${NC} %s\n" "$*"; }

FAILURES=0
UPDATE_GOLDEN=0

# ─── Argument parsing ─────────────────────────────────────────────────────────
for arg in "$@"; do
	case "$arg" in
		--update-golden) UPDATE_GOLDEN=1 ;;
		-h|--help)
			echo "Usage: $0 [--update-golden]"
			exit 0
			;;
		*)
			echo "Unknown argument: $arg" >&2
			exit 2
			;;
	esac
done

# ─── Dependency checks ────────────────────────────────────────────────────────
if ! command -v jq >/dev/null 2>&1; then
	echo "ERROR: jq is required but not installed." >&2
	exit 2
fi

# Locate acl-scan.  Try PATH first, then the built binary location.
ACL_SCAN=""
if command -v acl-scan >/dev/null 2>&1; then
	ACL_SCAN="$(command -v acl-scan)"
elif [[ -x "$REPO_ROOT/bin/acl-scan" ]]; then
	ACL_SCAN="$REPO_ROOT/bin/acl-scan"
elif [[ -x "$REPO_ROOT/acl/cmd/acl-scan/acl-scan" ]]; then
	ACL_SCAN="$REPO_ROOT/acl/cmd/acl-scan/acl-scan"
else
	info "acl-scan not found; attempting to build..."
	if command -v go >/dev/null 2>&1; then
		go build -o /tmp/acl-scan "$REPO_ROOT/acl/cmd/acl-scan/main.go" 2>/dev/null \
			|| go build -o /tmp/acl-scan ./acl/cmd/acl-scan/ 2>/dev/null \
			|| true
		if [[ -x /tmp/acl-scan ]]; then
			ACL_SCAN=/tmp/acl-scan
		fi
	fi
fi

if [[ -z "$ACL_SCAN" ]]; then
	echo "ERROR: acl-scan binary not found.  Build it first:" >&2
	echo "  go build -o bin/acl-scan ./acl/cmd/acl-scan/" >&2
	exit 2
fi

info "Using acl-scan: $ACL_SCAN"

# ─── Fixture setup ────────────────────────────────────────────────────────────
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

# Shell script fixture.
SCRIPT_FIXTURE="$TMPDIR/run.sh"
printf '#!/bin/sh\necho hello\n' > "$SCRIPT_FIXTURE"
chmod +x "$SCRIPT_FIXTURE"

# Windows PE fixture (MZ magic header only).
EXE_FIXTURE="$TMPDIR/tool.exe"
printf '\x4d\x5a\x00\x00' > "$EXE_FIXTURE"

# Unknown binary fixture (random bytes, no shebang, not ELF).
UNKNOWN_FIXTURE="$TMPDIR/unknown.bin"
printf '\x00\x01\x02\x03' > "$UNKNOWN_FIXTURE"

GOLDEN_DIR="$SCRIPT_DIR/testdata/golden"
mkdir -p "$GOLDEN_DIR"

# ─── Helper functions ─────────────────────────────────────────────────────────

# assert_json_field <json_file> <jq_filter> <expected_value> <test_name>
assert_json_field() {
	local json_file="$1"
	local filter="$2"
	local expected="$3"
	local name="$4"

	local actual
	actual="$(jq -r "$filter" "$json_file" 2>/dev/null || echo "__jq_error__")"
	if [[ "$actual" == "$expected" ]]; then
		pass "$name"
	else
		fail "$name: expected $(printf '%q' "$expected"), got $(printf '%q' "$actual")"
	fi
}

# assert_exit_code <expected> <actual> <test_name>
assert_exit_code() {
	local expected="$1"
	local actual="$2"
	local name="$3"
	if [[ "$expected" == "$actual" ]]; then
		pass "$name"
	else
		fail "$name: expected exit $expected, got $actual"
	fi
}

# run_scan_json <output_json_file> <args...>
# Runs acl-scan with --output json and writes JSON to output_json_file.
# Returns the exit code of acl-scan.
run_scan_json() {
	local out_file="$1"
	shift
	local exit_code=0
	"$ACL_SCAN" --output json "$@" > "$out_file" 2>/dev/null || exit_code=$?
	echo "$exit_code"
}

# compare_or_update_golden <golden_name> <actual_json_file>
compare_or_update_golden() {
	local name="$1"
	local actual="$2"
	local golden="$GOLDEN_DIR/${name}.json"

	# Strip generated_at so golden comparisons are deterministic.
	local normalised_actual
	normalised_actual="$(jq 'del(.generated_at)' "$actual")"

	if [[ "$UPDATE_GOLDEN" == "1" ]]; then
		echo "$normalised_actual" > "$golden"
		info "Golden updated: $golden"
		return
	fi

	if [[ ! -f "$golden" ]]; then
		fail "golden/$name.json missing — run with --update-golden to create it"
		return
	fi

	local normalised_golden
	normalised_golden="$(jq 'del(.generated_at)' "$golden")"

	if [[ "$normalised_actual" == "$normalised_golden" ]]; then
		pass "golden match: $name"
	else
		fail "golden mismatch: $name"
		diff <(echo "$normalised_golden") <(echo "$normalised_actual") >&2 || true
	fi
}

# ─── Tests ────────────────────────────────────────────────────────────────────

echo ""
echo "=== acl-scan JSON report tests ==="
echo ""

# ── T01: JSON output for a script file ───────────────────────────────────────
info "T01: script file → JSON"
T01_OUT="$TMPDIR/t01.json"
T01_CODE="$(run_scan_json "$T01_OUT" compat "$SCRIPT_FIXTURE")"

assert_exit_code 0 "$T01_CODE" "T01: exit code 0 for script"
assert_json_field "$T01_OUT" ".schema_version" "1.0"            "T01: schema_version"
assert_json_field "$T01_OUT" ".binaries | length" "1"           "T01: one binary"
assert_json_field "$T01_OUT" ".binaries[0].compat_category" "script" "T01: compat_category=script"
assert_json_field "$T01_OUT" ".binaries[0].recommendation.action" "script-no-elf-patch" "T01: action"
assert_json_field "$T01_OUT" ".summary.total" "1"               "T01: summary.total"
assert_json_field "$T01_OUT" ".summary.script" "1"              "T01: summary.script"
assert_json_field "$T01_OUT" ".summary.needs_patch" "0"         "T01: summary.needs_patch"
compare_or_update_golden "script-scan" "$T01_OUT"

# ── T02: JSON output for a Windows PE file ────────────────────────────────────
info "T02: Windows .exe → JSON"
T02_OUT="$TMPDIR/t02.json"
T02_CODE="$(run_scan_json "$T02_OUT" compat "$EXE_FIXTURE")"

# .exe is unsupported (known category); NeedsPatch=0, Unknown=0, Errors=0 → exit 0
assert_exit_code 0 "$T02_CODE" "T02: exit code 0 for unsupported (known category)"
assert_json_field "$T02_OUT" ".binaries[0].compat_category" "unsupported" "T02: compat_category=unsupported"
assert_json_field "$T02_OUT" ".binaries[0].recommendation.action" "unsupported" "T02: action=unsupported"
assert_json_field "$T02_OUT" ".summary.unsupported" "1" "T02: summary.unsupported"
compare_or_update_golden "exe-scan" "$T02_OUT"

# ── T03: compat-json sub-command always emits JSON ────────────────────────────
info "T03: compat-json sub-command"
T03_OUT="$TMPDIR/t03.json"
T03_CODE=0
"$ACL_SCAN" compat-json "$SCRIPT_FIXTURE" > "$T03_OUT" 2>/dev/null || T03_CODE=$?

assert_exit_code 0 "$T03_CODE" "T03: exit code"
# Must be valid JSON.
if jq . "$T03_OUT" >/dev/null 2>&1; then
	pass "T03: compat-json emits valid JSON"
else
	fail "T03: compat-json output is not valid JSON"
fi
assert_json_field "$T03_OUT" ".schema_version" "1.0" "T03: schema_version"

# ── T04: validate-compat-json always emits JSON ───────────────────────────────
info "T04: validate-compat-json sub-command"
T04_OUT="$TMPDIR/t04.json"
T04_CODE=0
"$ACL_SCAN" validate-compat-json "$SCRIPT_FIXTURE" > "$T04_OUT" 2>/dev/null || T04_CODE=$?

assert_exit_code 0 "$T04_CODE" "T04: exit code 0 for valid script"
assert_json_field "$T04_OUT" ".valid" "true"      "T04: valid=true"
assert_json_field "$T04_OUT" ".report.schema_version" "1.0" "T04: report.schema_version"

# ── T05: multi-file scan ──────────────────────────────────────────────────────
info "T05: multi-file scan"
T05_OUT="$TMPDIR/t05.json"
T05_CODE="$(run_scan_json "$T05_OUT" compat "$SCRIPT_FIXTURE" "$EXE_FIXTURE")"

assert_exit_code 0 "$T05_CODE" "T05: exit code 0"
assert_json_field "$T05_OUT" ".summary.total" "2"       "T05: total=2"
assert_json_field "$T05_OUT" ".summary.script" "1"      "T05: script=1"
assert_json_field "$T05_OUT" ".summary.unsupported" "1" "T05: unsupported=1"
compare_or_update_golden "multi-file-scan" "$T05_OUT"

# ── T06: required top-level fields present ────────────────────────────────────
info "T06: required top-level JSON fields"
T06_OUT="$TMPDIR/t06.json"
run_scan_json "$T06_OUT" compat "$SCRIPT_FIXTURE" > /dev/null 2>&1 || true

for field in schema_version generated_at binaries summary; do
	if jq -e "has(\"$field\")" "$T06_OUT" >/dev/null 2>&1; then
		pass "T06: field '$field' present"
	else
		fail "T06: field '$field' missing"
	fi
done

# ── T07: required per-binary fields present ───────────────────────────────────
info "T07: required per-binary JSON fields"
T07_OUT="$TMPDIR/t07.json"
run_scan_json "$T07_OUT" compat "$SCRIPT_FIXTURE" > /dev/null 2>&1 || true

for field in path compat_category recommendation; do
	if jq -e ".binaries[0] | has(\"$field\")" "$T07_OUT" >/dev/null 2>&1; then
		pass "T07: binary field '$field' present"
	else
		fail "T07: binary field '$field' missing"
	fi
done

for field in action rationale; do
	if jq -e ".binaries[0].recommendation | has(\"$field\")" "$T07_OUT" >/dev/null 2>&1; then
		pass "T07: recommendation.$field present"
	else
		fail "T07: recommendation.$field missing"
	fi
done

# ── T08: --output=json (equals form) ─────────────────────────────────────────
info "T08: --output=json (equals form)"
T08_OUT="$TMPDIR/t08.json"
T08_CODE=0
"$ACL_SCAN" --output=json compat "$SCRIPT_FIXTURE" > "$T08_OUT" 2>/dev/null || T08_CODE=$?
assert_exit_code 0 "$T08_CODE" "T08: exit code"
if jq . "$T08_OUT" >/dev/null 2>&1; then
	pass "T08: --output=json emits valid JSON"
else
	fail "T08: --output=json did not emit valid JSON"
fi

# ── T09: flag after sub-command ───────────────────────────────────────────────
info "T09: compat --output json (flag after sub-command)"
T09_OUT="$TMPDIR/t09.json"
T09_CODE=0
"$ACL_SCAN" compat --output json "$SCRIPT_FIXTURE" > "$T09_OUT" 2>/dev/null || T09_CODE=$?
assert_exit_code 0 "$T09_CODE" "T09: exit code"
if jq . "$T09_OUT" >/dev/null 2>&1; then
	pass "T09: flag-after-subcommand emits valid JSON"
else
	fail "T09: flag-after-subcommand did not emit valid JSON"
fi

# ── T10: unknown binary → unknown category ────────────────────────────────────
info "T10: unknown binary"
T10_OUT="$TMPDIR/t10.json"
T10_CODE="$(run_scan_json "$T10_OUT" compat "$UNKNOWN_FIXTURE")"
# NeedsPatch=0, Errors=0, Unknown=1 → exit 1.
assert_exit_code 1 "$T10_CODE" "T10: exit code 1 for unknown binary"
assert_json_field "$T10_OUT" ".binaries[0].compat_category" "unknown" "T10: compat_category=unknown"
assert_json_field "$T10_OUT" ".summary.unknown" "1" "T10: summary.unknown=1"

# ── T11: validate-compat exits 1 when unknown binary present ──────────────────
info "T11: validate-compat exits 1 for unknown binary"
T11_CODE=0
"$ACL_SCAN" --output json validate-compat "$UNKNOWN_FIXTURE" > /dev/null 2>/dev/null || T11_CODE=$?
assert_exit_code 1 "$T11_CODE" "T11: exit code 1 (unknown → invalid)"

# ── T12: no-args usage exits 2 ────────────────────────────────────────────────
info "T12: no-args → exit 2"
T12_CODE=0
"$ACL_SCAN" > /dev/null 2>&1 || T12_CODE=$?
assert_exit_code 2 "$T12_CODE" "T12: no-args exit 2"

# ─── Results ─────────────────────────────────────────────────────────────────
echo ""
if [[ "$FAILURES" -eq 0 ]]; then
	printf "${GREEN}All tests passed.${NC}\n"
	exit 0
else
	printf "${RED}%d test(s) failed.${NC}\n" "$FAILURES"
	exit 1
fi
