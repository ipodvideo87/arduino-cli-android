# ACL Patcher

The patcher module is responsible for preparing ELF binaries for the ACL runtime.
In v0.1 it only prints patch plans and safety checks; the actual binary rewrite path is
still experimental.

The interpreter and RPATH helpers verify that `patchelf` is available before doing any work.
