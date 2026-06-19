# ACL Runtime Store

This directory documents the ACL runtime store contract. It is not the copied
Termux/glibc runtime.

## Lifecycle Terms

- Runtime package: a prepared directory containing `manifest.json` plus runtime files.
- Installed runtime: a package copied into the ACL runtime root under `runtimes/<id>/`.
- Selected runtime: the compatible runtime chosen by the manager for a given ABI and architecture.
- Active runtime: the selected runtime recorded in `<runtime-root>/active.json`.

## Layout

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

`manifest.json` is the source of truth for a runtime package. The runtime manager uses it
to discover installed runtimes, validate their contents, and select a compatible runtime.

## Manifest Fields

The starter manifest format includes:

- `schema_version`
- `runtime_id`
- `runtime_version`
- `architecture`
- `supported_abis`
- `loader`
- `libraries`
- `hashes`
- `compatibility_level`
- `created_at`
- `build`
- `extensions`

Relative file paths are required. Absolute paths and path traversal are rejected.

## Notes

- Runtime packages should be produced for ACL, not copied from a Termux install.
- Execution is not part of this sprint.
- Validation is intentionally strict enough to catch broken packages before launch.
- Minimal fixtures in `internal/acl/runtime/testdata/minimal-runtime/` exist to prove
  lifecycle behavior before any real runtime builder is available.

## Example Commands

```bash
acl-runtime install <package-dir>
acl-runtime list
acl-runtime status
acl-runtime select --arch x86_64 --abi android-x86_64
acl-runtime activate <runtime-id>
acl-runtime deactivate
```
