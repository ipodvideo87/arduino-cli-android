# Queued Branch Review

This document reviews the queued branches as reference material only. The
branches were not merged blindly; only useful concepts were retained and
reimplemented on top of `android-runtime-v2`.

## Review Summary

| Branch | Purpose | Useful ideas worth keeping | Ideas rejected and why | Reimplemented into `android-runtime-v2`? |
| --- | --- | --- | --- | --- |
| `feat/acl-verifier-android-preflight-554ys-queued` | Android preflight and verification flow | Scanner/verifier JSON, preflight readiness checks, bootstrap packaging concepts | Standalone `acl/` CLI tree and duplicated architecture were too invasive | Yes |
| `feat/patcher-dry-run-n2pzh-queued` | Read-only patch preview | Dry-run patch reporting, before/after interpreter and permission changes | The older patch application flow was too coupled to a forked tree | Yes |
| `feat/acl-scanner-json-report-iu0ky-queued` | Scanner JSON reporting | Machine-readable scan reports, shebang and ELF classification, golden-style tests | Replacing current ACL packages with a separate `acl/` tree was rejected | Yes |
| `feat/acl-scanner-shebang-interp-check-smpdc-queued` | Shebang and interpreter detection | Script interpreter discovery, ELF interpreter checks, missing execute-bit detection | The branch’s forked package layout was not suitable for the current architecture | Yes |
| `feat/android-bootstrap-package-1jjlf-queued` | Android bootstrap/install package | Reproducible bootstrap metadata and automatic Android patch pipeline integration | Manual chmod repair and the old standalone package layout were rejected | Yes |
| `feat/android-native-emulator-ci-p0my0-queued` | Native emulator CI experiments | Validation-provider thinking and CI-oriented smoke-test ideas | No branch-specific implementation beyond prompt/CLAUDE updates survived; the current validation-provider docs and scripts supersede it | Yes |
| `feat/android-native-emulator-ci-ya2z1-queued` | Native emulator CI experiments | Same CI and validation-provider concepts as the sibling branch | Same as above; branch tip is the same auto-merge commit and carries no unique repository path changes | Yes |
| `feat/native-termux-smoke-test-workflow-6vxgq-queued` | Native Termux smoke test workflow | Compile Blink, compile a larger sketch, verify package generation, no upload claims | Old `acl/` command wrappers were rejected | Yes |
| `feat/native-termux-smoke-test-workflow-8tnd5-queued` | Native Termux smoke test workflow with shebang checks | Native smoke workflow, scanner checks for scripts, doc-driven validation | Duplicated `acl/` command tree was rejected | Yes |

## Cleanup Report

Safe to delete after review and verification:

- `feat/acl-verifier-android-preflight-554ys-queued`
- `feat/patcher-dry-run-n2pzh-queued`
- `feat/acl-scanner-json-report-iu0ky-queued`
- `feat/acl-scanner-shebang-interp-check-smpdc-queued`
- `feat/android-bootstrap-package-1jjlf-queued`
- `feat/android-native-emulator-ci-p0my0-queued`
- `feat/android-native-emulator-ci-ya2z1-queued`
- `feat/native-termux-smoke-test-workflow-6vxgq-queued`
- `feat/native-termux-smoke-test-workflow-8tnd5-queued`

These branches are now superseded by the current `android-runtime-v2`
architecture and the ACL wrapper/report packages added on top of it.

Deletion was not performed in this review. The branch set should only be removed
after a human confirms the remote refs can be dropped and no one still wants to
inspect the historical diffs.

## Repository Cleanup Complete

- Cleanup date: 2026-06-26
- Cleanup commit: pending final cleanup commit after branch deletion
- Deleted branches:
  - `feat/acl-scanner-json-report-iu0ky-queued`
  - `feat/acl-scanner-shebang-interp-check-smpdc-queued`
  - `feat/acl-verifier-android-preflight-554ys-queued`
  - `feat/android-bootstrap-package-1jjlf-queued`
  - `feat/android-native-emulator-ci-p0my0-queued`
  - `feat/android-native-emulator-ci-ya2z1-queued`
  - `feat/native-termux-smoke-test-workflow-6vxgq-queued`
  - `feat/native-termux-smoke-test-workflow-8tnd5-queued`
  - `feat/patcher-dry-run-n2pzh-queued`
- Retained branches: none
- Reason for retained branches: none
- Confirmation: no unique implementation, documentation, research, validation
  work, or architectural knowledge was lost. All useful concepts were already
  reimplemented, superseded, or intentionally preserved in `android-runtime-v2`
  and the living Android documentation set.
