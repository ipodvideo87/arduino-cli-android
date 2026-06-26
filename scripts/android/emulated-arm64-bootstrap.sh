#!/bin/sh

set -u

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)

MODE=inspect
INSTALL_REQUESTED=${ANDROID_VALIDATION_INSTALL:-0}
REPORT_ROOT=${ANDROID_VALIDATION_REPORT_DIR:-"$REPO_ROOT/reports/android"}
RUN_ID=$(date -u +%Y%m%dT%H%M%SZ)
REPORT_DIR="$REPORT_ROOT/$RUN_ID/bootstrap"
HUMAN_REPORT="$REPORT_DIR/report.txt"
JSON_REPORT="$REPORT_DIR/report.json"

mkdir -p "$REPORT_DIR"
: >"$HUMAN_REPORT"

json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

json_array_from_pipe_list() {
  list=${1:-}
  [ -n "$list" ] || {
    printf '[]'
    return 0
  }
  old_ifs=$IFS
  IFS='|'
  set -f
  out=
  for item in $list; do
    [ -n "$item" ] || continue
    esc=$(json_escape "$item")
    if [ -n "$out" ]; then
      out="$out, "
    fi
    out="$out\"$esc\""
  done
  set +f
  IFS=$old_ifs
  printf '[%s]' "$out"
}

say() {
  printf '%s\n' "$1"
  printf '%s\n' "$1" >>"$HUMAN_REPORT"
}

join_by_pipe() {
  out=
  for item in "$@"; do
    if [ -n "$out" ]; then
      out="$out|$item"
    else
      out=$item
    fi
  done
  printf '%s' "$out"
}

have_cmd() {
  command -v "$1" >/dev/null 2>&1
}

cmd_version() {
  if have_cmd "$1"; then
    "$@" 2>&1 | head -n 1
  else
    printf 'missing'
  fi
}

mode_from_args() {
  while [ $# -gt 0 ]; do
    case $1 in
      inspect|bootstrap|verify)
        MODE=$1
        shift
        ;;
      --mode)
        MODE=${2:-}
        shift 2
        ;;
      --install)
        INSTALL_REQUESTED=1
        shift
        ;;
      --report-dir)
        REPORT_ROOT=${2:-"$REPORT_ROOT"}
        RUN_ID=$(date -u +%Y%m%dT%H%M%SZ)
        REPORT_DIR="$REPORT_ROOT/$RUN_ID/bootstrap"
        HUMAN_REPORT="$REPORT_DIR/report.txt"
        JSON_REPORT="$REPORT_DIR/report.json"
        mkdir -p "$REPORT_DIR"
        : >"$HUMAN_REPORT"
        shift 2
        ;;
      --help|-h)
        cat <<EOF
Usage: $0 [inspect|bootstrap|verify] [--install] [--report-dir DIR]

Modes:
  inspect    detect and report only
  bootstrap  install missing dependencies only when explicitly requested
  verify     confirm the environment without modifying it
EOF
        exit 0
        ;;
      *)
        printf '%s\n' "unknown argument: $1" >&2
        exit 2
        ;;
    esac
  done
}

mode_from_args "$@"

HOST_UNAME=$(uname -a)
HOST_ARCH=$(uname -m)
HOST_KERNEL=$(uname -r)
HOST_OS=$(uname -s)
HOST_PWD=$(pwd)
HOST_SHELL=${SHELL:-/bin/sh}
HOST_IN_PROOT=no
case $HOST_UNAME in
  *PRoot*|*proot*)
    HOST_IN_PROOT=yes
    ;;
esac

KVM_PRESENT=no
[ -e /dev/kvm ] && KVM_PRESENT=yes

GO_VERSION=missing
GIT_VERSION=missing
APT_VERSION=missing
DPKG_VERSION=missing
QEMU_AARCH64_PATH=
QEMU_AARCH64_STATIC_PATH=
SHELLCHECK_VERSION=missing
PYTHON3_VERSION=missing

have_cmd go && GO_VERSION=$(go version 2>&1 | head -n 1)
have_cmd git && GIT_VERSION=$(git --version 2>&1 | head -n 1)
have_cmd apt-get && APT_VERSION=$(apt-get --version 2>&1 | head -n 1)
have_cmd dpkg && DPKG_VERSION=$(dpkg --version 2>&1 | head -n 1)
have_cmd qemu-aarch64 && QEMU_AARCH64_PATH=$(command -v qemu-aarch64)
have_cmd qemu-aarch64-static && QEMU_AARCH64_STATIC_PATH=$(command -v qemu-aarch64-static)
have_cmd shellcheck && SHELLCHECK_VERSION=$(shellcheck --version 2>&1 | head -n 1)
have_cmd python3 && PYTHON3_VERSION=$(python3 --version 2>&1 | head -n 1)

CRITICAL_MISSING=
MISSING_PACKAGES=
INSTALLED_PACKAGES=
ACTION_TAKEN=
RECOMMENDATIONS=
WARNINGS=
LIMITATIONS=
FAILED=0

