package android

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type PatchPreviewReport struct {
	Root       string              `json:"root"`
	RuntimeDir string              `json:"runtime_dir,omitempty"`
	Entries    []PatchPreviewEntry `json:"entries"`
	Summary    PatchPreviewSummary `json:"summary"`
}

type PatchPreviewSummary struct {
	TotalCandidates     int `json:"total_candidates"`
	WillModify          int `json:"will_modify"`
	PermissionRepairs   int `json:"permission_repairs"`
	ELFInterpreterFixes int `json:"elf_interpreter_fixes"`
	RPathFixes          int `json:"rpath_fixes"`
	WrapperLaunches     int `json:"wrapper_launches"`
}

type PatchPreviewEntry struct {
	Path         string           `json:"path"`
	RelativePath string           `json:"relative_path"`
	ModeBefore   string           `json:"mode_before,omitempty"`
	ModeAfter    string           `json:"mode_after,omitempty"`
	Action       string           `json:"action"`
	Reason       string           `json:"reason,omitempty"`
	WouldModify  bool             `json:"would_modify"`
	Permissions  PatchPermission  `json:"permissions"`
	ELF          *PatchPreviewELF `json:"elf,omitempty"`
}

type PatchPermission struct {
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

type PatchPreviewELF struct {
	Path              string   `json:"path"`
	Machine           string   `json:"machine"`
	FileType          string   `json:"file_type"`
	InterpreterBefore string   `json:"interpreter_before,omitempty"`
	InterpreterAfter  string   `json:"interpreter_after,omitempty"`
	RPathBefore       string   `json:"rpath_before,omitempty"`
	RPathAfter        string   `json:"rpath_after,omitempty"`
	RunPathBefore     string   `json:"runpath_before,omitempty"`
	RunPathAfter      string   `json:"runpath_after,omitempty"`
	ImportedLibraries []string `json:"imported_libraries,omitempty"`
	WouldUseWrapper   bool     `json:"would_use_wrapper"`
}

func (r PatchPreviewReport) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func (r PatchPreviewReport) BeginnerSummary() string {
	return fmt.Sprintf("%d candidates, %d changes", r.Summary.TotalCandidates, r.Summary.WillModify)
}

func (r PatchPreviewReport) ProfessionalDetails() []string {
	details := make([]string, 0, len(r.Entries))
	for _, entry := range r.Entries {
		details = append(details, fmt.Sprintf("%s: %s -> %s", entry.RelativePath, entry.ModeBefore, entry.ModeAfter))
		if entry.ELF != nil {
			details = append(details, fmt.Sprintf("%s: %s", entry.RelativePath, entry.Action))
		}
	}
	return details
}

func PreviewPatchTree(root string) (PatchPreviewReport, error) {
	runtimeDir, err := filepath.Abs(filepath.Join(root, aclDirName, aclRuntimeName))
	if err != nil {
		return PatchPreviewReport{}, err
	}
	return previewPatchTreeWithAnalyzer(root, runtimeDir, analyzeELFForPatch)
}

func previewPatchTreeWithAnalyzer(root, runtimeDir string, analyzer func(string) (elfAnalysis, error)) (PatchPreviewReport, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return PatchPreviewReport{}, err
	}
	report := PatchPreviewReport{Root: absRoot, RuntimeDir: runtimeDir}
	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if filepath.Clean(path) == filepath.Clean(runtimeDir) || filepath.Base(path) == ".acl" {
				return filepath.SkipDir
			}
			return nil
		}
		entry, ok, err := previewPatchEntry(absRoot, path, d, analyzer, runtimeDir)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		report.Entries = append(report.Entries, entry)
		if entry.WouldModify {
			report.Summary.WillModify++
			switch entry.Action {
			case string(patchActionWrapperLaunch):
				report.Summary.WrapperLaunches++
			case string(patchActionLoaderAndPath):
				report.Summary.ELFInterpreterFixes++
			case string(patchActionRunPathOnly):
				report.Summary.RPathFixes++
			default:
			}
			if entry.Permissions.Before != "" && entry.Permissions.After != "" && entry.Permissions.Before != entry.Permissions.After {
				report.Summary.PermissionRepairs++
			}
		}
		report.Summary.TotalCandidates++
		return nil
	})
	if err != nil {
		return PatchPreviewReport{}, err
	}

	sort.Slice(report.Entries, func(i, j int) bool {
		return report.Entries[i].RelativePath < report.Entries[j].RelativePath
	})
	return report, nil
}

