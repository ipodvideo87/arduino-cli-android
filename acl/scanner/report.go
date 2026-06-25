// Package scanner provides ELF and script compatibility scanning for the
// Android Compatibility Layer (ACL).
package scanner

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ─── Compatibility categories ────────────────────────────────────────────────

// CompatCategory classifies a binary or script by its Android compatibility.
type CompatCategory string

const (
	CategoryNativeAndroid CompatCategory = "native Android compatible"
	CategoryLinuxGlibc    CompatCategory = "Linux/glibc executable"
	CategoryStatic        CompatCategory = "static ELF"
	CategoryScript        CompatCategory = "script"
	CategoryUnknown       CompatCategory = "unknown"
	CategoryUnsupported   CompatCategory = "unsupported"
)

// PatchClass describes what kind of ELF patching (if any) is required.
type PatchClass string

const (
	PatchClassNone                 PatchClass = "none"
	PatchClassLoaderAndRpath       PatchClass = "loader-and-rpath"
	PatchClassRpathOnly            PatchClass = "rpath-only"
	PatchClassRuntimeDependency    PatchClass = "runtime-dependency-only"
	PatchClassScriptNoELFPatch     PatchClass = "script-no-elf-patch"
	PatchClassUnsupported          PatchClass = "unsupported"
)

// ─── JSON report schema ───────────────────────────────────────────────────────

// PatchAction is a single structured ELF-edit instruction.
type PatchAction struct {
	Action           string `json:"action"`
	Field            string `json:"field,omitempty"`
	CurrentValue     string `json:"current_value,omitempty"`
	RecommendedValue string `json:"recommended_value,omitempty"`
	Reason           string `json:"reason,omitempty"`
}

// ScriptInterpreterInfo holds the shebang validation result for a script entry.
type ScriptInterpreterInfo struct {
	// DeclaredPath is the interpreter path as written in the shebang line.
	DeclaredPath string `json:"declared_path"`

	// Args are any arguments following the interpreter path on the shebang line.
	Args []string `json:"args,omitempty"`

	// Status is one of "found", "missing", or "remapped".
	Status InterpreterStatus `json:"status"`

	// ResolvedPath is the effective interpreter path after resolution.
	// Empty when Status == "missing".
	ResolvedPath string `json:"resolved_path,omitempty"`

	// Recommendation is a human-readable description of what to do.
	Recommendation string `json:"recommendation"`
}

// ReportEntry is one item in the scan report (one binary or script).
type ReportEntry struct {
	// Path is the scanned file path.
	Path string `json:"path"`

	// Category is the compatibility classification.
	Category CompatCategory `json:"category"`

	// PatchClass is the patching classification (empty for scripts/unknown).
	PatchClass PatchClass `json:"patch_class,omitempty"`

	// Interpreter is the PT_INTERP value for ELF binaries (empty for scripts).
	Interpreter string `json:"interpreter,omitempty"`

	// Rpath is the RPATH/RUNPATH value for ELF binaries.
	Rpath string `json:"rpath,omitempty"`

	// Recommendation is the human-readable patch recommendation.
	Recommendation string `json:"recommendation"`

	// PatchActions is the structured list of ELF edits to apply.
	// Nil/empty for entries that need no patching.
	PatchActions []PatchAction `json:"patch_actions,omitempty"`

	// InterpreterStatus holds the shebang validation result for script entries.
	// Nil for non-script entries.
	InterpreterStatus *ScriptInterpreterInfo `json:"interpreter_status,omitempty"`
}

// ReportSummary holds pre-computed aggregate counters.
type ReportSummary struct {
	Total            int `json:"total"`
	NativeAndroid    int `json:"native_android"`
	LinuxGlibc       int `json:"linux_glibc"`
	Static           int `json:"static"`
	Script           int `json:"script"`
	Unknown          int `json:"unknown"`
	Unsupported      int `json:"unsupported"`
	Errors           int `json:"errors"`
	NeedsPatch       int `json:"needs_patch"`
	// Script-specific counters.
	ScriptFound      int `json:"script_interpreter_found,omitempty"`
	ScriptMissing    int `json:"script_interpreter_missing,omitempty"`
	ScriptRemapped   int `json:"script_interpreter_remapped,omitempty"`
}

// Report is the top-level JSON document emitted by the scanner.
type Report struct {
	SchemaVersion string        `json:"schema_version"`
	GeneratedAt   string        `json:"generated_at"`
	Target        string        `json:"target"`
	Summary       ReportSummary `json:"summary"`
	Entries       []ReportEntry `json:"entries"`
}

// ─── Builder ──────────────────────────────────────────────────────────────────

