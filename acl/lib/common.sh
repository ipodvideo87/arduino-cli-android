#!/usr/bin/env bash

ACL_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ACL_ROOT="$(cd "$ACL_LIB_DIR/.." && pwd)"

source "$ACL_LIB_DIR/logging.sh"

acl::repo_root() {
	printf '%s\n' "$ACL_ROOT"
}

acl::runtime_dir() {
	printf '%s\n' "$ACL_ROOT/runtime"
}

acl::scanner_source_dir() {
	printf '%s\n' "$ACL_ROOT/cmd/acl-scan"
}

acl::scanner_command() {
	if [[ -x "$ACL_ROOT/acl-scan" ]]; then
		ACL_SCANNER_CMD=("$ACL_ROOT/acl-scan")
		return 0
	fi

	if [[ -x "$ACL_ROOT/cmd/acl-scan/acl-scan" ]]; then
		ACL_SCANNER_CMD=("$ACL_ROOT/cmd/acl-scan/acl-scan")
		return 0
	fi

	if command -v go >/dev/null 2>&1; then
		ACL_SCANNER_CMD=(go run "$ACL_ROOT/cmd/acl-scan")
		return 0
	fi

	return 1
}

acl::run_scanner() {
	acl::scanner_command || acl::die 127 "acl-scan is unavailable; build acl/cmd/acl-scan or install Go"
	"${ACL_SCANNER_CMD[@]}" "$@"
}
