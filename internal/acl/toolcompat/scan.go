package toolcompat

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	aclscan "github.com/arduino/arduino-cli/internal/acl/elfscan"
)

const (
	CategoryAndroidCompatible = "native-android-compatible"
	CategoryLinuxGlibc        = "linux-glibc-executable"
	CategoryStaticELF         = "static-elf"
	CategoryScript            = "script"
	CategoryUnknown           = "unknown"
	CategoryUnsupported       = "unsupported"

	PatchClassNone              = "none"
	PatchClassLoaderAndRPath    = "loader-and-rpath"
	PatchClassRPathOnly         = "rpath-only"
	PatchClassRuntimeDependency = "runtime-dependency-only"
	PatchClassScript            = "script-no-elf-patch"
	PatchClassUnsupported       = "unsupported"
)

type Report struct {
	Root    string                 `json:"root"`
	Entries []Entry                `json:"entries"`
	Summary Summary                `json:"summary"`
	Notes   []string               `json:"notes,omitempty"`
	Extras  map[string]interface{} `json:"extras,omitempty"`
}

type Summary struct {
	TotalByCategory map[string]int `json:"total_by_category"`
	TotalByType     map[string]int `json:"total_by_type"`
	TotalEntries    int            `json:"total_entries"`
}

type Entry struct {
	Path                   string   `json:"path"`
	RelativePath           string   `json:"relative_path"`
	ExecutableType         string   `json:"executable_type"`
	CompatibilityCategory  string   `json:"compatibility_category"`
	PatchClass             string   `json:"patch_class"`
	Architecture           string   `json:"architecture,omitempty"`
	Interpreter            string   `json:"interpreter,omitempty"`
	SharedLibraries        []string `json:"shared_libraries,omitempty"`
	RPath                  string   `json:"rpath,omitempty"`
	RunPath                string   `json:"runpath,omitempty"`
	HardcodedAbsolutePath  []string `json:"hardcoded_absolute_paths,omitempty"`
	LooksAndroidCompatible bool     `json:"looks_android_compatible"`
	LooksLinuxGlibc        bool     `json:"looks_linux_glibc"`
	RequiresRuntime        bool     `json:"requires_runtime"`
	Notes                  []string `json:"notes,omitempty"`
}

type Scanner struct {
	inspectELF func(string) (aclscan.Inspection, error)
}

func NewScanner() *Scanner {
	return &Scanner{inspectELF: aclscan.Inspect}
}

func (s *Scanner) Scan(root string) (Report, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return Report{}, errors.New("missing scan root")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Report{}, fmt.Errorf("resolve scan root %q: %w", root, err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return Report{}, fmt.Errorf("scan root %q: %w", absRoot, err)
	}
	if !info.IsDir() {
		return Report{}, fmt.Errorf("scan root %q is not a directory", absRoot)
	}

	report := Report{
		Root: absRoot,
		Summary: Summary{
			TotalByCategory: map[string]int{},
			TotalByType:     map[string]int{},
		},
	}

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		entry, ok, err := s.inspectPath(absRoot, path, d)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		report.Entries = append(report.Entries, entry)
		report.Summary.TotalEntries++
		report.Summary.TotalByCategory[entry.CompatibilityCategory]++
		report.Summary.TotalByType[entry.ExecutableType]++
		return nil
	})
	if err != nil {
		return Report{}, err
	}

	sort.Slice(report.Entries, func(i, j int) bool {
		return report.Entries[i].RelativePath < report.Entries[j].RelativePath
	})
	return report, nil
}