// ReportBuilder accumulates scan results and produces a Report.
type ReportBuilder struct {
	target     string
	prefixDir  string
	runtimeDir string
	entries    []ReportEntry
}

// NewReportBuilder creates a builder for a scan of target (file or directory).
// prefixDir is the Termux $PREFIX (may be empty to skip shebang resolution).
// runtimeDir is an optional ACL runtime directory.
func NewReportBuilder(target, prefixDir, runtimeDir string) *ReportBuilder {
	return &ReportBuilder{
		target:     target,
		prefixDir:  prefixDir,
		runtimeDir: runtimeDir,
	}
}

// AddELFEntry records a scanned ELF binary.
func (b *ReportBuilder) AddELFEntry(
	path string,
	category CompatCategory,
	patchClass PatchClass,
	interpreter, rpath string,
	recommendation string,
	actions []PatchAction,
) {
	b.entries = append(b.entries, ReportEntry{
		Path:           path,
		Category:       category,
		PatchClass:     patchClass,
		Interpreter:    interpreter,
		Rpath:          rpath,
		Recommendation: recommendation,
		PatchActions:   actions,
	})
}

// AddScriptEntry records a scanned script file.  It calls ScanShebang
// automatically to populate the InterpreterStatus field.
func (b *ReportBuilder) AddScriptEntry(path string) error {
	entry := ReportEntry{
		Path:           path,
		Category:       CategoryScript,
		PatchClass:     PatchClassScriptNoELFPatch,
		Recommendation: "Script file; ELF patching is not applicable.",
	}

	shebangResult, err := ScanShebang(path, b.prefixDir, b.runtimeDir)
	if err != nil {
		// Non-fatal: record the error in the recommendation but keep going.
		entry.Recommendation = fmt.Sprintf(
			"Script file; ELF patching is not applicable. "+
				"Shebang read error: %v", err)
	} else if shebangResult != nil {
		entry.InterpreterStatus = &ScriptInterpreterInfo{
			DeclaredPath:   shebangResult.InterpreterPath,
			Args:           shebangResult.InterpreterArgs,
			Status:         shebangResult.Status,
			ResolvedPath:   shebangResult.ResolvedPath,
			Recommendation: shebangResult.Recommendation,
		}
	}

	b.entries = append(b.entries, entry)
	return nil
}

// AddRawEntry records an entry with full control over all fields (used by
// the text-format scanner and legacy callers).
func (b *ReportBuilder) AddRawEntry(entry ReportEntry) {
	b.entries = append(b.entries, entry)
}

// Build finalises and returns the Report.
func (b *ReportBuilder) Build() Report {
	summary := computeSummary(b.entries)
	return Report{
		SchemaVersion: "1.0",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Target:        b.target,
		Summary:       summary,
		Entries:       b.entries,
	}
}

// computeSummary tallies all entries into a ReportSummary.
func computeSummary(entries []ReportEntry) ReportSummary {
	var s ReportSummary
	s.Total = len(entries)
	for _, e := range entries {
		switch e.Category {
		case CategoryNativeAndroid:
			s.NativeAndroid++
		case CategoryLinuxGlibc:
			s.LinuxGlibc++
			s.NeedsPatch++
		case CategoryStatic:
			s.Static++
		case CategoryScript:
			s.Script++
			if e.InterpreterStatus != nil {
				switch e.InterpreterStatus.Status {
				case InterpreterFound:
					s.ScriptFound++
				case InterpreterMissing:
					s.ScriptMissing++
				case InterpreterRemapped:
					s.ScriptRemapped++
				}
			}
		case CategoryUnknown:
			s.Unknown++
		case CategoryUnsupported:
			s.Unsupported++
		}
		if e.PatchClass != "" &&
			e.PatchClass != PatchClassNone &&
			e.PatchClass != PatchClassScriptNoELFPatch &&
			e.Category == CategoryLinuxGlibc {
			// NeedsPatch already counted above for glibc; skip double-count.
		}
	}
	return s
}

// ─── Output helpers ───────────────────────────────────────────────────────────

