#!/usr/bin/env bash
# test-scanner-json.sh — ACL scanner JSON output smoke tests
#
# Exercises the acl-scan compat-json subcommand against fixture files and
# checks that the emitted JSON contains the fields mandated by the report
# schema (including the new interpreter_status field for scripts).
#
# Usage:
#   ./test-scanner-json.sh [--prefix <termux-prefix>] [--acl-bin <path-to-acl-scan>]
#
# Defaults:
#   --prefix  : $PREFIX if set, otherwise skips live interpreter resolution
#   --acl-bin : acl-scan on PATH, or build/acl-scan if present
#
# Exit codes:
#   0  all assertions passed
#   1  one or more assertions failed

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FIXTURE_SCRIPTS_DIR="$SCRIPT_DIR/testdata/scripts"

# ── argument parsing ──────────────────────────────────────────────────────────
PREFIX_DIR="${PREFIX:-}"
ACL_BIN=""

while [[ $# -gt 0 ]]; do
	case "$1" in
	--prefix)
		PREFIX_DIR="$2"
		shift 2
		;;
	--acl-bin)
		ACL_BIN="$2"
		shift 2
		;;
	*)
		echo "unknown argument: $1" >&2
		exit 2
		;;
	esac
done

# Locate acl-scan binary.
if [[ -z "$ACL_BIN" ]]; then
	if command -v acl-scan >/dev/null 2>&1; then
		ACL_BIN="acl-scan"
	elif [[ -x "$SCRIPT_DIR/../build/acl-scan" ]]; then
		ACL_BIN="$SCRIPT_DIR/../build/acl-scan"
	fi
fi

# ── helpers ───────────────────────────────────────────────────────────────────
PASS=0
FAIL=0

pass() { echo "  PASS: $*"; ((PASS++)) || true; }
fail() { echo "  FAIL: $*" >&2; ((FAIL++)) || true; }

assert_jq() {
	local label="$1"
	local json="$2"
	local query="$3"
	local expected="$4"
	local got
	got="$(echo "$json" | jq -r "$query" 2>/dev/null || true)"
	if [[ "$got" == "$expected" ]]; then
		pass "$label"
	else
		fail "$label — query: $query — expected: $expected — got: $got"
	fi
}

assert_contains() {
	local label="$1"
	local haystack="$2"
	local needle="$3"
	if echo "$haystack" | grep -qF "$needle"; then
		pass "$label"
	else
		fail "$label — expected to find: $needle"
	fi
}

# ── require jq ───────────────────────────────────────────────────────────────
if ! command -v jq >/dev/null 2>&1; then
	echo "SKIP: jq not found; install jq to run JSON assertions" >&2
	exit 0
fi

# ── build a minimal fake prefix for interpreter resolution tests ──────────────
FAKE_PREFIX="$(mktemp -d)"
trap 'rm -rf "$FAKE_PREFIX"' EXIT
mkdir -p "$FAKE_PREFIX/bin"
for interp in bash python3 perl env sh; do
	echo '#!/bin/sh' >"$FAKE_PREFIX/bin/$interp"
	chmod +x "$FAKE_PREFIX/bin/$interp"
done

echo "=== ACL Scanner JSON Tests ==="
echo ""

# ── Go unit tests (always available) ─────────────────────────────────────────
echo "--- Go unit tests ---"
if command -v go >/dev/null 2>&1; then
	MODULE_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
	if go test -count=1 ./acl/scanner/... -run TestParseShebang -v 2>&1 | grep -q "PASS"; then
		pass "ParseShebang unit tests (go test)"
	else
		# Re-run without -q to show errors.
		go test -count=1 ./acl/scanner/... -run TestParseShebang -v 2>&1 || true
		fail "ParseShebang unit tests"
	fi

	if go test -count=1 ./acl/scanner/... -run TestCheckInterpreter -v 2>&1 | grep -q "PASS"; then
		pass "CheckInterpreter unit tests (go test)"
	else
		go test -count=1 ./acl/scanner/... -run TestCheckInterpreter -v 2>&1 || true
		fail "CheckInterpreter unit tests"
	fi

	if go test -count=1 ./acl/scanner/... -run TestScanShebang -v 2>&1 | grep -q "PASS"; then
		pass "ScanShebang unit tests (go test)"
	else
		go test -count=1 ./acl/scanner/... -run TestScanShebang -v 2>&1 || true
		fail "ScanShebang unit tests"
	fi

	if go test -count=1 ./acl/scanner/... -run TestReportBuilder -v 2>&1 | grep -q "PASS"; then
		pass "ReportBuilder unit tests (go test)"
	else
		go test -count=1 ./acl/scanner/... -run TestReportBuilder -v 2>&1 || true
		fail "ReportBuilder unit tests"
	fi
