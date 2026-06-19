#!/usr/bin/env bash

set -euo pipefail

REPO_URL="https://github.com/ipodvideo87/arduino-cli-android.git"
DEFAULT_BRANCH="android-runtime-v2"
VERIFY_BRANCH="${ACL_VERIFY_BRANCH:-$DEFAULT_BRANCH}"
VERIFY_ROOT="${HOME}/acl-verification"
CLONE_DIR="${VERIFY_ROOT}"

log() {
	printf '%s\n' "$*"
}

pass() {
	printf '[PASS] %s\n' "$*"
}

fail() {
	printf '[FAIL] %s\n' "$*" >&2
}

display_verify_root() {
	printf '%s\n' "~/acl-verification"
}

require_command() {
	local cmd=$1
	if ! command -v "${cmd}" >/dev/null 2>&1; then
		fail "Required command not found: ${cmd}"
		exit 1
	fi
}

run_step() {
	local description=$1
	shift
	log "==> ${description}"
	if "$@"; then
		pass "${description}"
	else
		fail "${description}"
		exit 1
	fi
}

run_in_clone() {
	(
		cd "${CLONE_DIR}"
		"$@"
	)
}

build_if_present() {
	local description=$1
	local source_dir=$2
	local output_path=$3
	local package_path=$4

	if [[ -d "${CLONE_DIR}/${source_dir}" ]]; then
		run_step "${description}" run_in_clone go build -o "${output_path}" "${package_path}"
	else
		log "==> ${description}"
		log "Skipping ${source_dir}; source directory not present in ${VERIFY_BRANCH}"
		pass "${description} (skipped)"
	fi
}

main() {
	require_command git
	require_command go
	require_command bash

	if [[ -e "${VERIFY_ROOT}" ]]; then
		log "An existing verification directory was found:"
		log
		display_verify_root
		log
		printf 'Delete it and continue? [y/N] '
		read -r answer
		case "${answer}" in
			y|Y|yes|YES)
				rm -rf "${VERIFY_ROOT}"
				;;
			*)
				fail "Verification canceled"
				exit 1
				;;
		esac
	fi

	log "Verification directory: ${VERIFY_ROOT}"
	log "Repository: ${REPO_URL}"
	log "Branch: ${VERIFY_BRANCH}"

	run_step "Clone repository" git clone "${REPO_URL}" "${CLONE_DIR}"
	run_step "Checkout branch ${VERIFY_BRANCH}" git -C "${CLONE_DIR}" checkout -B "${VERIFY_BRANCH}" "origin/${VERIFY_BRANCH}"

	run_step "Build arduino-cli from repository root" run_in_clone go build -o arduino-cli .
	build_if_present "Build acl-scan" "acl/cmd/acl-scan" "acl/acl-scan" "./acl/cmd/acl-scan"
	build_if_present "Build acl-exec" "acl/cmd/acl-exec" "acl/acl-exec" "./acl/cmd/acl-exec"
	build_if_present "Build acl-runtime" "acl/cmd/acl-runtime" "acl/acl-runtime" "./acl/cmd/acl-runtime"
	build_if_present "Build acl-build-runtime" "acl/cmd/acl-build-runtime" "acl/acl-build-runtime" "./acl/cmd/acl-build-runtime"

	run_step "Run internal ACL tests" run_in_clone go test ./internal/acl/...
	run_step "Run ACL command tests" run_in_clone go test ./acl/cmd/...
	run_step "Check for whitespace and patch formatting issues" run_in_clone git diff --check

	log
	log "========================================"
	log "ACL Fresh Clone Verification Complete"
	log "========================================"
	log
	log "Verification repository:"
	log
	display_verify_root
	log
	log "The repository has intentionally been left in place for inspection."
	log
	log "When finished inspecting it, remove it manually:"
	log
	log "rm -rf ~/acl-verification"
	log
	log "This check proves repository reproducibility only."
	log "It does not install Arduino cores, modify Arduino state, patch ESP32 tools, execute Linux ELF binaries through ACL, compile sketches, or validate real Android execution."
	pass "Fresh-clone verification completed for branch ${VERIFY_BRANCH}"
}

main "$@"
