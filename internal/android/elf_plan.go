package android

import (
	"debug/elf"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type patchAction string

const (
	patchActionNoChange      patchAction = "no-change"
	patchActionRunPathOnly   patchAction = "runpath-only"
	patchActionLoaderAndPath patchAction = "loader-and-rpath"
	patchActionWrapperLaunch patchAction = "wrapper-launch"
	patchActionUnsupported   patchAction = "unsupported-skip"
	patchActionBlocked       patchAction = "blocked"
)

type elfProgramHeader struct {
	Type     string
	Offset   uint64
	VAddr    uint64
	FileSize uint64
	MemSize  uint64
	Flags    string
	Align    uint64
}

type elfAnalysis struct {
	Path              string
	PathClass         string
	Machine           elf.Machine
	FileType          string
	Interpreter       string
	ImportedLibraries []string
	RunPath           string
	RPath             string
	ProgramHeaders    []elfProgramHeader
	HasTLS            bool
	HasGNURelro       bool
	PageAligned       bool
	LoadSegmentCount  int
}

type patchPlan struct {
	Action         patchAction
	Reason         string
	Analysis       elfAnalysis
	Spec           patchSpec
	WrapperBackup  string
	WrapperTarget  string
	WrapperRuntime string
}

func analyzeELFForPatch(path string) (elfAnalysis, error) {
	f, err := elf.Open(path)
	if err != nil {
		return elfAnalysis{}, fmt.Errorf("open ELF %s: %w", path, err)
	}
	defer f.Close()

	analysis := elfAnalysis{
		Path:              path,
		PathClass:         classifyELFPath(path),
		Machine:           f.FileHeader.Machine,
		FileType:          strings.TrimPrefix(f.FileHeader.Type.String(), "ET_"),
		ImportedLibraries: nil,
	}

	if libs, err := f.ImportedLibraries(); err == nil {
		analysis.ImportedLibraries = append([]string(nil), libs...)
	} else {
		return elfAnalysis{}, fmt.Errorf("read DT_NEEDED from %s: %w", path, err)
	}
	if runPath, err := f.DynString(elf.DT_RUNPATH); err == nil && len(runPath) > 0 {
		analysis.RunPath = strings.Join(runPath, ":")
	}
	if rPath, err := f.DynString(elf.DT_RPATH); err == nil && len(rPath) > 0 {
		analysis.RPath = strings.Join(rPath, ":")
	}
	if interp, err := elfInterpreter(f); err == nil {
		analysis.Interpreter = interp
	}

	pageSize := uint64(os.Getpagesize())
	analysis.PageAligned = true
	for _, prog := range f.Progs {
		header := elfProgramHeader{
			Type:     prog.Type.String(),
			Offset:   prog.Off,
			VAddr:    prog.Vaddr,
			FileSize: prog.Filesz,
			MemSize:  prog.Memsz,
			Flags:    prog.Flags.String(),
			Align:    prog.Align,
		}
		analysis.ProgramHeaders = append(analysis.ProgramHeaders, header)
		if prog.Type == elf.PT_TLS {
			analysis.HasTLS = true
		}
		if prog.Type == elf.PT_GNU_RELRO {
			analysis.HasGNURelro = true
		}
		if prog.Type == elf.PT_LOAD {
			analysis.LoadSegmentCount++
			if prog.Align < pageSize || (prog.Vaddr%pageSize) != (prog.Off%pageSize) {
				analysis.PageAligned = false
			}
		}
	}
	if analysis.LoadSegmentCount == 0 {
		analysis.PageAligned = false
	}

	sort.SliceStable(analysis.ProgramHeaders, func(i, j int) bool {
		if analysis.ProgramHeaders[i].Offset != analysis.ProgramHeaders[j].Offset {
			return analysis.ProgramHeaders[i].Offset < analysis.ProgramHeaders[j].Offset
		}
		return analysis.ProgramHeaders[i].Type < analysis.ProgramHeaders[j].Type
	})

	return analysis, nil
}

func classifyELFPath(path string) string {
	clean := filepath.ToSlash(strings.ToLower(filepath.Clean(path)))
	switch {
	case strings.Contains(clean, "/libexec/gcc/"):
		return "gcc-libexec"
	case strings.Contains(clean, "/plugins/"):
		return "plugin"
	case strings.Contains(clean, "/bin/"):
		return "bin"
	case strings.Contains(clean, "/lib64/") || strings.Contains(clean, "/lib/"):
		return "shared-library"
	default:
		return "other"
	}
}

func planPatchForELF(analysis elfAnalysis, runtimeDir string) patchPlan {
	if analysis.Machine != elf.EM_AARCH64 {
		return patchPlan{
			Action:   patchActionUnsupported,
			Reason:   fmt.Sprintf("unsupported machine %q", analysis.Machine),
			Analysis: analysis,
		}
	}

	if analysis.PathClass == "gcc-libexec" {
		if analysis.FileType != "EXEC" && analysis.FileType != "DYN" {
			return patchPlan{
				Action:   patchActionBlocked,
				Reason:   fmt.Sprintf("GCC libexec executable %s has unexpected ELF type %q", analysis.Path, analysis.FileType),
				Analysis: analysis,
			}
		}
		return patchPlan{
			Action:         patchActionWrapperLaunch,
			Reason:         "GCC internal executables under libexec/gcc must not be rewritten with patchelf --set-interpreter because the loader rewrite changes the ELF layout and crashes during startup on native Termux",
			Analysis:       analysis,
			WrapperBackup:  filepath.Join(filepath.Dir(analysis.Path), ".acl", "original", filepath.Base(analysis.Path)),
			WrapperTarget:  analysis.Path,
			WrapperRuntime: runtimeDir,
		}
	}

	spec, ok := patchSpecForELFFields(analysis.Path, analysis.Machine, analysis.FileType, analysis.Interpreter, analysis.ImportedLibraries, runtimeDir)
	if !ok {
		return patchPlan{
			Action:   patchActionNoChange,
			Reason:   "ELF does not require Android patching",
			Analysis: analysis,
		}
	}
	if spec.setInterpreter {
		return patchPlan{
			Action:   patchActionLoaderAndPath,
			Reason:   "safe loader-and-RPATH patch",
			Analysis: analysis,
			Spec:     spec,
		}
	}
	if spec.rpath != "" {
		return patchPlan{
			Action:   patchActionRunPathOnly,
			Reason:   "safe RUNPATH-only patch",
			Analysis: analysis,
			Spec:     spec,
		}
	}
	return patchPlan{
		Action:   patchActionNoChange,
		Reason:   "ELF already satisfies Android runtime requirements",
		Analysis: analysis,
	}
}

func applyPatchPlan(plan patchPlan) error {
	switch plan.Action {
	case patchActionNoChange, patchActionUnsupported:
		return nil
	case patchActionBlocked:
		return fmt.Errorf("cannot patch %s: %s", plan.Analysis.Path, plan.Reason)
	case patchActionWrapperLaunch:
		return applyWrapperLaunch(plan)
	case patchActionLoaderAndPath, patchActionRunPathOnly:
		return applyPatchelfPlan(plan.Analysis.Path, plan.Spec)
	default:
		return fmt.Errorf("unsupported patch action %q for %s", plan.Action, plan.Analysis.Path)
	}
}

func applyPatchelfPlan(path string, spec patchSpec) error {
	if spec.setInterpreter && isGCCInternalExecutable(path) {
		return fmt.Errorf("refusing to apply --set-interpreter to GCC internal executable %q; use wrapper launch instead", path)
	}
	return patchWithPatchelf(path, spec)
}

func applyWrapperLaunch(plan patchPlan) error {
	backupPath := plan.WrapperBackup
	if backupPath == "" {
		return fmt.Errorf("missing wrapper backup path for %s", plan.Analysis.Path)
	}
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return fmt.Errorf("prepare wrapper backup directory for %s: %w", plan.Analysis.Path, err)
	}
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clear wrapper backup %s: %w", backupPath, err)
	}
	if err := os.Rename(plan.Analysis.Path, backupPath); err != nil {
		return fmt.Errorf("move GCC internal executable %s to %s: %w", plan.Analysis.Path, backupPath, err)
	}

	script := gccWrapperScript(filepath.Base(plan.Analysis.Path))
	if err := os.WriteFile(plan.Analysis.Path, []byte(script), 0o755); err != nil {
		_ = os.Rename(backupPath, plan.Analysis.Path)
		return fmt.Errorf("write GCC wrapper %s: %w", plan.Analysis.Path, err)
	}
	return nil
}

func gccWrapperScript(binaryName string) string {
	return strings.TrimSpace(fmt.Sprintf(`#!/bin/sh
set -eu
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
runtime_dir="$script_dir/../../../../.acl/runtime"
loader="$runtime_dir/ld-linux-aarch64.so.1"
target="$script_dir/.acl/original/%s"
exec "$loader" --library-path "$runtime_dir" "$target" "$@"`, binaryName)) + "\n"
}
