# ACL Verifier

The **verifier** package implements a suite of Android/Termux pre-flight
environment checks that run before any glibc-linked tool is launched. It
detects configuration problems early and surfaces actionable remediation hints
so users can fix their environment rather than debug cryptic runtime errors.

## Checks

| Name | What it checks | Exit code on failure |
|---|---|---|
| `prefix-accessible` | Termux `$PREFIX` dir exists and `bin/`+`lib/` are accessible | 4 (filesystem) |
| `selinux-mode` | SELinux enforcing mode + dangerous process context detected | 3 (SELinux) |
| `proc-self-exe` | `/proc/self/exe` is readable | 6 (procfs) |
| `wx-restriction` | No W^X bits on Termux `bin/`+`lib/` | 5 (W^X) |
| `patchelf-present` | `patchelf` is installed and executable | 2 (missing dep) |
| `linker-present` | Termux dynamic linker exists under `PREFIX` | 2 (missing dep) |

## Exit Codes

| Code | Meaning |
|---|---|
| 0 | All checks passed |
| 2 | Missing dependency (patchelf, linker) |
| 3 | SELinux enforcing with dangerous process context |
| 4 | Filesystem / PREFIX problem |
| 5 | W^X restriction on Termux directories |
| 6 | `/proc` filesystem restricted |
| 99 | Internal error during a check |

## Usage via `acl-verify`

```sh
# Run all checks with human-readable output
acl-verify

# Run only specific checks
acl-verify --check prefix-accessible --check selinux-mode

# Emit machine-readable JSON (useful in CI pipelines)
acl-verify --json

# List all available checks
acl-verify --list

# Suppress output; communicate only via exit code
acl-verify --quiet
```

### Example output

```
[PASS] prefix-accessible: PREFIX accessible: /data/data/com.termux/files/usr
[PASS] selinux-mode: SELinux is enforcing (context: u:r:untrusted_app:s0)
[PASS] proc-self-exe: /proc/self/exe -> /data/data/com.termux/files/home/bin/acl-verify
[PASS] wx-restriction: No W^X violations detected in /data/data/com.termux/files/usr/bin, /data/data/com.termux/files/usr/lib
[FAIL] patchelf-present: patchelf not found on PATH or under Termux PREFIX
       hint: Install patchelf in Termux: pkg install patchelf
             patchelf is required for ACL apply-mode ELF patching.
[PASS] linker-present: Linker found: /data/data/com.termux/files/usr/glibc/lib/ld-linux-aarch64.so.1

5/6 checks passed, 1 failed
```

### JSON output shape

```json
[
  {
    "name": "prefix-accessible",
    "passed": true,
    "exit_code": 0,
    "message": "PREFIX accessible: /data/data/com.termux/files/usr"
  },
  {
    "name": "patchelf-present",
    "passed": false,
    "exit_code": 2,
    "message": "patchelf not found on PATH or under Termux PREFIX",
    "hint": "Install patchelf in Termux: pkg install patchelf\n       patchelf is required for ACL apply-mode ELF patching."
  }
]
```

## Go API

```go
import "github.com/arduino/arduino-cli/acl/verifier"

// Run all checks
results := verifier.RunAll()
code    := verifier.OverallExitCode(results)

// Run selected checks
results = verifier.RunSelected([]string{"patchelf-present", "linker-present"})

// Inspect a result
for _, r := range results {
    fmt.Println(r) // [PASS]/[FAIL] name: message
}
```

## Adding a New Check

1. Write a function `func CheckMyThing() verifier.Result { … }`.
2. Append a `verifier.Check{Name: "my-thing", Description: "…", Run: CheckMyThing}` to `verifier.All`.
3. Add a test in `verifier_test.go`.

The `Run` function must **not** call `os.Exit`. Exit-code handling is the
responsibility of the caller (`acl-verify`'s `main`).
