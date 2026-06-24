// Package patcher implements ELF binary patching for Android/Termux compatibility.
//
// The core abstraction is a PatchPlan: a description of every ELF field edit
// needed to make a binary runnable under the ACL runtime.  Plans are computed
// without touching the filesystem so they can be inspected (dry-run) or
// applied (live) independently.
package patcher

import (
	"debug/elf"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FieldEdit describes a single ELF field that needs to be changed.
type FieldEdit struct {
	// Field is the ELF dynamic-section field being modified, e.g.
	// "PT_INTERP", "RUNPATH", "RPATH".
	Field string

	// Current is the value that exists in the binary right now.
	// Empty string means the field is absent.
	Current string

	// Proposed is the value that patchelf would write.
	// Empty string means the field would be removed.
	Proposed string

	// Reason is a short human-readable explanation of why this edit is
	// needed.
	Reason string
}

// PatchPlan holds the complete set of edits planned for one ELF file.
// When Edits is empty the binary needs no patching.
type PatchPlan struct {
	// Path is the absolute or relative path of the ELF file.
	Path string

	// Skipped is true when the file was deliberately not analysed (e.g.
	// non-ELF, wrong architecture, GCC libexec stub).  When true, Edits is
	// always empty and SkipReason explains why.
	Skipped bool

	// SkipReason is a short explanation populated only when Skipped == true.
	SkipReason string

	// Edits is the ordered list of field changes.  Consumers must apply them
	// in the order given.
	Edits []FieldEdit
}

// NeedsPatching reports whether at least one edit is required.
func (p PatchPlan) NeedsPatching() bool {
	return !p.Skipped && len(p.Edits) > 0
}

// PlanSummary aggregates counts across a set of plans for quick decisions in
// CI pipelines.
type PlanSummary struct {
	Total         int
	NeedPatching  int
	AlreadyOK     int
	Skipped       int
}

// Summarise computes aggregate statistics from a slice of plans.
func Summarise(plans []PatchPlan) PlanSummary {
	s := PlanSummary{Total: len(plans)}
	for _, p := range plans {
		switch {
		case p.Skipped:
			s.Skipped++
		case p.NeedsPatching():
			s.NeedPatching++
		default:
			s.AlreadyOK++
		}
	}
	return s
}

// PlanOptions controls how plans are computed.
type PlanOptions struct {
	// RuntimeDir is the directory that contains the glibc-compatible loader
	// and shared libraries.  It is used to compute proposed RUNPATH/RPATH
	// values and the PT_INTERP path.
	RuntimeDir string

	// LoaderName is the basename of the dynamic linker inside RuntimeDir,
	// e.g. "ld-linux-aarch64.so.1".  When empty the default for aarch64 is
	// used.
	LoaderName string
}

func (o PlanOptions) loaderName() string {
	if o.LoaderName != "" {
		return o.LoaderName
	}
	return "ld-linux-aarch64.so.1"
}

func (o PlanOptions) loaderPath() string {
	return filepath.Join(o.RuntimeDir, o.loaderName())
}

// ComputePlan inspects a single ELF file and returns the PatchPlan that
// describes every edit needed to make it run under the ACL runtime.
// No files are modified.
func ComputePlan(path string, opts PlanOptions) (PatchPlan, error) {
	f, err := elf.Open(path)
	if err != nil {
		// Not an ELF file — skip silently.
		return PatchPlan{
			Path:       path,
			Skipped:    true,
			SkipReason: fmt.Sprintf("not a valid ELF file: %v", err),
		}, nil
	}
	defer f.Close()

	// Only handle aarch64 binaries.
	if f.Machine != elf.EM_AARCH64 {
		return PatchPlan{
			Path:       path,
			Skipped:    true,
			SkipReason: fmt.Sprintf("unsupported machine %s (only EM_AARCH64 is patched)", f.Machine),
		}, nil
	}

	// GCC internal executables must use wrapper launch, not patchelf.
	if isGCCLibexec(path) {
		return PatchPlan{
			Path:       path,
			Skipped:    true,
			SkipReason: "GCC libexec binary: requires wrapper-launch, not patchelf (see ACL PATCHING.md)",
		}, nil
	}

	plan := PatchPlan{Path: path}

	currentInterp := readInterpreter(f)
	currentRunPath := readDynString(f, elf.DT_RUNPATH)
	currentRPath := readDynString(f, elf.DT_RPATH)

	proposedInterp := ""
	needsInterp := false

	// Only executables (ET_EXEC, ET_DYN used as executable) carry PT_INTERP.
	// Shared libraries typically do not, but glibc ships libc.so as ET_DYN
	// with a PT_INTERP; we patch any ELF that already has one.
	if currentInterp != "" && !isAndroidLoader(currentInterp) {
		proposedInterp = opts.loaderPath()
		needsInterp = true
	}

	proposedRunPath := ""
	needsRunPath := false

	// If the binary links against glibc-world libraries it needs an RUNPATH
	// that points at our runtime directory.
	libs, _ := f.ImportedLibraries()
	if needsAndroidRunPath(currentInterp, libs) {
		proposedRunPath = opts.RuntimeDir
		// Only add the edit when the current value differs.
		if currentRunPath != proposedRunPath {
			needsRunPath = true
		}
	}

	// If there is a legacy RPATH we prefer to migrate it to RUNPATH and clear
	// the old RPATH.  patchelf --set-rpath sets RUNPATH and removes RPATH.
	needsClearRPath := currentRPath != "" && currentRPath != proposedRunPath

	if needsInterp {
		plan.Edits = append(plan.Edits, FieldEdit{
			Field:    "PT_INTERP",
			Current:  currentInterp,
			Proposed: proposedInterp,
			Reason:   "binary uses a glibc dynamic linker; replace with ACL runtime loader",
		})
	}

	if needsRunPath {
		plan.Edits = append(plan.Edits, FieldEdit{
			Field:    "RUNPATH",
			Current:  currentRunPath,
			Proposed: proposedRunPath,
			Reason:   "set RUNPATH to ACL runtime directory so the loader finds glibc-world libraries",
		})
	}

	if needsClearRPath && !needsRunPath {
		// If we are already setting RUNPATH above, patchelf --set-rpath will
		// implicitly clear RPATH, so we don't need a separate edit.  Only
		// emit this edit when we are NOT also changing RUNPATH.
		plan.Edits = append(plan.Edits, FieldEdit{
			Field:    "RPATH",
			Current:  currentRPath,
			Proposed: "",
			Reason:   "remove legacy RPATH; RUNPATH is preferred and already set",
		})
	}

	return plan, nil
}

// ComputePlans computes a PatchPlan for every path in paths.
// Errors from individual files are embedded in their PatchPlan.SkipReason
// rather than aborting the whole scan, so callers always receive a result for
// every input path.
func ComputePlans(paths []string, opts PlanOptions) []PatchPlan {
	plans := make([]PatchPlan, 0, len(paths))
	for _, p := range paths {
		plan, err := ComputePlan(p, opts)
		if err != nil {
			plans = append(plans, PatchPlan{
				Path:       p,
				Skipped:    true,
				SkipReason: fmt.Sprintf("analysis error: %v", err),
			})
			continue
		}
		plans = append(plans, plan)
	}
	return plans
}

// ──────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────────────────────────────────────────

// readInterpreter returns the PT_INTERP path embedded in the ELF, or "" when
// no PT_INTERP segment is present.
func readInterpreter(f *elf.File) string {
	for _, prog := range f.Progs {
		if prog.Type != elf.PT_INTERP {
			continue
		}
		buf := make([]byte, prog.Filesz)
		if _, err := prog.ReadAt(buf, 0); err != nil {
			return ""
		}
		// PT_INTERP is a NUL-terminated string.
		return strings.TrimRight(string(buf), "\x00")
	}
	return ""
}

// readDynString returns the first value of a DT_* string tag, or "" when
// absent.
func readDynString(f *elf.File, tag elf.DynTag) string {
	vals, err := f.DynString(tag)
	if err != nil || len(vals) == 0 {
		return ""
	}
	return strings.Join(vals, ":")
}

// isAndroidLoader reports whether interp is already an Android/Bionic or
// Termux-native loader path that needs no rewriting.
func isAndroidLoader(interp string) bool {
	androidPaths := []string{
		"/system/bin/linker",
		"/system/bin/linker64",
		"/apex/com.android.runtime/bin/linker64",
		"/data/data/com.termux/files/usr/lib/",
	}
	for _, p := range androidPaths {
		if strings.HasPrefix(interp, p) {
			return true
		}
	}
	return false
}

// needsAndroidRunPath reports whether the binary needs an ACL RUNPATH set.
// We require patching when the binary either has a non-Android interpreter or
// any of its DT_NEEDED entries look like glibc-world libraries.
func needsAndroidRunPath(interp string, libs []string) bool {
	if interp != "" && !isAndroidLoader(interp) {
		return true
	}
	for _, lib := range libs {
		if isGlibcWorldLibrary(lib) {
			return true
		}
	}
	return false
}

// isGlibcWorldLibrary reports whether lib is a library that is typically
// provided by a glibc-based Linux distribution rather than Android/Bionic.
func isGlibcWorldLibrary(lib string) bool {
	glibcLibs := []string{
		"libc.so.6",
		"libm.so.6",
		"libpthread.so.0",
		"libdl.so.2",
		"librt.so.1",
		"libstdc++.so.6",
		"libgcc_s.so.1",
		"libgomp.so.1",
		"libz.so.1",
		"libusb-1.0.so.0",
		"libasan.so",
		"libubsan.so",
	}
	for _, g := range glibcLibs {
		if lib == g || strings.HasPrefix(lib, g) {
			return true
		}
	}
	return false
}

// isGCCLibexec reports whether path is under a GCC libexec directory.
func isGCCLibexec(path string) bool {
	clean := filepath.ToSlash(strings.ToLower(filepath.Clean(path)))
	return strings.Contains(clean, "/libexec/gcc/")
}

// CollectELFPaths walks root recursively and returns paths of regular files
// that appear to be ELF binaries (by magic bytes), skipping symlinks.
func CollectELFPaths(root string) ([]string, error) {
	var paths []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if looksLikeELF(path) {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

// looksLikeELF peeks at the first 4 bytes for the ELF magic number.
func looksLikeELF(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	magic := make([]byte, 4)
	if _, err := f.Read(magic); err != nil {
		return false
	}
	return magic[0] == 0x7f && magic[1] == 'E' && magic[2] == 'L' && magic[3] == 'F'
}
