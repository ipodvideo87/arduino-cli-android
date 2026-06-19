#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/runtime.sh"

runtime_dir="$(acl::runtime_investigation_dir)"
required_libs_text="$(acl::runtime_expected_files)"
required_libs=()
while IFS= read -r lib; do
	[[ -n "$lib" ]] || continue
	required_libs+=("$lib")
done <<<"$required_libs_text"

if [[ ! -d "$runtime_dir" ]]; then
	acl::die 1 "runtime directory is missing: $runtime_dir"
fi

missing=()
for lib in "${required_libs[@]}"; do
	match="$(find "$runtime_dir" \( -type f -o -type l \) -name "$lib" -print -quit)"
	if [[ -z "$match" ]]; then
		missing+=("$lib")
	fi
done

if (( ${#missing[@]} > 0 )); then
	acl::error "runtime layout check failed"
	acl::error "missing libraries: ${missing[*]}"
	exit 1
fi

acl::info "runtime layout looks present"
exit 0
