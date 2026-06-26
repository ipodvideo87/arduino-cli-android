#!/bin/sh

set -u

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)
BOOTSTRAP_SCRIPT="$SCRIPT_DIR/emulated-arm64-bootstrap.sh"

REPORT_ROOT=${ANDROID_VALIDATION_REPORT_DIR:-"$REPO_ROOT/reports/android"}
RUN_ID=$(date -u +%Y%m%dT%H%M%SZ)
REPORT_DIR="$REPORT_ROOT/$RUN_ID/smoke-test"
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

SKETCH_CANDIDATES=${ANDROID_SMOKE_SKETCHES:-"$HOME/Development/Sketches/Blink:$HOME/Development/Sketches/esp32ultimate"}
FQBN_DEFAULT=${ANDROID_SMOKE_FQBN:-"esp32:esp32:esp32s3"}
PACKAGE_TAG=$(printf '%s' "$FQBN_DEFAULT" | tr ':' '.')

HOST_OS=$(uname -s)
HOST_ARCH=$(uname -m)
HOST_KERNEL=$(uname -r)
HOST_PWD=$(pwd)
GO_VERSION=missing
have_cmd go && GO_VERSION=$(go version 2>&1 | head -n 1)

TESTS_EXECUTED=
TESTS_SKIPPED=
RESULTS=
WARNINGS=
LIMITATIONS=
RECOMMENDATIONS=
ACTION_TAKEN=
FAILED=0
BUILD_OK=0

ACTION_TAKEN=$(join_by_pipe "$ACTION_TAKEN" "sh $BOOTSTRAP_SCRIPT inspect")
sh "$BOOTSTRAP_SCRIPT" inspect
TESTS_EXECUTED=$(join_by_pipe "$TESTS_EXECUTED" "$BOOTSTRAP_SCRIPT inspect")

if [ ! -f "$REPO_ROOT/go.mod" ]; then
  printf '%s\n' "not in repository root" >&2
  exit 2
fi

if have_cmd go; then
  ACTION_TAKEN=$(join_by_pipe "$ACTION_TAKEN" "go build -o arduino-cli .")
  if (cd "$REPO_ROOT" && go build -o arduino-cli .); then
    BUILD_OK=1
    TESTS_EXECUTED=$(join_by_pipe "$TESTS_EXECUTED" "go build -o arduino-cli .")
  else
    FAILED=1
    WARNINGS=$(join_by_pipe "$WARNINGS" "go build -o arduino-cli . failed")
  fi
else
  TESTS_SKIPPED=$(join_by_pipe "$TESTS_SKIPPED" "go build -o arduino-cli . (go unavailable)")
fi

if have_cmd go; then
  ACTION_TAKEN=$(join_by_pipe "$ACTION_TAKEN" "go test ./internal/acl/... ./internal/cli/...")
  if (cd "$REPO_ROOT" && go test ./internal/acl/... ./internal/cli/...); then
    TESTS_EXECUTED=$(join_by_pipe "$TESTS_EXECUTED" "go test ./internal/acl/... ./internal/cli/...")
  else
    FAILED=1
    WARNINGS=$(join_by_pipe "$WARNINGS" "go test ./internal/acl/... ./internal/cli/... failed")
  fi
else
  TESTS_SKIPPED=$(join_by_pipe "$TESTS_SKIPPED" "go test ./internal/acl/... ./internal/cli/... (go unavailable)")
fi

