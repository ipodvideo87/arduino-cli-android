# Android Compatibility Research

This is the living research log for Android and Termux compatibility findings in this repository.

Use it to capture confirmed behavior, evidence, and open questions before making Android-specific changes.

## Evidence Legend

- Confirmed device evidence: observed on native Termux or on an Android device.
- Confirmed repo evidence: verified in this repository through tests, code paths, or reproducible validation.
- Inference: a reasoned conclusion that still depends on further proof.

## 1. Native Termux vs Ubuntu/proot

- Native Termux is the production target and the source of truth for Android behavior.
  - Evidence: project documentation policy in `AGENTS.md`, `README.md`, and `STATUS.md`; native validation commands run on device.
- Ubuntu inside `proot-distro` is tooling-only and does not prove Android compatibility.
  - Evidence: project documentation policy; repeated native-vs-proot differences observed during development.

## 2. Android Bionic vs glibc

- Native Termux uses Android/Bionic runtime behavior by default.
  - Evidence: Termux execution environment documentation states Termux runs natively on the Android host OS and is linked against Android `bionic`.
- Ubuntu inside proot uses glibc behavior.
  - Evidence: Ubuntu/proot environment; glibc package behavior in Termux is separate from the default Termux runtime.
- A glibc runtime in Termux is a separate compatibility layer, not the default platform libc.
  - Evidence: `termux-pacman/glibc-packages` repository and native install behavior.

## 3. Filesystem Behavior

- Hard links fail in native Termux with `Permission denied`, even under `~/termux-env-test` in `/data/data/com.termux/files/home`.
  - Evidence: native device test:
    ```bash
    cd ~
    mkdir -p ~/termux-env-test
    cd ~/termux-env-test
    echo test > a.txt
    ln a.txt b.txt
    ```
    Result: `ln: failed to create hard link 'b.txt' => 'a.txt': Permission denied`
- This hard-link restriction is a native Android/Termux filesystem behavior difference, not a desktop Linux assumption.
  - Evidence: same device test; reproducible on Android 16 / arm64-v8a.

## 4. Archive Extraction Behavior

- Arduino CLI archive extraction needed a hard-link-to-copy fallback for native Termux.
  - Evidence: tool and platform installs failed on `TypeLink` entries before the fallback was added.
- The fallback is generic across archive formats used by Arduino CLI installs.
  - Evidence: AVR hard-linked executable and ESP32 hard-linked static archive both installed successfully after the shared extractor change.
- The shared extractor path is used by both resource installs and zip-library installs.
  - Evidence: `internal/arduino/resources/install.go` and `internal/arduino/libraries/librariesmanager/install.go` both call `resources.ExtractArchive(...)`.

## 5. ELF Interpreter and RUNPATH Patching

- GCC internal executables under `libexec/gcc/` are treated differently from normal tool drivers for RPATH construction.
  - Evidence: `internal/android/patcher.go` special-cases `isGCCInternalExecutable(path)` and adds `$.acl/runtime` plus sibling library search paths.
- Tool patching is applied after installation, not during extraction.
  - Evidence: `internal/arduino/cores/packagemanager/install_uninstall.go` calls `android.PatchToolForAndroid(...)` after `toolResource.Install(...)`.
- Builtin tools installed under `packages/builtin/tools/...` now also pass through the Android patcher after install.
  - Evidence: `commands/instances.go` now calls the same Android patch step after builtin tool installation.
- `patchelf --set-interpreter` is the operation that rewrites large GCC internal executables into a split-load layout with an extra low-address `LOAD` and a relocated `DYNAMIC` segment; `--set-rpath` alone preserves the original program-header shape.
  - Evidence: local comparison of upstream `cc1plus` from `xtensa-esp-elf-14.2.0_20260121-aarch64-linux-gnu.tar.gz` against copies patched with `patchelf 0.18.0`.
  - Original `cc1plus`: 10 program headers, `PT_INTERP=/lib/ld-linux-aarch64.so.1`, `DYNAMIC` at `0x1fe19f8`.
  - Patched with `--set-interpreter` only: 11 program headers, new low-address `LOAD` at `0x3e0000`, `DYNAMIC` moved to `0x66db88`.
  - Patched with `--set-rpath` only: still 10 program headers, original load layout retained.
