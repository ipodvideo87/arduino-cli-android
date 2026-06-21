# Runtime Status

ACL runtime packages are ACL-owned artifacts, not copied Termux trees. The current
runtime manager defines the storage contract and validation rules, and ACL now has a
first experimental execution backend behind `acl-exec --apply`.

## Current Concern

The copied Termux/glibc runtime remains an investigation reference only.

- The glibc loader can contain hardcoded Termux paths.
- Copying `ld-linux-aarch64.so.1` and companion libraries is not enough to prove
  portability.
- An ELF patch can succeed while execution still fails during loading.
- Runtime portability must be verified separately from ELF patching.
- Minimal test fixtures are used to prove the package lifecycle independently of any
  particular runtime payload.

## Runtime Root Layout

The ACL runtime manager stores installed runtimes as:

```text
<runtime-root>/
  active.json
  runtimes/
    <runtime-id>/
      manifest.json
      loader/
      lib/
      metadata/
```

The exact subdirectory layout inside a runtime package is intentionally simple and
independent of the copied Termux/glibc tree. Future runtime builders can add more files
as long as they update the manifest.

The manager lifecycle is:

- install a prepared package into the ACL runtime root
- discover installed runtimes from disk
- validate file presence, hashes, and compatibility metadata
- select the best compatible runtime for a requested ABI and architecture
- activate the selected runtime by writing `active.json`
- deactivate by removing the active selection file

## Validation Model

Runtime validation is intentionally limited to manager-owned guarantees:

- the manifest must parse and match the installed tree
- loader and library files must exist where the manifest says they do
- recorded hashes must match the installed files
- compatibility metadata must match the requested ABI and architecture
- activation only succeeds after validation passes

These checks prove package integrity and selection safety. They do not prove that a
runtime can execute every Linux binary on Android.

The manager distinguishes between:

- runtime package: the source directory passed to `install`
- installed runtime: the copy stored under `runtimes/<runtime-id>/`
- selected runtime: the best compatible runtime returned by `select`
- active runtime: the runtime recorded in `active.json`

## Manifest Contract

The manifest records:

- runtime version
- runtime ID
- architecture
- supported ABI list
- loader metadata
- library metadata
- hashes
- compatibility level
- creation date
- build information
- extension fields for future growth

Validation rejects absolute paths and path traversal in manifest file entries.

## Lifecycle

- `install` copies a prepared runtime package into the ACL runtime root.
- `validate` checks file presence, ELF shape, architecture, and recorded hashes.
- `select` chooses a compatible runtime without launching anything.
- `activate` marks the chosen runtime as active only after validation passes.

The builder lifecycle is separate:

- `Validate()` checks the package inputs before any files are copied.
- `GenerateManifest()` builds the runtime manifest from validated inputs.
- `ComputeHashes()` produces integrity data for a package directory.
- `Package()` imports runtime assets and emits a reproducible package.
- `Verify()` checks a package on disk without installing it.

## Reproducibility Guarantees

The package format and builder tests require stable output for identical inputs.
Repeated builds should produce the same runtime ID, manifest content, hashes, checksums,
and directory layout. Any change in those results should be treated as a packaging
change.

## Manual Package Build

Use the builder CLI to create a package from source assets:

```bash
acl-build-runtime \
  --name acl-runtime-aarch64 \
  --version 0.1.0 \
  --arch aarch64 \
  --abi android-aarch64 \
  --compatibility experimental \
  --loader /path/to/ld-linux-aarch64.so.1 \
  --lib /path/to/libc.so.6 \
  --lib /path/to/libdl.so.2 \
  --output /tmp/acl-runtime-package
```

The resulting package includes `manifest.json`, `metadata.json`, `checksums.json`,
`version`, and the imported runtime assets. The CLI verifies the package before exit.

## Proof Target

The next proof target after this sprint is to build an ACL-native runtime package from
real runtime assets using the current builder, then rerun the same lifecycle tests
against that package. The next execution proof after that is to patch `esptool`, run
`esptool version`, and capture the exact linker/runtime error if execution still fails.
Execution support remains intentionally narrow until package management is proven
separately from launch.

## Execution Planning

`acl-exec` consumes the active runtime and the scanner output to build a plan before any
attempted launch. Dry-run planning is the default. `--apply` is explicit, uses the
selected runtime loader directly, and remains experimental until it is validated on a
fresh Termux install outside proot.

Successful execution in proot is not proof of Android-native compatibility.

## Tool Compatibility Integration

The tool compatibility layer feeds the runtime manager indirectly. `acl-scan compat`
identifies which installed Arduino tools look like Linux/glibc executables and therefore
may require an ACL runtime. That report does not activate runtimes or execute tools by
itself. It exists so future compatibility work can target real installed binaries with a
clear record of what each tool appears to need.

`acl-scan validate-compat` uses the same scan output to check whether the installed
package tree is internally consistent with ACL’s current patching rules. That step is a
precondition for later execution or compilation work, because it catches mismatches
between the scanner, runtime manager, and patching policy before any launch attempt.

Validation PASS means the compatibility data currently satisfies ACL’s rules. It does
not mean that the underlying tool actually executed on Android, and it does not prove
that compilation or upload workflows are finished. WARN means ACL found a real host tool
that still needs future compatibility work. FAIL means ACL found a broken or inconsistent
installed tool state.

## Security Assumptions

The manager assumes package authors are trusted to supply runtime assets and metadata.
It guards against malformed manifests, missing files, tampering, path traversal, and
symlink escapes, but it does not sandbox untrusted code or prove runtime portability.