if [ "$BUILD_OK" = 1 ] && [ -x "$REPO_ROOT/arduino-cli" ]; then
  ACTION_TAKEN=$(join_by_pipe "$ACTION_TAKEN" "./arduino-cli acl --help")
  if (cd "$REPO_ROOT" && ./arduino-cli acl --help >/dev/null); then
    TESTS_EXECUTED=$(join_by_pipe "$TESTS_EXECUTED" "./arduino-cli acl --help")
  else
    FAILED=1
    WARNINGS=$(join_by_pipe "$WARNINGS" "./arduino-cli acl --help failed")
  fi

  ACTION_TAKEN=$(join_by_pipe "$ACTION_TAKEN" "./arduino-cli acl workflow --help")
  if (cd "$REPO_ROOT" && ./arduino-cli acl workflow --help >/dev/null); then
    TESTS_EXECUTED=$(join_by_pipe "$TESTS_EXECUTED" "./arduino-cli acl workflow --help")
  else
    FAILED=1
    WARNINGS=$(join_by_pipe "$WARNINGS" "./arduino-cli acl workflow --help failed")
  fi

  ACTION_TAKEN=$(join_by_pipe "$ACTION_TAKEN" "./arduino-cli acl bootstrap --details")
  if (cd "$REPO_ROOT" && ./arduino-cli acl bootstrap --details >/dev/null); then
    TESTS_EXECUTED=$(join_by_pipe "$TESTS_EXECUTED" "./arduino-cli acl bootstrap --details")
  else
    FAILED=1
    WARNINGS=$(join_by_pipe "$WARNINGS" "./arduino-cli acl bootstrap --details failed")
  fi

  ACTION_TAKEN=$(join_by_pipe "$ACTION_TAKEN" "./arduino-cli acl workflow bootstrap --details")
  if (cd "$REPO_ROOT" && ./arduino-cli acl workflow bootstrap --details >/dev/null); then
    TESTS_EXECUTED=$(join_by_pipe "$TESTS_EXECUTED" "./arduino-cli acl workflow bootstrap --details")
  else
    FAILED=1
    WARNINGS=$(join_by_pipe "$WARNINGS" "./arduino-cli acl workflow bootstrap --details failed")
  fi

  ACTION_TAKEN=$(join_by_pipe "$ACTION_TAKEN" "./arduino-cli acl workflow diagnostics --details")
  if (cd "$REPO_ROOT" && ./arduino-cli acl workflow diagnostics --details >/dev/null); then
    TESTS_EXECUTED=$(join_by_pipe "$TESTS_EXECUTED" "./arduino-cli acl workflow diagnostics --details")
  else
    FAILED=1
    WARNINGS=$(join_by_pipe "$WARNINGS" "./arduino-cli acl workflow diagnostics --details failed")
  fi
else
  TESTS_SKIPPED=$(join_by_pipe "$TESTS_SKIPPED" "./arduino-cli ACL checks (binary unavailable)")
fi

FOUND_SKETCH=
for sketch in $(printf '%s\n' "$SKETCH_CANDIDATES" | tr ':' ' '); do
  if [ -d "$sketch" ]; then
    FOUND_SKETCH=$sketch
    break
  fi
done

FOUND_CORE=no
if [ -d "$HOME/.arduino15/packages/esp32" ] || [ -d "$HOME/.arduino15/packages/arduino/hardware/esp32" ]; then
  FOUND_CORE=yes
fi

if [ -n "$FOUND_SKETCH" ] && [ "$FOUND_CORE" = yes ] && [ "$BUILD_OK" = 1 ] && [ -x "$REPO_ROOT/arduino-cli" ]; then
  ACTION_TAKEN=$(join_by_pipe "$ACTION_TAKEN" "./arduino-cli acl workflow compile --fqbn $FQBN_DEFAULT $FOUND_SKETCH")
  if (cd "$REPO_ROOT" && ./arduino-cli acl workflow compile --fqbn "$FQBN_DEFAULT" "$FOUND_SKETCH" >/dev/null); then
    TESTS_EXECUTED=$(join_by_pipe "$TESTS_EXECUTED" "./arduino-cli acl workflow compile --fqbn $FQBN_DEFAULT $FOUND_SKETCH")

    PACKAGE_DIR="$FOUND_SKETCH/build/$PACKAGE_TAG/firmware-package"
    if [ -d "$PACKAGE_DIR" ]; then
      for path in \
        manifest.json \
        flash-plan.json \
        validation-report.json \
        analysis.json \
        README_FLASHING.txt \
        artifacts/application.bin \
        artifacts/firmware.elf \
        artifacts/firmware.map
      do
        if [ -e "$PACKAGE_DIR/$path" ]; then
          :
        else
          WARNINGS=$(join_by_pipe "$WARNINGS" "missing expected package file: $path")
          FAILED=1
        fi
      done
      if [ -e "$PACKAGE_DIR/artifacts/bootloader.bin" ]; then
        RESULTS=$(join_by_pipe "$RESULTS" "full-flash package includes bootloader.bin")
      else
        RESULTS=$(join_by_pipe "$RESULTS" "package is app-only or missing bootloader.bin")
      fi
    else
      WARNINGS=$(join_by_pipe "$WARNINGS" "compile completed but firmware-package directory was not found")
      FAILED=1
    fi
  else
    WARNINGS=$(join_by_pipe "$WARNINGS" "./arduino-cli acl workflow compile failed")
    FAILED=1
  fi