- GCC internal executables under `libexec/gcc/` are now launched through a shell wrapper that preserves the original binary in `.acl/original/` and invokes the bundled loader with `--library-path`; they are no longer rewritten in place with `patchelf --set-interpreter`.
  - Evidence: repository implementation in `internal/android/elf_plan.go` and regression tests in `internal/android/patcher_test.go`.
  - This is the current generic launch strategy for GCC internal executables on Android.
- The GCC wrapper scrubs `LD_PRELOAD`, `LD_LIBRARY_PATH`, `LD_AUDIT`, Termux loader variables, and `QEMU_` / `PROOT_` state before invoking the bundled glibc loader.
  - Evidence: repository implementation in `internal/android/elf_plan.go` and regression tests in `internal/android/patcher_test.go`.

## 6. Dynamic Loader Behavior

- Android 10+ maps some system binaries and libraries execute-only.
  - Evidence: Android developer documentation.
- A `SIGSEGV` with `SEGV_ACCERR` can indicate invalid access to mapped memory permissions, including execute-only pages.
  - Evidence: Android documentation and the observed `SIGSEGV {si_code=SEGV_ACCERR}` on `cc1plus`.
- `$ORIGIN` is supported in GNU ld runtime search paths.
  - Evidence: GNU ld documentation.

## 7. Runtime Library Closure

- The Android runtime closure must source glibc libraries from Termux glibc locations, not from the default Bionic runtime.
  - Evidence: code path in `internal/android/runtime_closure.go` and native validation of copied glibc runtime files.
- `gcc-libs-glibc` provides `libstdc++.so.6`.
  - Evidence: native Termux package installation and runtime closure result.
- The current repo vendors the Termux glibc runtime set used by Android patching
  in `internal/android/runtime`; if a future dependency is not bundled there, the
  fallback source is native Termux `gcc-libs-glibc` / `glibc-repo`.
  - Evidence: repo runtime bundle contents, native install behavior, and the
    runtime-closure package hint for missing glibc libraries.
- `cc1plus` currently has only `libdl.so.2`, `libm.so.6`, and `libc.so.6` in `DT_NEEDED`, so additional runtime files should not be added unless proven by metadata or loader traces.
  - Evidence: `readelf -d` on the patched `cc1plus`.
- A present ELF executable can still fail with `fork/exec ...: no such file or directory` on native Android when its PT_INTERP points at an unavailable loader path.
  - Evidence: native builtin `ctags` failure before patching, where `file` and `readelf` showed an AArch64 ELF with `PT_INTERP /lib/ld-linux-aarch64.so.1`.
- Shell-script test fixtures that use `/usr/bin/env bash` do not generalize to native Termux; Android-aware exec tests now resolve a shell from `PREFIX/bin/sh` when `GOOS=android`.
  - Evidence: repository test update in `internal/acl/exec/exec_test.go` after the native Termux `fork/exec ... loader.sh: no such file or directory` failure.

## 8. GCC / Binutils Internal Executables

- `cc1plus` is a GCC internal executable under `libexec/gcc/`, not a top-level driver in `bin/`.
  - Evidence: file path and GCC installation layout.
- The current patching logic already recognizes GCC internal executables separately.
  - Evidence: `internal/android/patcher.go` and `internal/android/patcher_test.go`.
- The current open blocker is a `cc1plus` crash immediately after `execve`.
  - Evidence: native Termux `strace` output:
    `execve(.../cc1plus, ...) = 0`
    followed by `SIGSEGV {si_code=SEGV_ACCERR, ...}`

## 9. SELinux / Memory Protection / mmap / mprotect

- Android can reject or harden memory access patterns that are tolerated elsewhere.
  - Evidence: Android behavior documentation on execute-only mappings and `mprotect`.
- The current `cc1plus` crash has not yet been proven to be a code bug versus a glibc-loader/TLS/memory-protection incompatibility.
  - Evidence: crash occurs before `main`, immediately after `execve`.

## 10. Known Fixed Issues

- Hard-link archive extraction fallback.
  - Evidence: native AVR and ESP32 core installs now succeed after the shared extractor fix.
- Runtime closure now copies required glibc runtime libraries into `.acl/runtime`.
  - Evidence: native validation and the runtime closure implementation.
