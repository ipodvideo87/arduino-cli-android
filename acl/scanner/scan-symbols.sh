#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

if [[ $# -gt 1 ]]; then
	acl::die 2 "Usage: scan-symbols.sh [elf-file]"
fi

acl::warn "symbol scanning not implemented yet"
exit 0
