#!/usr/bin/env bash

ACL_ELF_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$ACL_ELF_LIB_DIR/common.sh"

acl::elf_scan() {
	acl::run_scanner scan "$1"
}

acl::elf_deps() {
	acl::run_scanner deps "$1"
}

acl::elf_interpreter() {
	acl::run_scanner interpreter "$1"
}

acl::elf_symbols() {
	acl::run_scanner symbols "$1"
}

acl::elf_is_elf() {
	acl::run_scanner scan "$1" >/dev/null
}
