#!/usr/bin/env bash

ACL_RUNTIME_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$ACL_RUNTIME_LIB_DIR/common.sh"
ACL_REPO_ROOT="$(cd "$ACL_RUNTIME_LIB_DIR/../.." && pwd)"

acl::runtime_investigation_dir() {
	if [[ -n "${ACL_RUNTIME_DIR:-}" && -d "${ACL_RUNTIME_DIR}" ]]; then
		printf '%s\n' "$ACL_RUNTIME_DIR"
		return 0
	fi

	if [[ -d "$ACL_REPO_ROOT/internal/android/runtime" ]]; then
		printf '%s\n' "$ACL_REPO_ROOT/internal/android/runtime"
		return 0
	fi

	printf '%s\n' "$ACL_REPO_ROOT/acl/runtime"
}

acl::runtime_expected_files() {
	cat <<'EOF'
ld-linux-aarch64.so.1
libc.so.6
libdl.so.2
libm.so.6
libpthread.so.0
librt.so.1
libz.so.1
EOF
}

acl::runtime_absolute_path_markers() {
	cat <<'EOF'
/data/data/com.termux/files/usr/glibc
/data/data/com.termux/files/usr
/system/bin
/vendor
EOF
}
