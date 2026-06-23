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
- `patchelf --set-interpreter` is the operation that rewrites large GCC internal executables into a split-load layout with an extra low-address `LOAD` and a relocated `DYNAMIC` segment; `--set-rpath` alone preserves the original program-header shape.
  - Evidence: local comparison of upstream `cc1plus` from `xtensa-esp-elf-14.2.0_20260121-aarch64-linux-gnu.tar.gz` against copies patched with `patchelf 0.18.0`.
  - Original `cc1plus`: 10 program headers, `PT_INTERP=/lib/ld-linux-aarch64.so.1`, `DYNAMIC` at `0x1fe19f8`.
  - Patched with `--set-interpreter` only: 11 program headers, new low-address `LOAD` at `0x3e0000`, `DYNAMIC` moved to `0x66db88`.
  - Patched with `--set-rpath` only: still 10 program headers, original load layout retained.

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
- Debug-only tools such as OpenOCD and GDB remain skipped for Android patching.
  - Evidence: `internal/android/patcher.go` and tests.
- Termux environment scrubbing is preserved.
  - Evidence: `main.go` and Android execution tests.

## 11. Known Open Issues

- `cc1plus` segfaults immediately after `execve` during ESP32-S3 Blink compilation on native Termux.
  - Evidence: native `strace` and the current validation blocker reported by the device.
- It is not yet proven whether `cc1plus` requires additional runtime files beyond the current glibc set.
  - Evidence: current `DT_NEEDED` only names `libdl.so.2`, `libm.so.6`, and `libc.so.6`.
- The exact failure layer for `cc1plus` is still under investigation: loader, relocation, TLS, `mprotect`, or another glibc/Android interaction.
  - Evidence: crash timing and `SEGV_ACCERR`.
- The direct `cc1plus` crash is now narrowed to the interpreter-rewrite path rather than missing libraries or compiler arguments.
  - Evidence: the crash persists for direct `cc1plus --help`, and the rewritten binary's fault address lands inside the relocated `.dynamic` area produced by `patchelf --set-interpreter`.

## 12. Native Termux Validation Commands

Use native Termux as the final authority. The current validation commands are:

```bash
rm -rf ~/.arduino15/packages/esp32 ~/.arduino15/staging ~/.cache/arduino ~/.arduino15/tmp
./arduino-cli core install esp32:esp32
./arduino-cli compile -v --fqbn esp32:esp32:esp32s3 ~/Development/Sketches/Blink
```

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