else
	echo "  (go not found — skipping Go unit tests)"
fi

echo ""

# ── fixture file assertions via jq on directly-built JSON ─────────────────────
# These tests use a small inline Go program to invoke the scanner library
# directly if acl-scan binary is not available.

echo "--- Fixture shebang content checks ---"

# Check fixture files have correct shebang lines.
check_shebang() {
	local file="$FIXTURE_SCRIPTS_DIR/$1"
	local expected_shebang="$2"
	local label="$3"
	if [[ ! -f "$file" ]]; then
		fail "$label — fixture not found: $file"
		return
	fi
	local first_line
	first_line="$(head -1 "$file")"
	if [[ "$first_line" == "$expected_shebang" ]]; then
		pass "$label"
	else
		fail "$label — expected: $expected_shebang — got: $first_line"
	fi
}

check_shebang "bash_script.sh"        "#!/bin/bash"              "fixture bash_script.sh has /bin/bash shebang"
check_shebang "usr_bin_bash_script.sh" "#!/usr/bin/bash"          "fixture usr_bin_bash_script.sh has /usr/bin/bash shebang"
check_shebang "env_bash_script.sh"    "#!/usr/bin/env bash"      "fixture env_bash_script.sh has /usr/bin/env bash shebang"
check_shebang "python3_script.py"     "#!/usr/bin/python3"       "fixture python3_script.py has /usr/bin/python3 shebang"
check_shebang "env_python3_script.py" "#!/usr/bin/env python3"   "fixture env_python3_script.py has /usr/bin/env python3 shebang"
check_shebang "perl_script.pl"        "#!/usr/bin/perl"          "fixture perl_script.pl has /usr/bin/perl shebang"
check_shebang "env_perl_script.pl"    "#!/usr/bin/env perl"      "fixture env_perl_script.pl has /usr/bin/env perl shebang"
check_shebang "env_unknown_script.sh" "#!/usr/bin/env notarealinterpreter" "fixture env_unknown_script.sh has unknown env shebang"

