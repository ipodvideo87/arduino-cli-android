#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/common.sh"

acl::warn "launch verification is a safe placeholder in ACL v0.1"
acl::info "no launch checks are enforced yet"
exit 0