func (s *Scanner) inspectPath(root, path string, d fs.DirEntry) (Entry, bool, error) {
	info, err := d.Info()
	if err != nil {
		return Entry{}, false, err
	}
	if !(info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0) {
		return Entry{}, false, nil
	}

	kind, ok, err := detectCandidateType(path, info.Mode())
	if err != nil {
		return Entry{}, false, err
	}
	if !ok {
		return Entry{}, false, nil
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return Entry{}, false, err
	}
	entry := Entry{
		Path:         path,
		RelativePath: filepath.ToSlash(rel),
	}

	switch kind {
	case "elf":
		inspection, err := s.inspectELF(path)
		if err != nil {
			return Entry{}, false, err
		}
		entry.ExecutableType = "elf"
		entry.Architecture = inspection.Machine
		entry.Interpreter = inspection.Interpreter
		entry.SharedLibraries = append([]string(nil), inspection.ImportedLibraries...)
		entry.RPath = inspection.RPath
		entry.RunPath = inspection.RunPath
		entry.HardcodedAbsolutePath = append([]string(nil), inspection.HardcodedAbsolutePaths...)
		entry.LooksLinuxGlibc = inspection.LooksLikeLinuxTarget
		entry.LooksAndroidCompatible = looksAndroidCompatible(inspection)
		entry.RequiresRuntime = entry.LooksLinuxGlibc && !entry.LooksAndroidCompatible
		entry.CompatibilityCategory = classifyELF(entry)
		entry.PatchClass = PatchClassForELFInspection(inspection)
		if inspection.Interpreter == "" {
			entry.Notes = append(entry.Notes, "ELF has no PT_INTERP entry")
		}
	case "shell":
		entry.ExecutableType = "shell-script"
		entry.Interpreter = readShebang(path)
		entry.CompatibilityCategory = CategoryScript
		entry.PatchClass = PatchClassScript
		entry.Notes = append(entry.Notes, "script execution should be evaluated separately from ACL runtime execution")
	case "python":
		entry.ExecutableType = "python-script"
		entry.Interpreter = readShebang(path)
		entry.CompatibilityCategory = CategoryScript
		entry.PatchClass = PatchClassScript
		entry.Notes = append(entry.Notes, "Python script compatibility depends on a suitable Python host")
	case "perl":
		entry.ExecutableType = "perl-script"
		entry.Interpreter = readShebang(path)
		entry.CompatibilityCategory = CategoryScript
		entry.PatchClass = PatchClassScript
	case "java":
		entry.ExecutableType = "java-archive"
		entry.CompatibilityCategory = CategoryUnknown
		entry.PatchClass = PatchClassUnsupported
		entry.Notes = append(entry.Notes, "Java archives require a suitable Java runtime and launch path")
	default:
		entry.ExecutableType = kind
		entry.CompatibilityCategory = CategoryUnsupported
		entry.PatchClass = PatchClassUnsupported
		entry.Notes = append(entry.Notes, "unrecognized executable type")
	}

	return entry, true, nil
}

func detectCandidateType(path string, mode fs.FileMode) (string, bool, error) {
	if mode&os.ModeSymlink != 0 {
		resolved, err := os.Stat(path)
		if err != nil {
			return "", false, nil
		}
		mode = resolved.Mode()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}

	isExecutableBit := mode&0o111 != 0
	trimmedName := strings.ToLower(filepath.Base(path))

	if len(data) >= 4 && string(data[:4]) == "\x7fELF" {
		return "elf", isExecutableBit, nil
	}
	if shebang := parseShebang(data); shebang != "" {
		switch {
		case strings.Contains(shebang, "python"):
			return "python", true, nil
		case strings.Contains(shebang, "sh"), strings.Contains(shebang, "bash"):
			return "shell", true, nil
		case strings.Contains(shebang, "perl"):
			return "perl", true, nil
		default:
			return "script", true, nil
		}
	}
	switch {
	case strings.HasSuffix(trimmedName, ".jar"):
		if _, err := zip.OpenReader(path); err == nil {
			return "java", true, nil
		}
	case strings.HasSuffix(trimmedName, ".py"):
		return "python", isExecutableBit, nil
	case strings.HasSuffix(trimmedName, ".sh"):
		return "shell", isExecutableBit, nil
	case strings.HasSuffix(trimmedName, ".pl"):
		return "perl", isExecutableBit, nil
	}
	if isExecutableBit {
		return "binary", true, nil
	}
	return "", false, nil
}

func parseShebang(data []byte) string {
	if len(data) < 2 || string(data[:2]) != "#!" {
		return ""
	}
	line := string(data)
	if idx := strings.IndexByte(line, '\n'); idx >= 0 {
		line = line[:idx]
	}
	return strings.TrimSpace(strings.TrimPrefix(line, "#!"))
}

func readShebang(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return parseShebang(data)
}

func looksAndroidCompatible(inspection aclscan.Inspection) bool {
	interp := strings.ToLower(strings.TrimSpace(inspection.Interpreter))
	if strings.Contains(interp, "/system/bin/linker") || strings.Contains(interp, "/apex/com.android.runtime") {
		return true
	}
	for _, lib := range inspection.ImportedLibraries {
		switch lib {
		case "libc.so", "libdl.so", "libm.so", "liblog.so", "libandroid.so":
			return true
		}
	}
	for _, path := range inspection.HardcodedAbsolutePaths {
		if strings.Contains(path, "/system/bin") || strings.Contains(path, "/apex/com.android.runtime") {
			return true
		}
	}
	return false
}

