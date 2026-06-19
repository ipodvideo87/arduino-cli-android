#!/usr/bin/env bash

acl::log() {
	local level="$1"
	shift
	printf '%s: %s\n' "$level" "$*"
}

acl::info() {
	acl::log info "$@"
}

acl::warn() {
	acl::log warn "$@"
}

acl::error() {
	acl::log error "$@"
}

acl::die() {
	local code="${1:-1}"
	shift || true
	acl::error "$@"
	exit "$code"
}
