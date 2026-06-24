// Package scanner provides ELF inspection, compatibility classification,
// and machine-readable report generation for the Android Compatibility Layer.
//
// The report pipeline is:
//
//	Inspect → Classify → Recommend → Marshal
//
// Each stage is independent and testable in isolation.
package scanner

import (
	"debug/elf"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ─── Schema version ──────────────────────────────────────────────────────────

// ReportSchemaVersion is the version of the JSON report schema produced by
// this package.  Consumers MUST reject reports whose schema_version they do
// not understand.
const ReportSchemaVersion = "1.0"

// ─── Enumerations ────────────────────────────────────────────────────────────

// CompatCategory mirrors the ACL compatibility classification system described
// in the project context document.
type CompatCategory string

const (
	CompatNativeAndroid CompatCategory = "native Android compatible"
	CompatLinuxGlibc    CompatCategory = "Linux/glibc executable"
	CompatStatic        CompatCategory = "static ELF"
	CompatScript        CompatCategory = "script"
	CompatUnknown       CompatCategory = "unknown"
	CompatUnsupported   CompatCategory = "unsupported"
)

// PatchAction is the concrete action the Patcher stage should apply.
type PatchAction string

const (
	// PatchNoAction — binary is already Android-compatible.
	PatchNoAction PatchAction = "no-action"
	// PatchRewriteInterpreter — PT_INTERP must be updated to the ACL loader path.
	PatchRewriteInterpreter PatchAction = "rewrite-interpreter"
	// PatchInjectRpath — RPATH/RUNPATH must be added or replaced.
	PatchInjectRpath PatchAction = "inject-rpath"
	// PatchRewriteInterpreterAndRpath — both interpreter and RPATH need updating.
	PatchRewriteInterpreterAndRpath PatchAction = "rewrite-interpreter-and-rpath"
	// PatchScriptNoop — script; ELF patching does not apply.
	PatchScriptNoop PatchAction = "script-no-elf-patch"
	// PatchUnsupported — binary cannot be patched for Android use.
	PatchUnsupported PatchAction = "unsupported"
)

// ─── Core report types ───────────────────────────────────────────────────────

// ELFInfo captures raw ELF metadata extracted from a binary.
// Fields are empty/nil when not present in the binary.
type ELFInfo struct {
	// Class is the ELF class: "ELF32", "ELF64", or "unknown".
	Class string `json:"class"`
	// Machine is the target ISA, e.g. "EM_AARCH64", "EM_386", "EM_X86_64".
	Machine string `json:"machine"`
	// Interpreter is the PT_INTERP value, i.e. the dynamic linker path embedded
	// in the binary.  Empty for static binaries and non-ELF files.
	Interpreter string `json:"interpreter,omitempty"`
	// Rpath is the DT_RPATH value (deprecated but still seen in older toolchains).
	Rpath string `json:"rpath,omitempty"`
	// Runpath is the DT_RUNPATH value.
	Runpath string `json:"runpath,omitempty"`
	// Needed is the list of DT_NEEDED shared-library names.
	Needed []string `json:"needed,omitempty"`
	// IsStatic is true when the binary has no PT_INTERP and no DT_NEEDED entries.
	IsStatic bool `json:"is_static"`
}

// MissingSymbol records a glibc symbol that is imported by the binary but is
// absent from Android's Bionic libc.
type MissingSymbol struct {
	// Name is the unmangled symbol name, e.g. "__cxa_thread_atexit_impl".
	Name string `json:"name"`
	// Library is the DT_NEEDED entry that is expected to provide the symbol.
	// Empty when the source library is unknown.
	Library string `json:"library,omitempty"`
	// Reason is a human-readable explanation, e.g. "glibc-only extension".
	Reason string `json:"reason,omitempty"`
}

// PatchRecommendation is the output of the Recommend stage: a single,
// concrete action for the Patcher to execute, together with the values it
// needs.
type PatchRecommendation struct {
	// Action is the patch action the Patcher should apply.
	Action PatchAction `json:"action"`
	// SuggestedInterpreter is the ACL runtime loader path that should replace
	// the current PT_INTERP value.  Set when Action contains interpreter rewrite.
	SuggestedInterpreter string `json:"suggested_interpreter,omitempty"`
	// SuggestedRpath is the colon-separated RPATH/RUNPATH that should be
	// injected.  Set when Action contains rpath injection.
	SuggestedRpath string `json:"suggested_rpath,omitempty"`
	// Rationale explains why this action was chosen.
	Rationale string `json:"rationale"`
}

// BinaryReport is the full compatibility report for a single file.
type BinaryReport struct {
	// Path is the absolute or relative path of the inspected file.
	Path string `json:"path"`
	// CompatCategory is the high-level classification.
	CompatCategory CompatCategory `json:"compat_category"`
	// ELF contains the raw ELF metadata (nil for scripts and unknowns).
	ELF *ELFInfo `json:"elf,omitempty"`
	// MissingSymbols lists glibc symbols that Bionic cannot satisfy.
	MissingSymbols []MissingSymbol `json:"missing_symbols,omitempty"`
	// Recommendation is the concrete patch action for this binary.
	Recommendation PatchRecommendation `json:"recommendation"`
	// Error holds an inspection error message when classification failed.
	Error string `json:"error,omitempty"`
}

// ScanReport is the top-level JSON document emitted by acl-scan.
type ScanReport struct {
	// SchemaVersion identifies the report schema.  Always ReportSchemaVersion.
	SchemaVersion string `json:"schema_version"`
	// GeneratedAt is the RFC 3339 timestamp when the report was produced.
	GeneratedAt string `json:"generated_at"`
	// Binaries contains one BinaryReport per inspected path.
	Binaries []BinaryReport `json:"binaries"`
	// Summary aggregates counts across all inspected binaries.
	Summary ScanSummary `json:"summary"`
}

// ScanSummary provides aggregate statistics over the full scan.
type ScanSummary struct {
	Total         int `json:"total"`
	NativeAndroid int `json:"native_android"`
	LinuxGlibc    int `json:"linux_glibc"`
	Static        int `json:"static"`
	Script        int `json:"script"`
	Unknown       int `json:"unknown"`
	Unsupported   int `json:"unsupported"`
	Errors        int `json:"errors"`
	// NeedsPatch is the count of binaries that require any ELF modification.
	NeedsPatch int `json:"needs_patch"`
}

// ─── Well-known path constants ───────────────────────────────────────────────

const (
	// termuxPrefix is the Termux installation prefix.  Paths that start with
	// this prefix indicate a Termux-origin binary that already targets Android.
	termuxPrefix = "/data/data/com.termux/files/usr"

	// androidLinkerAarch64 is the Bionic dynamic linker path on aarch64 devices.
	androidLinkerAarch64 = "/system/bin/linker64"
	// androidLinkerArm is the Bionic dynamic linker path on 32-bit ARM devices.
	androidLinkerArm = "/system/bin/linker"
	// androidLinkerX86_64 is the Bionic dynamic linker on x86_64 Android.
	androidLinkerX86_64 = "/system/bin/linker64"

	// aclDefaultRpath is the RPATH the ACL patcher injects when the binary
	// needs glibc-style libraries from the ACL runtime store.
	aclDefaultRpath = "/data/data/com.termux/files/usr/lib/acl-runtime/lib"

	// aclDefaultLoader is the loader the ACL patcher injects when a
	// Linux/glibc binary needs a compatible PT_INTERP.
	aclDefaultLoader = "/data/data/com.termux/files/usr/lib/acl-runtime/loader/ld-linux-aarch64.so.1"
)

// glibcOnlyLibraries is the set of DT_NEEDED library names that are specific
// to glibc and have no equivalent in Bionic.  A binary that imports one of
// these will require runtime wrapping or patching.
var glibcOnlyLibraries = map[string]string{
	"libpthread.so.0": "POSIX threads — folded into libc.so on Bionic/Android ≥ 5.0",
	"librt.so.1":      "POSIX realtime extensions — folded into libc.so on Bionic",
	"libdl.so.2":      "glibc dynamic linker interface — Bionic exposes equivalent via libc.so",
	"libresolv.so.2":  "GNU resolver — not present in Bionic; use getaddrinfo(3) instead",
	"libnsl.so.1":     "NIS/NIS+ — absent from Android; avoid or stub",
	"libm.so.6":       "glibc math — present in Bionic but as libm.so, not libm.so.6",
	"libc.so.6":       "glibc libc — Bionic provides libc.so, not libc.so.6",
	"libgcc_s.so.1":   "GCC runtime — not provided by Android NDK default paths",
}

// glibcInterpreterPrefixes are path prefixes that indicate a Linux/glibc
// dynamic linker embedded as PT_INTERP.
var glibcInterpreterPrefixes = []string{
	"/lib/ld-linux",
	"/lib64/ld-linux",
	"/lib/ld-musl",
	"/usr/lib/ld-linux",
}

// androidInterpreterPrefixes are path prefixes that identify Android/Bionic
// dynamic linkers.
var androidInterpreterPrefixes = []string{
	"/system/bin/linker",
	"/system/bin/linker64",
}

// ─── Inspector ───────────────────────────────────────────────────────────────

// InspectFile reads the ELF metadata from path.  For non-ELF files it returns
// nil, nil (the caller should classify by other means).  For genuine ELF
// parsing errors it returns nil, err.
func InspectFile(path string) (*ELFInfo, error) {
	f, err := elf.Open(path)
	if err != nil {
		// Not an ELF file — let the caller decide what to do.
		return nil, nil //nolint:nilerr // intentional: not-ELF is handled upstream
	}
	defer f.Close()

	info := &ELFInfo{}

	switch f.Class {
	case elf.ELFCLASS32:
		info.Class = "ELF32"
	case elf.ELFCLASS64:
		info.Class = "ELF64"
	default:
		info.Class = "unknown"
	}

	info.Machine = f.Machine.String()

	// PT_INTERP — the path to the dynamic linker.
	for _, prog := range f.Progs {
		if prog.Type == elf.PT_INTERP {
			buf := make([]byte, prog.Filesz)
			if _, rerr := prog.ReadAt(buf, 0); rerr == nil {
				// Strip the NUL terminator.
				info.Interpreter = strings.TrimRight(string(buf), "\x00")
			}
			break
		}
	}

	// Dynamic section entries: DT_NEEDED, DT_RPATH, DT_RUNPATH.
	libs, _ := f.ImportedLibraries()
	info.Needed = libs

	if dynStrings, derr := f.DynString(elf.DT_RPATH); derr == nil && len(dynStrings) > 0 {
		info.Rpath = dynStrings[0]
	}
	if dynStrings, derr := f.DynString(elf.DT_RUNPATH); derr == nil && len(dynStrings) > 0 {
		info.Runpath = dynStrings[0]
	}

	// A binary is considered static when it has no PT_INTERP and no DT_NEEDED.
	info.IsStatic = info.Interpreter == "" && len(info.Needed) == 0

	return info, nil
}

// ─── Classifier ──────────────────────────────────────────────────────────────

// ClassifyFile determines the CompatCategory for the file at path.
// It also returns any ELFInfo collected during inspection.
func ClassifyFile(path string) (CompatCategory, *ELFInfo, error) {
	// Check for Windows PE / other foreign formats first.
	if isWindowsBinary(path) {
		return CompatUnsupported, nil, nil
	}

	// Check for scripts (shebang).
	if isScript(path) {
		return CompatScript, nil, nil
	}

	info, err := InspectFile(path)
	if err != nil {
		return CompatUnknown, nil, fmt.Errorf("ELF parse error for %s: %w", path, err)
	}
	if info == nil {
		// Not an ELF and not a script — unknown.
		return CompatUnknown, nil, nil
	}

	if info.IsStatic {
		return CompatStatic, info, nil
	}

	// Termux-origin binaries target Android directly.
	if strings.HasPrefix(info.Interpreter, termuxPrefix) {
		return CompatNativeAndroid, info, nil
	}

	// Bionic dynamic linker — native Android binary.
	for _, pfx := range androidInterpreterPrefixes {
		if strings.HasPrefix(info.Interpreter, pfx) {
			return CompatNativeAndroid, info, nil
		}
	}

	// glibc dynamic linker — needs patching.
	for _, pfx := range glibcInterpreterPrefixes {
		if strings.HasPrefix(info.Interpreter, pfx) {
			return CompatLinuxGlibc, info, nil
		}
	}

	// No interpreter but has DT_NEEDED — treat as a shared library that is a
	// runtime dependency rather than a standalone executable.
	if info.Interpreter == "" && len(info.Needed) > 0 {
		return CompatLinuxGlibc, info, nil
	}

	return CompatUnknown, info, nil
}

// ─── Symbol inspector ────────────────────────────────────────────────────────

// FindMissingSymbols returns the glibc-only DT_NEEDED entries that Bionic
// cannot satisfy.  It does not perform deep symbol-level inspection (that
// requires a live linker); instead it uses the curated glibcOnlyLibraries map
// as a conservative first pass.
func FindMissingSymbols(info *ELFInfo) []MissingSymbol {
	if info == nil {
		return nil
	}
	var missing []MissingSymbol
	for _, lib := range info.Needed {
		if reason, ok := glibcOnlyLibraries[lib]; ok {
			missing = append(missing, MissingSymbol{
				Name:    lib,
				Library: lib,
				Reason:  reason,
			})
		}
	}
	return missing
}

// ─── Recommender ─────────────────────────────────────────────────────────────

// Recommend produces a concrete PatchRecommendation for the given binary.
func Recommend(cat CompatCategory, info *ELFInfo) PatchRecommendation {
	switch cat {
	case CompatNativeAndroid:
		return PatchRecommendation{
			Action:    PatchNoAction,
			Rationale: "Binary already targets Android/Bionic; no patching required.",
		}

	case CompatStatic:
		return PatchRecommendation{
			Action:    PatchNoAction,
			Rationale: "Statically linked binary; no dynamic linker or RPATH to patch.",
		}

	case CompatScript:
		return PatchRecommendation{
			Action:    PatchScriptNoop,
			Rationale: "Script file; ELF patching is not applicable.",
		}

	case CompatUnsupported:
		return PatchRecommendation{
			Action:    PatchUnsupported,
			Rationale: "Binary format not supported on Android (e.g. Windows PE); cannot patch.",
		}

	case CompatLinuxGlibc:
		if info == nil {
			return PatchRecommendation{
				Action:    PatchUnsupported,
				Rationale: "Linux/glibc classification but no ELF info available; cannot determine patch strategy.",
			}
		}
		needsInterp := isGlibcInterpreter(info.Interpreter)
		needsRpath := !hasAclRpath(info)

		switch {
		case needsInterp && needsRpath:
			return PatchRecommendation{
				Action:               PatchRewriteInterpreterAndRpath,
				SuggestedInterpreter: aclDefaultLoader,
				SuggestedRpath:       aclDefaultRpath,
				Rationale: fmt.Sprintf(
					"PT_INTERP %q is a glibc dynamic linker; must be replaced with ACL loader. "+
						"No ACL RPATH present; must inject %q so the ACL runtime libraries are found.",
					info.Interpreter, aclDefaultRpath,
				),
			}
		case needsInterp && !needsRpath:
			return PatchRecommendation{
				Action:               PatchRewriteInterpreter,
				SuggestedInterpreter: aclDefaultLoader,
				Rationale: fmt.Sprintf(
					"PT_INTERP %q is a glibc dynamic linker; must be replaced with ACL loader. "+
						"RPATH/RUNPATH already includes an ACL path.",
					info.Interpreter,
				),
			}
		case !needsInterp && needsRpath:
			return PatchRecommendation{
				Action:         PatchInjectRpath,
				SuggestedRpath: aclDefaultRpath,
				Rationale: fmt.Sprintf(
					"Interpreter is already Android-compatible but no ACL RPATH present; "+
						"must inject %q so the ACL runtime libraries are found.",
					aclDefaultRpath,
				),
			}
		default:
			// Classified as Linux/glibc but interpreter and rpath look OK —
			// conservative: still flag as no-action but explain.
			return PatchRecommendation{
				Action: PatchNoAction,
				Rationale: "Classified Linux/glibc but interpreter and RPATH already appear compatible; " +
					"manual verification recommended before marking as safe.",
			}
		}

	default: // CompatUnknown
		return PatchRecommendation{
			Action:    PatchUnsupported,
			Rationale: "Classification is unknown; manual inspection required before patching.",
		}
	}
}

// ─── Summary builder ─────────────────────────────────────────────────────────

// BuildSummary computes a ScanSummary from a slice of BinaryReports.
func BuildSummary(reports []BinaryReport) ScanSummary {
	s := ScanSummary{Total: len(reports)}
	for _, r := range reports {
		if r.Error != "" {
			s.Errors++
		}
		switch r.CompatCategory {
		case CompatNativeAndroid:
			s.NativeAndroid++
		case CompatLinuxGlibc:
			s.LinuxGlibc++
		case CompatStatic:
			s.Static++
		case CompatScript:
			s.Script++
		case CompatUnsupported:
			s.Unsupported++
		default:
			s.Unknown++
		}
		switch r.Recommendation.Action {
		case PatchRewriteInterpreter,
			PatchInjectRpath,
			PatchRewriteInterpreterAndRpath:
			s.NeedsPatch++
		}
	}
	return s
}

// ─── High-level entry point ──────────────────────────────────────────────────

// ScanPaths inspects each path and returns a populated ScanReport.
// Errors during individual file inspection are captured in BinaryReport.Error
// and do not abort the scan.
func ScanPaths(paths []string) ScanReport {
	reports := make([]BinaryReport, 0, len(paths))

	for _, p := range paths {
		br := scanOne(p)
		reports = append(reports, br)
	}

	return ScanReport{
		SchemaVersion: ReportSchemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Binaries:      reports,
		Summary:       BuildSummary(reports),
	}
}

func scanOne(path string) BinaryReport {
	cat, info, err := ClassifyFile(path)
	br := BinaryReport{
		Path:           path,
		CompatCategory: cat,
		ELF:            info,
	}
	if err != nil {
		br.Error = err.Error()
		br.Recommendation = PatchRecommendation{
			Action:    PatchUnsupported,
			Rationale: "Inspection failed; see error field.",
		}
		return br
	}

	br.MissingSymbols = FindMissingSymbols(info)
	br.Recommendation = Recommend(cat, info)
	return br
}

// ─── Marshalling helpers ──────────────────────────────────────────────────────

// MarshalReport serialises report to indented JSON.
func MarshalReport(report ScanReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func isWindowsBinary(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".exe" || ext == ".dll" || ext == ".sys" {
		return true
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	// Read the DOS MZ magic bytes.
	hdr := make([]byte, 2)
	if n, err := f.Read(hdr); n == 2 && err == nil {
		return hdr[0] == 0x4D && hdr[1] == 0x5A // "MZ"
	}
	return false
}

func isScript(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	hdr := make([]byte, 2)
	if n, err := f.Read(hdr); n == 2 && err == nil {
		return hdr[0] == '#' && hdr[1] == '!'
	}
	return false
}

func isGlibcInterpreter(interp string) bool {
	for _, pfx := range glibcInterpreterPrefixes {
		if strings.HasPrefix(interp, pfx) {
			return true
		}
	}
	return false
}

func hasAclRpath(info *ELFInfo) bool {
	combined := info.Rpath + ":" + info.Runpath
	return strings.Contains(combined, "acl-runtime")
}