# Confirm no_shebang_script.sh does NOT start with #!
if [[ -f "$FIXTURE_SCRIPTS_DIR/no_shebang_script.sh" ]]; then
	first="$(head -1 "$FIXTURE_SCRIPTS_DIR/no_shebang_script.sh")"
	if [[ "$first" != \#\!* ]]; then
		pass "fixture no_shebang_script.sh has no shebang"
	else
		fail "fixture no_shebang_script.sh unexpectedly has shebang: $first"
	fi
else
	fail "fixture no_shebang_script.sh not found"
fi

echo ""

# ── acl-scan binary tests (optional) ─────────────────────────────────────────
if [[ -z "$ACL_BIN" ]]; then
	echo "--- acl-scan binary not found; skipping binary-level JSON tests ---"
	echo "    Build with: go build -o build/acl-scan ./acl/cmd/acl-scan/"
	echo ""
else
	echo "--- acl-scan binary JSON output tests (bin: $ACL_BIN) ---"

	PREFIX_ARGS=()
	if [[ -n "$PREFIX_DIR" ]]; then
		PREFIX_ARGS=(--prefix "$PREFIX_DIR")
	else
		PREFIX_ARGS=(--prefix "$FAKE_PREFIX")
	fi

	# Test 1: bash script → interpreter_status present, status=remapped
	BASH_SCRIPT="$FIXTURE_SCRIPTS_DIR/bash_script.sh"
	if [[ -f "$BASH_SCRIPT" ]]; then
		JSON="$("$ACL_BIN" compat-json "${PREFIX_ARGS[@]}" "$BASH_SCRIPT" 2>/dev/null || true)"
		if [[ -n "$JSON" ]]; then
			assert_jq "bash_script: schema_version=1.0"           "$JSON" '.schema_version'          "1.0"
			assert_jq "bash_script: entry category=script"        "$JSON" '.entries[0].category'     "script"
			assert_jq "bash_script: interpreter_status present"   "$JSON" '.entries[0].interpreter_status | type' "object"
			assert_jq "bash_script: declared_path=/bin/bash"      "$JSON" '.entries[0].interpreter_status.declared_path' "/bin/bash"
			assert_jq "bash_script: status=remapped"              "$JSON" '.entries[0].interpreter_status.status' "remapped"
		else
			fail "bash_script: acl-scan returned no output"
		fi
	fi

	# Test 2: env python3 script
	PY_SCRIPT="$FIXTURE_SCRIPTS_DIR/env_python3_script.py"
	if [[ -f "$PY_SCRIPT" ]]; then
		JSON="$("$ACL_BIN" compat-json "${PREFIX_ARGS[@]}" "$PY_SCRIPT" 2>/dev/null || true)"
		if [[ -n "$JSON" ]]; then
			assert_jq "env_python3: interpreter=/usr/bin/env"    "$JSON" '.entries[0].interpreter_status.declared_path' "/usr/bin/env"
			assert_jq "env_python3: args[0]=python3"            "$JSON" '.entries[0].interpreter_status.args[0]'        "python3"
			assert_jq "env_python3: status=remapped"            "$JSON" '.entries[0].interpreter_status.status'         "remapped"
		else
			fail "env_python3: acl-scan returned no output"
		fi
	fi

	# Test 3: perl script
	PERL_SCRIPT="$FIXTURE_SCRIPTS_DIR/perl_script.pl"
	if [[ -f "$PERL_SCRIPT" ]]; then
		JSON="$("$ACL_BIN" compat-json "${PREFIX_ARGS[@]}" "$PERL_SCRIPT" 2>/dev/null || true)"
		if [[ -n "$JSON" ]]; then
			assert_jq "perl_script: declared_path=/usr/bin/perl" "$JSON" '.entries[0].interpreter_status.declared_path' "/usr/bin/perl"
			assert_jq "perl_script: status=remapped"             "$JSON" '.entries[0].interpreter_status.status'         "remapped"
		else
			fail "perl_script: acl-scan returned no output"
		fi
	fi

	# Test 4: no shebang → interpreter_status absent (null)
	NO_SHEBANG="$FIXTURE_SCRIPTS_DIR/no_shebang_script.sh"
	if [[ -f "$NO_SHEBANG" ]]; then
		JSON="$("$ACL_BIN" compat-json "${PREFIX_ARGS[@]}" "$NO_SHEBANG" 2>/dev/null || true)"
		if [[ -n "$JSON" ]]; then
			assert_jq "no_shebang: interpreter_status=null" "$JSON" '.entries[0].interpreter_status' "null"
		else
			fail "no_shebang: acl-scan returned no output"
		fi
	fi

	# Test 5: unknown env delegate → missing
	UNK_SCRIPT="$FIXTURE_SCRIPTS_DIR/env_unknown_script.sh"
	if [[ -f "$UNK_SCRIPT" ]]; then
		JSON="$("$ACL_BIN" compat-json "${PREFIX_ARGS[@]}" "$UNK_SCRIPT" 2>/dev/null || true)"
		if [[ -n "$JSON" ]]; then
			assert_jq "env_unknown: status=missing" "$JSON" '.entries[0].interpreter_status.status' "missing"
		else
			fail "env_unknown: acl-scan returned no output"
		fi
	fi

	echo ""
fi

# ── summary ───────────────────────────────────────────────────────────────────
echo "Results: $PASS passed, $FAIL failed"
echo ""
if [[ $FAIL -gt 0 ]]; then
	exit 1
fi
exit 0