if [ -z "$GO_VERSION" ] || [ "$GO_VERSION" = missing ]; then
  CRITICAL_MISSING=$(join_by_pipe "$CRITICAL_MISSING" go)
fi
if [ -z "$GIT_VERSION" ] || [ "$GIT_VERSION" = missing ]; then
  CRITICAL_MISSING=$(join_by_pipe "$CRITICAL_MISSING" git)
fi

NEEDS_QEMU=no
case $HOST_ARCH in
  aarch64|arm64)
    NEEDS_QEMU=no
    ;;
  *)
    NEEDS_QEMU=yes
    ;;
esac

if [ "$NEEDS_QEMU" = yes ] && [ -z "$QEMU_AARCH64_PATH" ] && [ -z "$QEMU_AARCH64_STATIC_PATH" ]; then
  MISSING_PACKAGES=$(join_by_pipe "$MISSING_PACKAGES" qemu-user-static)
fi
if [ "$SHELLCHECK_VERSION" = missing ]; then
  MISSING_PACKAGES=$(join_by_pipe "$MISSING_PACKAGES" shellcheck)
fi

if [ "$MODE" = bootstrap ] && [ "$INSTALL_REQUESTED" = 1 ]; then
  if [ -z "${MISSING_PACKAGES:-}" ]; then
    ACTION_TAKEN=$(join_by_pipe "$ACTION_TAKEN" "no missing packages to install")
  elif [ "$APT_VERSION" = missing ] || [ "$DPKG_VERSION" = missing ]; then
    WARNINGS=$(join_by_pipe "$WARNINGS" "apt-get/dpkg are unavailable; cannot install missing packages")
  else
    if [ "$(id -u)" != 0 ] && ! have_cmd sudo; then
      WARNINGS=$(join_by_pipe "$WARNINGS" "not running as root and sudo is unavailable; cannot install missing packages")
    else
      if [ "$(id -u)" != 0 ]; then
        APT_PREFIX="sudo"
      else
        APT_PREFIX=""
      fi
      ACTION_TAKEN=$(join_by_pipe "$ACTION_TAKEN" "apt-get update")
      if [ -n "$APT_PREFIX" ]; then
        if ! $APT_PREFIX apt-get update; then
          WARNINGS=$(join_by_pipe "$WARNINGS" "apt-get update failed")
          FAILED=1
        fi
      elif ! apt-get update; then
        WARNINGS=$(join_by_pipe "$WARNINGS" "apt-get update failed")
        FAILED=1
      fi
      old_ifs=$IFS
      IFS='|'
      set -f
      set -- $MISSING_PACKAGES
      INSTALL_LIST=
      for pkg in "$@"; do
        if [ -n "$INSTALL_LIST" ]; then
          INSTALL_LIST="$INSTALL_LIST $pkg"
        else
          INSTALL_LIST=$pkg
        fi
      done
      set +f
      IFS=$old_ifs
      ACTION_TAKEN=$(join_by_pipe "$ACTION_TAKEN" "apt-get install -y $INSTALL_LIST")
      if [ -n "$APT_PREFIX" ]; then
        if ! $APT_PREFIX apt-get install -y $INSTALL_LIST; then
          WARNINGS=$(join_by_pipe "$WARNINGS" "apt-get install failed")
          FAILED=1
        fi
      elif ! apt-get install -y $INSTALL_LIST; then
        WARNINGS=$(join_by_pipe "$WARNINGS" "apt-get install failed")
        FAILED=1
      fi
      INSTALLED_PACKAGES=$INSTALL_LIST
    fi
  fi
elif [ "$MODE" = bootstrap ] && [ "$INSTALL_REQUESTED" != 1 ]; then
  ACTION_TAKEN=$(join_by_pipe "$ACTION_TAKEN" "bootstrap mode requested without --install; no changes made")
fi

if [ "$MODE" = verify ]; then
  if [ -n "${MISSING_PACKAGES:-}" ] || [ -n "${CRITICAL_MISSING:-}" ]; then
    WARNINGS=$(join_by_pipe "$WARNINGS" "verification failed because required capabilities are missing")
  fi
else
  if [ -n "${MISSING_PACKAGES:-}" ]; then
    RECOMMENDATIONS=$(join_by_pipe "$RECOMMENDATIONS" "run bootstrap --install to add missing optional dependencies")
  fi
  if [ -n "${CRITICAL_MISSING:-}" ]; then
    RECOMMENDATIONS=$(join_by_pipe "$RECOMMENDATIONS" "install or restore critical tools before running smoke tests")
  fi
fi

if [ "$HOST_IN_PROOT" = yes ]; then
  LIMITATIONS=$(join_by_pipe "$LIMITATIONS" "host runs inside proot-distro; this is development tooling only")
fi
if [ "$KVM_PRESENT" = no ]; then
  LIMITATIONS=$(join_by_pipe "$LIMITATIONS" "/dev/kvm is absent in this environment")
