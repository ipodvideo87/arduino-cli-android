# Android Compatibility Layer (ACL)

ACL is the experimental runtime and toolchain compatibility layer for `arduino-cli-android`.
It provides a staged way to inspect Linux ELF binaries, verify the embedded runtime layout,
and eventually patch binaries so they can run under Android or Termux.

Status: v0.1 experimental

## What ACL does today

- Scans ELF files and reports interpreter, RPATH/RUNPATH, and imported libraries.
- Verifies basic runtime layout expectations.
- Documents the current patching and launcher flow without pretending it is production-ready.

## Module layout

- `builder`: build and package the ACL runtime artifacts.
- `scanner`: inspect ELF files and extract runtime metadata.
- `patcher`: plan and eventually apply ELF edits.
- `verifier`: check runtime and binary assumptions before launch.
- `launcher`: bootstrap the runtime environment for wrapped tools.
- `runtime`: embedded runtime layout and its experimental constraints.
- `database`: starter metadata used by ACL planning and compatibility checks.
- `tools`: small helper utilities and future command-line wrappers.
