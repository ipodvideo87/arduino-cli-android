#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/elf.sh"

if [[ $# -ne 1 ]]; then
	acl::die 2 "Usage: verify-elf.sh <elf-file>"
fi

tmp_output="$(mktemp)"
trap 'rm -f "$tmp_output"' EXIT

if acl::run_scanner scan "$1" >"$tmp_output"; then
	acl::info "ELF verification passed for $1"
	cat "$tmp_output"
	exit 0
else
	status=$?
	acl::error "ELF verification failed for $1"
	cat "$tmp_output" >&2
	exit "$status"
fi
