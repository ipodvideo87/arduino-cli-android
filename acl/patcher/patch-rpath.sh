#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

if ! command -v patchelf >/dev/null 2>&1; then
	acl::die 127 "patchelf is required for RPATH/RUNPATH patching; install patchelf and retry"
fi

acl::die 1 "patch-rpath.sh is not implemented yet in ACL v0.1"
