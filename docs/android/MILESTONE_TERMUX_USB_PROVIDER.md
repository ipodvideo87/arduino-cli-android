# Milestone: Termux USB Provider Validation Checklist

This milestone documents the native-Termux checks for the diagnostic-only
Termux USB provider.

It is a validation checklist, not an upload or flashing milestone.

Validated on:

- Samsung A17
- Android 16
- native Termux

Validated behaviors:

- discovery
- permission acquisition
- TERMUX_USB_FD handoff
- fd observation
- fd inspection

Not validated by this milestone:

- upload
- flashing
- serial monitor
- usable byte-stream endpoint
- byte-stream read/write

## Scope

Validate only:

- discovery
- permission acquisition
- diagnostics
- file-descriptor handoff evidence
- bounded TERMUX_USB_FD probe evidence

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
./arduino-cli acl transport probe-fd --device <device-path> --details
./arduino-cli acl transport stream-status --device <device-path> --details
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

### 8. TERMUX_USB_FD handoff probe

The provider now also exposes a safe fd-handoff probe surface:

```sh
./arduino-cli acl transport probe-fd --device <device-path>
./arduino-cli acl transport probe-fd --json --device <device-path>
./arduino-cli acl transport probe-fd-helper --json
./arduino-cli acl transport probe-fd-helper <fd> --json
```

The native working handoff command is:

```sh
termux-usb -r -E -e "./arduino-cli acl transport probe-fd-helper --json" /dev/bus/usb/001/002
```

Expected behavior:

- the probe reports whether `TERMUX_USB_FD` is present
- the helper records whether the fd value is valid and inspectable
- the helper records whether the fd came from `TERMUX_USB_FD` or a positional
  command-line argument
- the helper and session reports now carry an explicit stream `state`, which is
  `experimental` when the fd handoff has been observed but byte-stream support
  is still provisional
- the report stays diagnostic-only
- the report does not prove read/write stream support
- the report does not prove upload, flashing, or serial monitor behavior

Expected interpretation:

- `fd_observed=true` means the helper saw a Termux fd handoff
- `fd_valid=true` means the environment variable parsed as an integer fd
- `fd_inspectable=true` means the helper could inspect the file descriptor
- `fd_source=environment` means the helper used `TERMUX_USB_FD`
- `fd_source=argument` means the helper used the positional fd argument
- `state=experimental` means the fd handoff was observed and the stream
  foundation is present, but byte-stream behavior is not yet validated
- `stream_supported=false` means no usable byte-stream bridge is claimed yet
- `read_state`, `write_state`, `close_state`, `eof_state`, and
  `disconnect_state` remain bounded diagnostics until a real bridge exists
- if `termux-usb` returns `No such device` before helper JSON appears, the
  failure is a handoff/acquisition failure, not a helper parse failure

Root cause and fix:

- the first implementation only looked at `TERMUX_USB_FD`
- native Termux evidence showed `termux-usb -r -E -e <helper> <device>` is the
  working handoff shape on the Samsung A17 / Android 16 device
- the helper now accepts both env and argv fd sources so diagnostics can record
  either handoff mode clearly

## Native-Termux Validation Result

Observed on the Samsung A17 / Android 16 / native Termux environment:

- `./arduino-cli acl transport list`
  - Status: warning
  - Devices: 1
- `./arduino-cli acl transport diagnose --details`
  - device: `/dev/bus/usb/001/002`
  - provider: `termuxusb`
  - selected kind: `android-usb-fd`
  - `termux-usb -l` discovered the same path
  - the warning correctly explains that `TERMUX_USB_FD` is unavailable without
    the fd handoff path
- `termux-usb -l`
  - returned `/dev/bus/usb/001/002`
- `./arduino-cli acl transport acquire --device /dev/bus/usb/001/002`
  - USB permission granted
  - Status: passed
- `termux-usb -r -E -e "./arduino-cli acl transport probe-fd-helper --json" /dev/bus/usb/001/002`
  - `fd_env_present: true`
  - `fd_env_value: "7"`
  - `fd_observed: true`
  - `fd_valid: true`
  - `fd_inspectable: true`
  - `fd_source: "environment"`
  - `handoff_mode: "env"`
  - `status: "warning"`
  - `beginner_summary: "TERMUX_USB_FD observed via environment; stream support remains experimental"`
- `./arduino-cli acl transport probe-fd --device /dev/bus/usb/001/002`
  - now constructs `termux-usb -r -E -e "./arduino-cli acl transport probe-fd-helper --json" /dev/bus/usb/001/002`
  - if `termux-usb` returns `No such device` before helper JSON appears, that is
    a handoff/acquisition failure
- `./arduino-cli --json acl transport probe-fd --device /dev/bus/usb/001/002`
  - JSON output is validated
- `./arduino-cli acl transport stream-status --device /dev/bus/usb/001/002 --details`
  - generic alias for the same stream-state report

Validated conclusion:

- discovery is native-Termux validated
- permission acquisition is native-Termux validated
- TERMUX_USB_FD handoff is native-Termux validated
- fd observation is native-Termux validated
- fd inspection is native-Termux validated
- stream-state reporting is now modeled in code, but native byte-stream
  behavior remains unvalidated
- upload, flashing, and serial monitor remain unvalidated

Next recommended milestone:

- bounded byte-stream bridge foundation
- native validation of bounded `read_state` and `write_state` behavior without
  claiming upload or serial bridge success

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
