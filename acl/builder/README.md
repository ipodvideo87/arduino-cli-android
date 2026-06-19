# ACL Builder

The builder module assembles ACL runtime packages. It does not execute binaries, patch
toolchains, or talk to Arduino CLI.

Current implementation:

- `internal/acl/builder` provides the Go API.
- `Package()` creates ACL runtime packages that conform to the runtime manager contract.
- `Verify()` checks package integrity without installing it.

Package format:

- `manifest.json` - runtime contract consumed by the runtime manager
- `metadata.json` - builder metadata and package format version
- `checksums.json` - file integrity map for the package contents
- `version` - package format version marker
- `loader/` and `lib/` - imported runtime assets

Manual build example:

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

The CLI copies the supplied loader and libraries into the package layout, writes the
manifest and metadata files, computes checksums, and verifies the package before exit.

The builder is intentionally separate from the runtime manager. The builder produces
packages; the manager installs, validates, and selects them.

## Validation Model

Builder validation is strict by design:

- manifest fields must use supported architecture, ABI, and compatibility values
- package inputs must not contain empty paths, duplicate libraries, or symlink escapes
- package output must not overlap the input tree
- package contents must be regular files inside the package layout

## Reproducibility

The builder is designed to produce stable package layouts from the same inputs. The test
suite checks that repeated builds yield the same runtime ID, manifest content, hashes,
checksums, and directory structure.

## Security Assumptions

The builder proves package structure and integrity only. It does not prove that a runtime
will execute successfully on Android, and it does not validate the semantics of copied
glibc assets. That execution proof remains a later milestone.

## What the Hardening Tests Prove

- malformed or incomplete packages are rejected
- absolute paths and path traversal are rejected
- tampered or incomplete packages fail verification
- reproducible package output is enforced for the same inputs
- CLI input errors produce early, actionable failures