- The install pipeline now treats `permission-runtime-fixes` as a runtime execute-bit repair stage instead of rerunning the broader ELF patch pass.
  - Evidence: repository update in `internal/acl/install/android_executor.go` after the native Termux stage failure on permission repair.
- GCC internal executable patching now uses wrapper launch instead of interpreter rewrite.
  - Evidence: repository implementation, unit tests, and native Termux validation that now gets past the earlier `cc1plus` startup crash.
- Debug-only tools such as OpenOCD and GDB remain skipped for Android patching.
  - Evidence: `internal/android/patcher.go` and tests.
- Termux environment scrubbing is preserved.
  - Evidence: `main.go` and Android execution tests.
- Builtin `ctags` now executes under the Android patcher on native Termux, clearing the previous `fork/exec ... no such file or directory` failure.
  - Evidence: native Termux ESP32-S3 Blink compilation now completes successfully after the builtin patch pass.
- Native ESP32-S3 Blink compilation is now validated on-device.
  - Evidence: native Termux command `./arduino-cli compile -v --fqbn esp32:esp32:esp32s3 ~/Development/Sketches/Blink` completed successfully with the reported program storage and dynamic memory usage.

## 11. Known Open Issues

- Native Termux firmware upload and broader compile-and-upload proof are still pending.
  - Evidence: validation has reached a successful ESP32-S3 compile, but firmware upload has not yet been proven on-device.

## 12. Native Termux Validation Commands

Use native Termux as the final authority. The current validation commands are:

```bash
rm -rf ~/.arduino15/packages/esp32 ~/.arduino15/staging ~/.cache/arduino ~/.arduino15/tmp
./arduino-cli core install esp32:esp32
./arduino-cli compile -v --fqbn esp32:esp32:esp32s3 ~/Development/Sketches/Blink
```

## 13. Upload / Flash Architecture

- The Arduino CLI upload path is recipe-driven and ultimately executes the
  selected `upload.pattern`, `program.pattern`, or `bootloader.pattern`.
  - Evidence: `commands/service_upload.go` resolves the upload tool, handles
    optional reset behavior, expands the recipe, and then runs the tool.
- ESP32-family upload recipes are built around `esptool` and serial port
  properties.
  - Evidence: ESP32 platform test data shows `tools.esptool.upload.pattern`
    invoking `esptool` with `--port "{serial.port}"`, `--baud {upload.speed}`,
    and flash image arguments.
- `esptool` supports remote serial ports over RFC2217, so a loopback bridge can
  still preserve serial semantics when a direct PTY is not enough.
  - Evidence: Espressif remote serial port documentation.
- A serial-like endpoint is still the cleanest contract for Arduino CLI and
  serial monitor tooling.
  - Evidence: Arduino CLI upload is recipe-driven and `pySerial`-backed tools
    expect a serial port abstraction.
- Android should therefore own USB acquisition and expose a serial-like
  endpoint, not a board-specific uploader.
  - Evidence: Android USB permission model and the Termux API file-descriptor
    handoff path.

## 14. Android USB Transport Framework

- Termux-native USB access should be treated as a transport design problem, not
  just an upload-command problem.
  - Evidence: Android USB host APIs require app-scoped permission and explicit
    device selection, while Arduino CLI upload is recipe-driven and assumes a
    transport already exists.
- `termux-usb` is the current Termux-native acquisition path with the strongest
  evidence.
  - Evidence: Termux API source uses `TERMUX_USB_FD` and `termux-callback`, and
    release notes document USB fd and Android 14 USB fixes.
- Android USB host access remains app-scoped and permission-gated.
  - Evidence: `UsbManager.requestPermission(...)` and `UsbDeviceConnection`
    documentation.
- The bridge must be descriptor-driven and generic.
  - Evidence: `usb-serial-for-android` detects CDC/ACM by interface type and
    exposes raw read/write semantics for multiple serial chip families.
- The bridge should expose a PTY or RFC2217-compatible endpoint to existing
  tools.
  - Evidence: PTYs are a standard serial abstraction and RFC2217 preserves
    serial semantics across a socket boundary.
- A companion APK remains a last resort, not the preferred architecture.
  - Evidence: current project policy prefers Termux-native solutions first.

## 15. Evidence Log

