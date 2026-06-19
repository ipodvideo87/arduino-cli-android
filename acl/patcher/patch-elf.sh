#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

apply_patch_mode=0

usage() {
	cat <<'EOF'
Usage: patch-elf.sh [--apply] <elf-file>

Default behavior prints a patch plan only.
EOF
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		--apply)
			apply_patch_mode=1
			shift
			;;
		-h|--help)
			usage
			exit 0
			;;
		--)
			shift
			break
			;;
		-*)
			acl::die 2 "unknown option: $1"
			;;
		*)
			break
			;;
	esac
done

if [[ $# -ne 1 ]]; then
	usage >&2
	exit 2
fi

target="$1"

acl::info "patch plan for $target"
acl::info " - inspect PT_INTERP, RPATH, RUNPATH and DT_NEEDED entries"
acl::info " - no ELF bytes are modified by default"

if (( apply_patch_mode == 0 )); then
	exit 0
fi

acl::warn "ACL v0.1 does not modify ELF files yet"
exit 0