fi
if [ "$NEEDS_QEMU" = yes ] && [ -z "$QEMU_AARCH64_PATH" ] && [ -z "$QEMU_AARCH64_STATIC_PATH" ]; then
  LIMITATIONS=$(join_by_pipe "$LIMITATIONS" "no qemu-aarch64 binary available for ARM64 user-mode smoke execution")
fi

if [ -n "${WARNINGS:-}" ]; then
  :
fi

if [ "$MODE" = verify ] && { [ -n "${MISSING_PACKAGES:-}" ] || [ -n "${CRITICAL_MISSING:-}" ]; }; then
  OVERALL_STATUS=failed
  EXIT_CODE=1
elif [ "$FAILED" -eq 1 ]; then
  OVERALL_STATUS=failed
  EXIT_CODE=1
else
  OVERALL_STATUS=passed
  EXIT_CODE=0
fi

say "Validation provider: emulated ARM64 bootstrap"
say "Mode: $MODE"
say "Status: $OVERALL_STATUS"
say "Environment: $HOST_OS / $HOST_ARCH / $HOST_KERNEL"
say "Shell: $HOST_SHELL"
say "Working directory: $HOST_PWD"
say "Inside proot: $HOST_IN_PROOT"
say "KVM available: $KVM_PRESENT"
say "Go: $GO_VERSION"
say "Git: $GIT_VERSION"
say "apt-get: $APT_VERSION"
say "dpkg: $DPKG_VERSION"
say "qemu-aarch64: ${QEMU_AARCH64_PATH:-missing}"
say "qemu-aarch64-static: ${QEMU_AARCH64_STATIC_PATH:-missing}"
say "shellcheck: $SHELLCHECK_VERSION"
say "python3: $PYTHON3_VERSION"
say "Critical missing: ${CRITICAL_MISSING:-none}"
say "Missing capabilities: ${MISSING_PACKAGES:-none}"
say "Installed packages: ${INSTALLED_PACKAGES:-none}"
say "Actions taken: ${ACTION_TAKEN:-none}"
say "Warnings: ${WARNINGS:-none}"
say "Limitations: ${LIMITATIONS:-none}"
say "Recommendations: ${RECOMMENDATIONS:-none}"
say "Report files:"
say "  Human: $HUMAN_REPORT"
say "  JSON:  $JSON_REPORT"
say "Validation level: emulated preflight only"
say "Confidence: low to medium depending on available capabilities"

TOOLS_JSON=$(cat <<EOF
{
  "go": "$(json_escape "$GO_VERSION")",
  "git": "$(json_escape "$GIT_VERSION")",
  "apt_get": "$(json_escape "$APT_VERSION")",
  "dpkg": "$(json_escape "$DPKG_VERSION")",
  "qemu_aarch64": "$(json_escape "${QEMU_AARCH64_PATH:-missing}")",
  "qemu_aarch64_static": "$(json_escape "${QEMU_AARCH64_STATIC_PATH:-missing}")",
  "shellcheck": "$(json_escape "$SHELLCHECK_VERSION")",
  "python3": "$(json_escape "$PYTHON3_VERSION")"
}
EOF
)

cat >"$JSON_REPORT" <<EOF
{
  "schema": "github.com/arduino/arduino-cli/android-validation-report/v1",
  "provider": {
    "name": "emulated-arm64-bootstrap",
    "mode": "$(json_escape "$MODE")",
    "validation_level": "emulated-preflight-only"
  },
  "environment": {
    "os": "$(json_escape "$HOST_OS")",
    "architecture": "$(json_escape "$HOST_ARCH")",
    "kernel": "$(json_escape "$HOST_KERNEL")",
    "shell": "$(json_escape "$HOST_SHELL")",
    "working_directory": "$(json_escape "$HOST_PWD")",
    "inside_proot": $([ "$HOST_IN_PROOT" = yes ] && printf 'true' || printf 'false'),
    "kvm_available": $([ "$KVM_PRESENT" = yes ] && printf 'true' || printf 'false')
  },
  "tools": $TOOLS_JSON,
  "critical_missing": $(json_array_from_pipe_list "$CRITICAL_MISSING"),
  "installed_packages": $(json_array_from_pipe_list "$INSTALLED_PACKAGES"),
  "missing_packages": $(json_array_from_pipe_list "$MISSING_PACKAGES"),
  "actions_taken": $(json_array_from_pipe_list "$ACTION_TAKEN"),
  "warnings": $(json_array_from_pipe_list "$WARNINGS"),
  "limitations": $(json_array_from_pipe_list "$LIMITATIONS"),
  "recommendations": $(json_array_from_pipe_list "$RECOMMENDATIONS"),
  "results": {
    "overall_status": "$(json_escape "$OVERALL_STATUS")",
    "confidence": "low to medium",
    "notes": "This report is preflight-only and does not prove Android or Termux behavior."
  }
}
EOF

exit "$EXIT_CODE"
