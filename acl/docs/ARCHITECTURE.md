# ACL Architecture

ACL is organized as a pipeline:

`scanner` -> `database` -> `builder` -> `patcher` -> `verifier` -> `exec` -> `launcher`

The runtime manager sits alongside that pipeline as the control plane for installed
runtimes. It does not execute binaries; it discovers, validates, selects, and tracks
runtime packages.

## Runtime Flow

- `scanner` inspects ELF files and extracts runtime requirements.
- the scanner also inventories installed Arduino tool executables and classifies their compatibility.
- `database` stores compatibility metadata and known signatures.
- `builder` produces ACL-native runtime packages.
- `patcher` prepares ELF binaries for a selected runtime.
- `verifier` checks that a runtime tree and target binary are compatible.
- `exec` builds a dry-run execution plan and, only when requested, hands off to an
  execution backend.
- `launcher` will eventually start patched binaries inside the chosen runtime.
- `internal/acl/runtime` manages installed runtime packages and selection state.
- `internal/acl/builder` generates reproducible runtime packages and package metadata.
- `acl/cmd/acl-runtime` exposes the runtime manager as a standalone control-plane CLI.
- `acl/cmd/acl-build-runtime` exposes the builder as a manual packaging tool.

## Runtime Model

Installed runtimes live under an ACL-owned root directory:

`<runtime-root>/runtimes/<runtime-id>/manifest.json`

The root also stores the active selection record:

`<runtime-root>/active.json`

This layout is independent of the copied Termux/glibc tree. The copied runtime may still
be used for investigation, but it is not the production format.

The runtime manager distinguishes these concepts:

- runtime package: a source directory with `manifest.json` and runtime files
- installed runtime: a package copied into the ACL runtime root
- selected runtime: the best compatible runtime chosen for a requested ABI/architecture
- active runtime: the selected runtime recorded in `active.json`

Minimal test fixtures are used to prove this lifecycle independently of any particular
runtime payload.

## Tool Compatibility Layer

The first tool compatibility layer is focused on understanding the Arduino ecosystem,
not executing it.

`acl-scan compat` walks an Arduino packages directory, discovers executable candidates,
classifies them, and records the metadata ACL needs for later execution work:

- executable type
- architecture
- interpreter
- shared library dependencies
- RPATH/RUNPATH
- hardcoded absolute paths
- compatibility category

Linux/glibc executables discovered by this layer become candidates for future
runtime-managed execution. Static ELFs, native Android-compatible binaries, scripts, and
unknown artifacts are reported separately so the repository can reason about them without
pretending they are already supported.

`acl-scan validate-compat` runs the same scan and then checks the resulting
classifications against ACL’s current consistency rules. Validation is the gate before
execution or compilation work, because it proves that the scanner and patching policy
agree on what each installed tool appears to be.

Validation PASS means the installed tree matches the current rules. It does not prove
that every tool executes successfully on Android, and it does not prove that sketch
compilation or firmware upload already works. WARN means ACL found a real host tool that
needs future compatibility work. FAIL means ACL found a broken or inconsistent installed
tool state.

## Hardening Guarantees

The current hardening work proves that the runtime stack rejects common failure modes
before activation:

- malformed or missing manifests are rejected
- invalid metadata and unsupported package format versions are rejected
- duplicate runtime IDs do not overwrite an existing install
- path traversal, absolute paths, and symlink escapes are rejected
- tampered packages fail verification instead of activating

These checks improve safety and reproducibility, but they do not prove execution
compatibility. They only prove that the package manager is difficult to confuse.

## Execution Sprint 1

The first execution layer is intentionally narrow:

- dry-run planning is the default
- `--apply` is explicit and experimental
- the planner reuses scanner output and runtime validation before building a plan
- the first real execution backend invokes the selected runtime loader explicitly
- execution proof on native Android is still a later milestone

Android-native validation must still be performed from a fresh Termux environment after
publishing changes. Proot execution is not proof of Android compatibility.

## Architectural Constraints

- Prefer native Android/bionic builds when practical.
- Use ACL-managed runtime packages for Linux binaries that still require glibc semantics.
- Validate runtime integrity before selection or activation.
- Keep execution separate from package discovery and validation.
- Keep compatibility validation separate from execution so failures stay observable and
  reproducible.

## Next Milestone

The current builder already emits ACL-native runtime packages that match this contract.
The next implementation milestone is to build packages from real runtime assets and then
validate execution behavior against those packages without changing the package
lifecycle model.

## Manual Build Flow

The builder CLI takes a runtime name, version, architecture, ABI, loader, and one or
more libraries. It copies those assets into the package layout, writes the manifest and
metadata files, computes checksums, and verifies the package in place. The runtime
manager remains separate and only consumes the finished package.
