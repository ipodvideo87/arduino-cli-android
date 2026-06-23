# Native ESP32-S3 Compile Milestone

## Milestone

Native Termux on Android can now compile an ESP32-S3 Blink sketch successfully.

Validation command:

```bash
./arduino-cli compile -v --fqbn esp32:esp32:esp32s3 ~/Development/Sketches/Blink
```

Successful output:

```text
Sketch uses 300684 bytes (22%) of program storage space. Maximum is 1310720 bytes.
Global variables use 21992 bytes (6%) of dynamic memory, leaving 305688 bytes for local variables. Maximum is 327680 bytes.
```

## Original Blockers

1. Archive extraction failed on hard-link entries during tool install.
2. Android runtime closure could not resolve required glibc runtime libraries for GCC-based tools.
3. GCC internal executables such as `cc1plus` crashed after `patchelf --set-interpreter` rewrote the ELF layout.
4. Builtin tools such as `ctags` were present but not Android-patched, so their ELF interpreter still pointed at `/lib/ld-linux-aarch64.so.1`.

## Root Causes

- Native Termux denies hard links, so archive extraction needed a copy fallback.
- Termux glibc runtime libraries were required for the vendored runtime bundle.
- `patchelf --set-interpreter` was unsafe for GCC libexec binaries under `libexec/gcc/` because it changed the loader layout.
- Builtin tool installation bypassed the Android patch pass.

## Fixes Implemented

- Added a generic archive hard-link fallback that copies link targets when hard links are rejected.
- Vendored the required glibc runtime libraries into `internal/android/runtime`.
- Kept GCC internal executables on a wrapper-launch path instead of rewriting them in place with `patchelf --set-interpreter`.
- Added ELF analysis and patch planning so patch behavior depends on the executable shape and path class.
- Patched builtin tools after install so builtin `ctags` receives the same Android handling as normal tools.
- Scrubbed launcher environment variables before invoking the bundled glibc loader.

## Final Native Termux Validation

```bash
rm -rf ~/.arduino15/packages/esp32 ~/.arduino15/staging ~/.cache/arduino ~/.arduino15/tmp
./arduino-cli core install esp32:esp32
./arduino-cli compile -v --fqbn esp32:esp32:esp32s3 ~/Development/Sketches/Blink
```

## Remaining Work

- Verify firmware upload on real ESP32-S3 hardware.
- Validate serial port access on native Termux.
- Validate the `esptool` upload path.
- Exercise the serial monitor path.
- Test library-heavy sketches and broader compile coverage.