func previewPatchEntry(root, path string, d fs.DirEntry, analyzer func(string) (elfAnalysis, error), runtimeDir string) (PatchPreviewEntry, bool, error) {
	info, err := d.Info()
	if err != nil {
		return PatchPreviewEntry{}, false, err
	}
	if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return PatchPreviewEntry{}, false, nil
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return PatchPreviewEntry{}, false, err
	}
	entry := PatchPreviewEntry{
		Path:         path,
		RelativePath: filepath.ToSlash(rel),
		ModeBefore:   info.Mode().String(),
		ModeAfter:    info.Mode().String(),
	}

	execByContent, execErr := looksExecutableByContent(path)
	if execErr != nil {
		return PatchPreviewEntry{}, false, execErr
	}

	if info.Mode()&0o111 == 0 && execByContent {
		entry.Action = "permission-repair"
		entry.Reason = "executable payload is missing execute bits"
		entry.WouldModify = true
		entry.Permissions.Before = info.Mode().String()
		entry.Permissions.After = (info.Mode() | 0o111).String()
		entry.ModeAfter = entry.Permissions.After
	}

	if !execByContent {
		return entry, entry.WouldModify, nil
	}

	isElf, err := isELF(path)
	if err != nil {
		return PatchPreviewEntry{}, false, err
	}
	if !isElf {
		if entry.WouldModify {
			return entry, true, nil
		}
		return PatchPreviewEntry{}, false, nil
	}

	analysis, err := analyzer(path)
	if err != nil {
		return PatchPreviewEntry{}, false, err
	}
	plan := planPatchForELF(analysis, runtimeDir)
	entry.ELF = &PatchPreviewELF{
		Path:              path,
		Machine:           analysis.Machine.String(),
		FileType:          analysis.FileType,
		InterpreterBefore: analysis.Interpreter,
		RPathBefore:       analysis.RPath,
		RunPathBefore:     analysis.RunPath,
		ImportedLibraries: append([]string(nil), analysis.ImportedLibraries...),
	}

	switch plan.Action {
	case patchActionNoChange, patchActionUnsupported:
		entry.Action = string(plan.Action)
		entry.Reason = plan.Reason
		entry.WouldModify = entry.WouldModify
	case patchActionBlocked:
		entry.Action = string(plan.Action)
		entry.Reason = plan.Reason
	case patchActionWrapperLaunch:
		entry.Action = string(plan.Action)
		entry.Reason = plan.Reason
		entry.WouldModify = true
		entry.ELF.WouldUseWrapper = true
	case patchActionLoaderAndPath:
		entry.Action = string(plan.Action)
		entry.Reason = plan.Reason
		entry.WouldModify = true
		entry.ELF.InterpreterAfter = plan.Spec.interpreter
		entry.ELF.RPathAfter = plan.Spec.rpath
		entry.ModeAfter = (info.Mode() | 0o111).String()
	case patchActionRunPathOnly:
		entry.Action = string(plan.Action)
		entry.Reason = plan.Reason
		entry.WouldModify = true
		entry.ELF.RunPathAfter = plan.Spec.rpath
		entry.ModeAfter = (info.Mode() | 0o111).String()
	default:
		entry.Action = string(plan.Action)
		entry.Reason = plan.Reason
	}

	if entry.Action == "" {
		entry.Action = "no-change"
	}
	if entry.ModeAfter == "" {
		entry.ModeAfter = entry.ModeBefore
	}
	if entry.Action != "no-change" {
		entry.WouldModify = true
	}
	return entry, true, nil
}
