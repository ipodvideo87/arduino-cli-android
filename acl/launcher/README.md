# ACL Launcher

The launcher module is the future execution backend for ACL. It will eventually start
patched tools inside the selected runtime, but it does not own planning, selection, or
package management.

Current status:

- execution planning now exists in `acl-exec`
- real launch support is still experimental
- dry-run planning is the default behavior
- Android-native validation still needs to happen from a fresh Termux environment

The launcher remains a structural placeholder until the execution backend is proven on
device.