else
  TESTS_SKIPPED=$(join_by_pipe "$TESTS_SKIPPED" "acl workflow compile (sketch or core unavailable)")
  RECOMMENDATIONS=$(join_by_pipe "$RECOMMENDATIONS" "set ANDROID_SMOKE_SKETCHES to one or more sketches and ensure the ESP32 core is installed to exercise compile packaging")
fi

LIMITATIONS=$(join_by_pipe "$LIMITATIONS" "emulated preflight only; not native Termux validation")
LIMITATIONS=$(join_by_pipe "$LIMITATIONS" "not Android emulator validation unless separately proven")
LIMITATIONS=$(join_by_pipe "$LIMITATIONS" "not real hardware validation")

if [ -z "${RESULTS:-}" ]; then
  RESULTS=$(join_by_pipe "$RESULTS" "host smoke checks completed")
fi

say "Validation provider: emulated ARM64 smoke test"
say "Environment: $HOST_OS / $HOST_ARCH / $HOST_KERNEL"
say "Go: $GO_VERSION"
say "Repository: $REPO_ROOT"
say "Tests executed: ${TESTS_EXECUTED:-none}"
say "Tests skipped: ${TESTS_SKIPPED:-none}"
say "Results: ${RESULTS:-none}"
say "Warnings: ${WARNINGS:-none}"
say "Limitations: ${LIMITATIONS:-none}"
say "Recommendations: ${RECOMMENDATIONS:-none}"
say "Report files:"
say "  Human: $HUMAN_REPORT"
say "  JSON:  $JSON_REPORT"
say "Validation level: emulated preflight only"
say "Labels: Not native Termux validation; not Android emulator validation unless proven; not real hardware validation"

cat >"$JSON_REPORT" <<EOF
{
  "schema": "github.com/arduino/arduino-cli/android-validation-report/v1",
  "provider": {
    "name": "emulated-arm64-smoke-test",
    "mode": "inspect",
    "validation_level": "emulated-preflight-only"
  },
  "environment": {
    "os": "$(json_escape "$HOST_OS")",
    "architecture": "$(json_escape "$HOST_ARCH")",
    "kernel": "$(json_escape "$HOST_KERNEL")",
    "working_directory": "$(json_escape "$HOST_PWD")",
    "repository_root": "$(json_escape "$REPO_ROOT")"
  },
  "tools": {
    "go": "$(json_escape "$GO_VERSION")"
  },
  "tests_executed": $(json_array_from_pipe_list "$TESTS_EXECUTED"),
  "tests_skipped": $(json_array_from_pipe_list "$TESTS_SKIPPED"),
  "results": $(json_array_from_pipe_list "$RESULTS"),
  "warnings": $(json_array_from_pipe_list "$WARNINGS"),
  "limitations": $(json_array_from_pipe_list "$LIMITATIONS"),
  "actions_taken": $(json_array_from_pipe_list "$ACTION_TAKEN"),
  "recommendations": $(json_array_from_pipe_list "$RECOMMENDATIONS"),
  "confidence": "low to medium",
  "labels": [
    "Emulated preflight only",
    "Not native Termux validation",
    "Not Android emulator validation unless proven",
    "Not real hardware validation"
  ]
}
EOF

if [ "$FAILED" -eq 1 ]; then
  exit 1
fi

exit 0
