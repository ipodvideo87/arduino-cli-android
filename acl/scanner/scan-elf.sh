#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/elf.sh"

if [[ $# -ne 1 ]]; then
	acl::die 2 "Usage: scan-elf.sh <elf-file>"
fi

acl::elf_scan "$1"