- Native-device evidence from this workstream: `termux-usb -l` enumerates the
  device, the Android permission dialog appears, permission can be granted, and
  acquisition can still fail intermittently with `No such device`.
  - Status: confirmed on the target device by manual observation in this
    workstream; the failure mode is not yet explained.
- Upstream Termux API evidence: `termux-api.c` sends USB file descriptors
  through `termux-callback` and `SCM_RIGHTS`.
  - Source: [termux-api-package termux-api.c](https://github.com/termux/termux-api-package/blob/master/termux-api.c)
- Upstream Termux API evidence: release notes mention a fix for USB fd delivery
  and a fix for Android 14 USB hangs.
  - Source: [termux-api releases](https://github.com/termux/termux-api/releases)
- Android USB host evidence: permission is temporary until disconnect and
  communication is performed with `openDevice`, `claimInterface`, `controlTransfer`,
  and `bulkTransfer`.
  - Source: [UsbManager API](https://developer.android.com/reference/android/hardware/usb/UsbManager)
  - Source: [UsbDeviceConnection API](https://developer.android.com/reference/android/hardware/usb/UsbDeviceConnection)
- Android USB host evidence: enumeration can happen through attach intents or
  by querying the connected device list.
  - Source: [Android USB host overview](https://developer.android.com/develop/connectivity/usb/host)
- Generic USB serial evidence: `usb-serial-for-android` provides a raw serial
  port API and detects CDC/ACM by interface type.
  - Source: [usb-serial-for-android](https://github.com/mik3y/usb-serial-for-android)
- PTY evidence: a pseudoterminal is a bidirectional master/slave pair.
  - Source: [pty(7)](https://man7.org/linux/man-pages/man7/pty.7.html)
- Upload tooling evidence: `esptool` supports RFC2217 remote serial ports and
  uses pySerial for serial access.
  - Source: [Remote Serial Ports](https://docs.espressif.com/projects/esptool/en/latest/esp32/remote-serial-ports.html)
  - Source: [Troubleshooting](https://docs.espressif.com/projects/esptool/en/latest/esp32/troubleshooting.html)
- App precedent evidence: ArduinoDroid, ESP32_Flasher, and TCPUART all expose
  Android USB host serial or bridge behavior on this class of hardware.
  - Source: [ArduinoDroid](https://play.google.com/store/apps/details?id=name.antonsmirnov.android.arduinodroid2)
  - Source: [ESP32_Flasher](https://play.google.com/store/apps/details?id=com.esp_flash.esp_flash_app)
  - Source: [TCPUART](https://play.google.com/store/apps/details?hl=en_US&id=com.hardcodedjoy.tcpuart)

## 16. Open Questions

- Why does USB acquisition sometimes succeed and sometimes fail after permission?
- Is Android re-enumerating the device after permission is granted?
- Does the device path change between list, permission grant, and open?
- Is the observed latency a Termux API issue, an Android USB lifecycle issue,
  or a bridge bug?
- Is PTY enough for all supported tools, or will RFC2217 be required for some
  monitor or upload paths?
- Which package types should automatically run Android post-install patching and
  self-test today?

## 17. Native Android Experiments

The next experiments to run on device are:

1. `termux-usb -l`
2. `termux-usb -r <device>`
3. `termux-usb -e <probe> <device>`
4. `ls -l /dev/bus/usb/*/*` before and after disconnect/reconnect
5. dump raw USB descriptors from the opened fd
6. identify the CDC or vendor-specific serial interface from those descriptors
7. open the same fd from native code and verify the fd reaches the bridge
8. expose a PTY and connect a serial monitor to it
9. run an `esptool` chip-identification probe through the bridge
10. flash a known-good firmware package and verify the boot output

## Agent Workflow

Before fixing Android-specific behavior:

1. Read this file.
2. Check whether the behavior is already documented.
3. If it is not documented, research and gather evidence.
4. Add the finding to this file.
5. Implement a targeted fix.
6. Add regression tests.
7. Validate on native Termux.

## Notes

- Do not guess at Android behavior.
- Do not stack speculative patches when a root cause is still unclear.
- Preserve normal Linux behavior unless Android requires a scoped fallback.
- Update this log when new Android-vs-Linux differences are confirmed.
