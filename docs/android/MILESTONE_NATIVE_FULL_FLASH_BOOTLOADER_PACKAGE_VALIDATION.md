# Native Full-Flash Bootloader Package Validation

This milestone validates the firmware-package boundary on native Termux.

It is a package-validation milestone, not an upload, flashing, USB transfer, or
serial-monitor milestone.

## Objective

Validate that the firmware-package path can produce and consume a true
`full-flash` package on native Termux when the build metadata supports it.

## Scope

Validate only:

- firmware package correctness
- full-flash vs app-only package mode behavior
- bootloader, partition table, boot_app0, and application artifact detection
- manifest validation
- flash-plan validation
- prepare-only upload consumption if already supported

## Non-Goals

Do not claim or implement:

- real upload
- flashing
- USB transfer work
- serial monitor
- ESP32 protocol framing
- stream readiness

## Current State

The repository already provides:

- firmware-package generation in the compile path
- `manifest.json`
- `flash-plan.json`
- `validation-report.json`
- `analysis.json`
- `README_FLASHING.txt`
- prepare-only ACL upload consumption of a firmware package

The remaining question is whether native Termux can prove a `full-flash`
package with the bootloader artifact and boot flash entry present when the
metadata supports it.

## Validation Package

Use the exact command sequence below on native Termux.

### 1. Compile a known ESP32-S3 sketch

```sh
SKETCH_DIR="$HOME/Development/Sketches/esp32ultimate"
FQBN="esp32:esp32:esp32s3"
./arduino-cli compile -v --fqbn "$FQBN" "$SKETCH_DIR"
```

### 2. Locate the generated firmware package

```sh
PKG_DIR="$(find "$SKETCH_DIR/build" -path '*/firmware-package' -type d | head -n 1)"
test -n "$PKG_DIR"
find "$PKG_DIR" -maxdepth 2 -type f | sort
```

### 3. Confirm required package metadata exists

```sh
test -f "$PKG_DIR/manifest.json"
test -f "$PKG_DIR/flash-plan.json"
test -f "$PKG_DIR/validation-report.json"
test -f "$PKG_DIR/analysis.json"
test -f "$PKG_DIR/README_FLASHING.txt"
```

### 4. Confirm the package is full-flash

```sh
jq -e '.manifest.package_mode == "full-flash"' "$PKG_DIR/manifest.json"
```

### 5. Confirm the required artifacts are recorded in the manifest

```sh
jq -e '
  .manifest.artifacts["bootloader-binary"].path
  and .manifest.artifacts["partition-table-binary"].path
  and .manifest.artifacts["boot-app0-binary"].path
  and .manifest.artifacts["application-binary"].path
' "$PKG_DIR/manifest.json"
```

### 6. Confirm the flash plan contains the required entries

```sh
jq -e '
  .flash_plan.entries | map(.artifact) |
  (index("bootloader-binary") != null)
  and (index("partition-table-binary") != null)
  and (index("boot-app0-binary") != null)
  and (index("application-binary") != null)
' "$PKG_DIR/flash-plan.json"
```

### 7. Confirm the copied artifacts are present on disk

```sh
jq -r '.manifest.artifacts["bootloader-binary"].path' "$PKG_DIR/manifest.json" | xargs test -f
jq -r '.manifest.artifacts["partition-table-binary"].path' "$PKG_DIR/manifest.json" | xargs test -f
jq -r '.manifest.artifacts["boot-app0-binary"].path' "$PKG_DIR/manifest.json" | xargs test -f
jq -r '.manifest.artifacts["application-binary"].path' "$PKG_DIR/manifest.json" | xargs test -f
```

### 8. Exercise prepare-only upload consumption if it is already supported

```sh
./arduino-cli acl workflow upload "$PKG_DIR" --details
./arduino-cli --json acl workflow upload "$PKG_DIR"
```

## Expected Results

- `manifest.json` exists and reports `package_mode=full-flash`
- `flash-plan.json` exists and includes bootloader, partition table, boot_app0,
  and application entries
- `validation-report.json`, `analysis.json`, and `README_FLASHING.txt` exist
- copied artifact paths in the manifest resolve on disk
- prepare-only upload consumption accepts the package and does not open a real
  transport stream

## Failure Interpretation

- If `package_mode` is `app-only`, treat the result as a metadata fallback, not
  as a full-flash success.
- If any of the required artifacts are missing, the package is incomplete and
  the milestone remains open.
- If prepare-only upload rejects the package, record the package validation
  error and keep the failure scoped to package handling rather than transport
  execution.

## Evidence To Record

When the validation is run, record in `docs/android/VALIDATED_FINDINGS.md`:

- native Termux environment details
- the sketch and FQBN used
- the exact package path
- package mode
- artifact presence
- flash-plan entry presence
- prepare-only upload report status, if exercised
- remaining uncertainty, if any

## Completion Criteria

This milestone is complete when native Termux evidence shows:

- a true `full-flash` package when metadata supports it
- the package files exist and are internally consistent
- the prepare-only upload consumer accepts the package if exercised
- any fallback to `app-only` is explicitly reported and documented

## Safest Next Action

Run the validation package above on native Termux, then write the findings to
`VALIDATED_FINDINGS.md` and update `STATUS.md` only if the milestone state
changes.