// WriteJSON serialises the report to w as indented JSON.
func WriteJSON(w io.Writer, r Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteText writes a human-readable text representation of the report to w.
func WriteText(w io.Writer, r Report) error {
	fmt.Fprintf(w, "ACL Compatibility Scan\n")
	fmt.Fprintf(w, "======================\n")
	fmt.Fprintf(w, "Target      : %s\n", r.Target)
	fmt.Fprintf(w, "Generated   : %s\n", r.GeneratedAt)
	fmt.Fprintf(w, "Schema      : %s\n\n", r.SchemaVersion)

	for _, e := range r.Entries {
		fmt.Fprintf(w, "  [%s] %s\n", e.Category, e.Path)
		if e.PatchClass != "" {
			fmt.Fprintf(w, "      patch_class : %s\n", e.PatchClass)
		}
		if e.Interpreter != "" {
			fmt.Fprintf(w, "      interpreter : %s\n", e.Interpreter)
		}
		if e.Rpath != "" {
			fmt.Fprintf(w, "      rpath       : %s\n", e.Rpath)
		}
		if e.Recommendation != "" {
			fmt.Fprintf(w, "      note        : %s\n", e.Recommendation)
		}
		if e.InterpreterStatus != nil {
			is := e.InterpreterStatus
			fmt.Fprintf(w, "      shebang     : %s\n", is.DeclaredPath)
			if len(is.Args) > 0 {
				fmt.Fprintf(w, "      shebang args: %s\n", strings.Join(is.Args, " "))
			}
			fmt.Fprintf(w, "      interp_status: %s\n", is.Status)
			if is.ResolvedPath != "" {
				fmt.Fprintf(w, "      resolved    : %s\n", is.ResolvedPath)
			}
			fmt.Fprintf(w, "      interp_note : %s\n", is.Recommendation)
		}
		for _, a := range e.PatchActions {
			fmt.Fprintf(w, "      action      : %s field=%s current=%q recommended=%q\n",
				a.Action, a.Field, a.CurrentValue, a.RecommendedValue)
			if a.Reason != "" {
				fmt.Fprintf(w, "                    reason: %s\n", a.Reason)
			}
		}
		fmt.Fprintln(w)
	}

	s := r.Summary
	fmt.Fprintf(w, "Summary\n")
	fmt.Fprintf(w, "-------\n")
	fmt.Fprintf(w, "  Total              : %d\n", s.Total)
	fmt.Fprintf(w, "  Native Android     : %d\n", s.NativeAndroid)
	fmt.Fprintf(w, "  Linux/glibc        : %d\n", s.LinuxGlibc)
	fmt.Fprintf(w, "  Static ELF         : %d\n", s.Static)
	fmt.Fprintf(w, "  Scripts            : %d\n", s.Script)
	if s.Script > 0 {
		fmt.Fprintf(w, "    interp found     : %d\n", s.ScriptFound)
		fmt.Fprintf(w, "    interp remapped  : %d\n", s.ScriptRemapped)
		fmt.Fprintf(w, "    interp missing   : %d\n", s.ScriptMissing)
	}
	fmt.Fprintf(w, "  Unknown            : %d\n", s.Unknown)
	fmt.Fprintf(w, "  Unsupported        : %d\n", s.Unsupported)
	fmt.Fprintf(w, "  Needs patch        : %d\n", s.NeedsPatch)
	return nil
}

// ─── File-based helpers ───────────────────────────────────────────────────────

// ScanFile inspects a single file and appends an entry to the builder.
// It uses heuristics: shebang → script; ELF magic → ELF; otherwise unknown.
// For richer ELF classification callers should use the ELF-specific paths
// and call AddELFEntry directly.
func (b *ReportBuilder) ScanFile(path string) error {
	// Read first bytes to detect file type.
	magic, err := readMagic(path, 4)
	if err != nil {
		b.entries = append(b.entries, ReportEntry{
			Path:           path,
			Category:       CategoryUnknown,
			Recommendation: fmt.Sprintf("Cannot read file: %v", err),
		})
		return nil
	}

	// Script: starts with "#!"
	if len(magic) >= 2 && magic[0] == '#' && magic[1] == '!' {
		return b.AddScriptEntry(path)
	}

	// ELF: starts with 0x7f 'E' 'L' 'F'
	if len(magic) == 4 && magic[0] == 0x7f && magic[1] == 'E' && magic[2] == 'L' && magic[3] == 'F' {
		// Minimal classification — callers that have full ELF metadata
		// should use AddELFEntry directly.
		b.entries = append(b.entries, ReportEntry{
			Path:           path,
			Category:       CategoryUnknown,
			PatchClass:     PatchClassNone,
			Recommendation: "ELF binary detected; full classification requires ELF inspection.",
		})
		return nil
	}

	// Unknown / data file.
	b.entries = append(b.entries, ReportEntry{
		Path:           path,
		Category:       CategoryUnknown,
		Recommendation: "File type not recognised; not an ELF binary or script.",
	})
	return nil
}

// ScanDirectory walks dir, calling ScanFile for each regular file.
func (b *ReportBuilder) ScanDirectory(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		return b.ScanFile(path)
	})
}

// readMagic reads up to n bytes from the start of path.
func readMagic(path string, n int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, n)
	got, err := f.Read(buf)
	if err != nil && got == 0 {
		return nil, err
	}
	return buf[:got], nil
}
