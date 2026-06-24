// Package patcher implements the ACL ELF patcher stage.
//
// It computes and (optionally) applies the ELF field edits required to make
// glibc-linked binaries run under the ACL runtime on Android/Termux:
//
//   - PT_INTERP rewrite — replace the glibc dynamic linker path with the ACL
//     runtime loader.
//   - RUNPATH injection — point DT_RUNPATH at the ACL runtime library
//     directory.
//   - RPATH migration — remove legacy DT_RPATH entries in favour of RUNPATH.
//
// # Dry-run first
//
// No file is ever modified without an explicit Apply call.  ComputePlan and
// ComputePlans are read-only: they open binaries with debug/elf, extract the
// relevant fields, and return a PatchPlan describing every intended change.
//
// # Rendering
//
// WritePlan renders a slice of PatchPlans as a unified-diff-style report that
// is suitable for terminal output or CI log files.  The report shows current
// and proposed values for each ELF field alongside a human-readable reason.
//
// # Applying
//
// Apply calls patchelf (an external binary that must be on PATH) to rewrite
// the ELF fields described in the plans.  patchelf is only required for apply
// mode; dry-run mode has no external dependencies.
//
// # GCC libexec
//
// Binaries under libexec/gcc/ are classified as requiring wrapper-launch (not
// patchelf) and are returned as skipped plans.  The wrapper-launch logic lives
// in internal/android/elf_plan.go.
//
// See acl/docs/PATCHING.md for the full rationale and architecture.
package patcher
