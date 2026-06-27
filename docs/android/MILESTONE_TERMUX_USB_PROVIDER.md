# Milestone: Termux USB Provider Validation Checklist

This milestone documents the native-Termux checks for the diagnostic-only
Termux USB provider.

It is a validation checklist, not an upload or flashing milestone.

## Scope

Validate only:

- discovery
- permission acquisition
- diagnostics
- file-descriptor handoff evidence

Do not use this checklist to claim:

- USB flashing success
- upload success
- serial monitor success

Native Termux remains the authority for Android behavior.

## Commands

Run these on the Android device in native Termux:

```sh
./arduino-cli acl transport list
./arduino-cli acl transport diagnose --details
./arduino-cli acl transport acquire --device <device-path>
```

Optional JSON forms, if you want machine-readable output:

```sh
./arduino-cli acl transport list --json
./arduino-cli acl transport diagnose --json
./arduino-cli acl transport acquire --json --device <device-path>
```

## Expected Results

### 1. No USB devices connected

Expected behavior:

- `acl transport list` reports a warning-style result
- the beginner summary says no USB devices were discovered
- device count is `0`
- `acl transport diagnose --details` reports the discovery trace from
  `termux-usb -l`

Example interpretation:

- this means the provider is working but no USB device is present
- this does not indicate a failure in upload or flashing

### 2. `termux-usb` unavailable

Expected behavior:

- `acl transport list` reports that `termux-usb` is unavailable on the host
- `acl transport diagnose --details` includes a limitation stating that the
  Termux USB command is unavailable
- traces should show the lookup or availability failure

Example interpretation:

- Termux API tooling is missing or not installed in the current environment
- the provider is unavailable until `termux-usb` exists in PATH

### 3. Android permission prompt appears

Expected behavior:

- `acl transport acquire --device <device-path>` triggers the Android USB
  permission flow
- the UI should show the Android permission prompt when Termux requests access
- the command should report a permission-acquisition trace for the selected
  device path

Example interpretation:

- the provider reached the Android permission boundary successfully
- this is permission acquisition evidence, not upload evidence

### 4. Permission is granted

Expected behavior:

- `acl transport acquire --device <device-path>` reports granted permission
- the beginner summary should indicate permission granted
- the professional details should include:
  - the exact `termux-usb` command
  - stdout
  - stderr
  - exit code
  - interpretation

If JSON is enabled:

- `permission.state` should be `granted`
- `permission.method` should record the `termux-usb -r` or `termux-usb -r -e`
  form that was used

### 5. Stale path or `No such device`

Expected behavior:

- `acl transport acquire --device <device-path>` reports a stale or unavailable
  device
- the professional details should preserve the exact stderr/stdout evidence
- the interpretation should mention `No such device` or the stale-path status

Example interpretation:

- the device path changed between enumeration and acquisition, or Android
  re-enumerated the device
- retry discovery before retrying acquisition

### 6. `TERMUX_USB_FD` observed

Expected behavior:

- when `TERMUX_USB_FD` is present in the native Termux environment, the
  diagnostics report should show file-descriptor handoff evidence
- the diagnostics output should mention `TERMUX_USB_FD`
- the selected endpoint should report file-descriptor export readiness

Example interpretation:

- the provider can observe a file descriptor handed off by `termux-usb -r -e`
- this is a transport-bridge signal, not a flashing claim

### 7. Endpoint export unsupported

Expected behavior:

- when `TERMUX_USB_FD` is absent, the provider should report endpoint export as
  unsupported or unavailable
- the diagnostics should explain that byte-stream export is not yet implemented

Example interpretation:

- discovery and permission acquisition can still be useful even without fd
  export
- upload and monitor remain unimplemented

## What Success Means

Success for this milestone means the provider can:

- list devices
- report accurate diagnostics
- attempt permission acquisition
- explain fd-handoff availability or absence

It does not mean upload, flashing, or monitor behavior has been proven.

## What to Record

When running native Termux validation, record:

- device model
- Android version
- Termux version
- `termux-api` / `termux-usb` availability
- command output
- whether the permission prompt appeared
- whether permission was granted
- whether the device path changed
- whether `TERMUX_USB_FD` was present
- whether the endpoint export remained unsupported

Store the results in:

- `docs/android/VALIDATED_FINDINGS.md`
- `STATUS.md`