func classifyELF(entry Entry) string {
	switch {
	case entry.LooksAndroidCompatible:
		return CategoryAndroidCompatible
	case entry.LooksLinuxGlibc:
		return CategoryLinuxGlibc
	case isStaticELF(entry):
		return CategoryStaticELF
	default:
		return CategoryUnknown
	}
}

func isStaticELF(entry Entry) bool {
	return strings.EqualFold(strings.TrimSpace(entry.ExecutableType), "elf") &&
		strings.TrimSpace(entry.Interpreter) == "" &&
		len(entry.SharedLibraries) == 0
}

func PatchClassForELFInspection(inspection aclscan.Inspection) string {
	if looksAndroidCompatible(inspection) {
		return PatchClassNone
	}
	if !inspection.LooksLikeLinuxTarget {
		return PatchClassUnsupported
	}
	return PatchClassForELFFields(inspection.FileType, inspection.Interpreter, inspection.ImportedLibraries)
}

func PatchClassForELFFields(fileType string, interpreter string, importedLibraries []string) string {
	if interpreter != "" {
		return PatchClassLoaderAndRPath
	}
	if fileType == "DYN" {
		for _, lib := range importedLibraries {
			if looksLikeGlibcLibrary(lib) {
				return PatchClassRPathOnly
			}
		}
	}
	return PatchClassRuntimeDependency
}

func looksLikeGlibcLibrary(lib string) bool {
	switch lib {
	case "libc.so.6", "libpthread.so.0", "libdl.so.2", "librt.so.1", "libm.so.6", "libstdc++.so.6", "libgcc_s.so.1", "libz.so.1":
		return true
	default:
		return false
	}
}

func DefaultPackagesRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".arduino15", "packages"), nil
}

func (r Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func FormatReport(report Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ACL Tool Compatibility Report\n")
	fmt.Fprintf(&b, "-----------------------------\n")
	fmt.Fprintf(&b, "Root: %s\n", report.Root)
	fmt.Fprintf(&b, "Entries: %d\n", report.Summary.TotalEntries)

	if len(report.Summary.TotalByCategory) > 0 {
		categories := sortedKeys(report.Summary.TotalByCategory)
		fmt.Fprintln(&b, "Categories:")
		for _, key := range categories {
			fmt.Fprintf(&b, "  - %s: %d\n", key, report.Summary.TotalByCategory[key])
		}
	}

	if len(report.Entries) == 0 {
		fmt.Fprintln(&b, "No executable tool candidates found.")
		return b.String()
	}

	for _, entry := range report.Entries {
		fmt.Fprintf(&b, "\n- %s\n", entry.RelativePath)
		fmt.Fprintf(&b, "  type: %s\n", entry.ExecutableType)
		fmt.Fprintf(&b, "  compatibility: %s\n", entry.CompatibilityCategory)
		fmt.Fprintf(&b, "  patch class: %s\n", entry.PatchClass)
		if entry.Architecture != "" {
			fmt.Fprintf(&b, "  architecture: %s\n", entry.Architecture)
		}
		if entry.Interpreter != "" {
			fmt.Fprintf(&b, "  interpreter: %s\n", entry.Interpreter)
		}
		if entry.RPath != "" {
			fmt.Fprintf(&b, "  rpath: %s\n", entry.RPath)
		}
		if entry.RunPath != "" {
			fmt.Fprintf(&b, "  runpath: %s\n", entry.RunPath)
		}
		if len(entry.SharedLibraries) > 0 {
			fmt.Fprintf(&b, "  shared libraries: %s\n", strings.Join(entry.SharedLibraries, ", "))
		}
		if len(entry.HardcodedAbsolutePath) > 0 {
			fmt.Fprintf(&b, "  hardcoded paths: %s\n", strings.Join(entry.HardcodedAbsolutePath, ", "))
		}
		if entry.RequiresRuntime {
			fmt.Fprintf(&b, "  requires runtime: yes\n")
		}
		for _, note := range entry.Notes {
			fmt.Fprintf(&b, "  note: %s\n", note)
		}
	}
	return b.String()
}

func sortedKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
