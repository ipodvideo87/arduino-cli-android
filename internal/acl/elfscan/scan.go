package elfscan

import (
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var ErrNotELF = errors.New("not an ELF file")

type Inspection struct {
	Path                     string
	Exists                   bool
	IsELF                    bool
	Class                    string
	Machine                  string
	FileType                 string
	SONAME                   string
	Interpreter              string
	RPath                    string
	RunPath                  string
	ImportedLibraries        []string
	Needed                   []string
	HardcodedAbsolutePaths   []string
	LauncherDelegateTargets  []LauncherDelegateTarget
	LooksLikeLinuxTarget     bool
	LooksLikeRustLauncher    bool
	LooksLikeXtensaDynConfig bool
	HasProgramInterpreter    bool
}

type LauncherDelegateTarget struct {
	Path                   string   `json:"path"`
	Exists                 bool     `json:"exists"`
	Executable             bool     `json:"executable"`
	Mode                   string   `json:"mode,omitempty"`
	Symlink                bool     `json:"symlink"`
	SymlinkTarget          string   `json:"symlink_target,omitempty"`
	Source                 string   `json:"source,omitempty"`
	InspectionError        string   `json:"inspection_error,omitempty"`
	IsELF                  bool     `json:"is_elf"`
	Class                  string   `json:"class,omitempty"`
	Machine                string   `json:"machine,omitempty"`
	FileType               string   `json:"file_type,omitempty"`
	Interpreter            string   `json:"interpreter,omitempty"`
	RPath                  string   `json:"rpath,omitempty"`
	RunPath                string   `json:"runpath,omitempty"`
	ImportedLibraries      []string `json:"imported_libraries,omitempty"`
	Needed                 []string `json:"needed,omitempty"`
	HardcodedAbsolutePaths []string `json:"hardcoded_absolute_paths,omitempty"`
	LooksLikeLinuxTarget   bool     `json:"looks_like_linux_target"`
	LooksLikeRustLauncher  bool     `json:"looks_like_rust_launcher"`
	HasProgramInterpreter  bool     `json:"has_program_interpreter"`
}

func Inspect(path string) (Inspection, error) {
	return inspect(path, true)
}

func inspect(path string, detectDelegates bool) (Inspection, error) {
	result := Inspection{Path: path}

	info, err := os.Stat(path)
	if err != nil {
		return result, err
	}
	result.Exists = info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0

	f, err := elf.Open(path)
	if err != nil {
		return result, fmt.Errorf("%w: %v", ErrNotELF, err)
	}
	defer f.Close()

	result.IsELF = true
	result.Class = f.FileHeader.Class.String()
	result.Machine = f.FileHeader.Machine.String()
	result.FileType = f.FileHeader.Type.String()
	result.SONAME = readDynamicString(f, elf.DT_SONAME)
	result.Interpreter = readInterpreter(f)
	result.RPath = readDynamicString(f, elf.DT_RPATH)
	result.RunPath = readDynamicString(f, elf.DT_RUNPATH)
	result.HasProgramInterpreter = result.Interpreter != ""

	imports, err := f.ImportedLibraries()
	if err != nil {
		return result, fmt.Errorf("failed to read imported libraries: %w", err)
	}
	result.ImportedLibraries = imports
	result.Needed = readDynamicStrings(f, elf.DT_NEEDED)
	result.HardcodedAbsolutePaths = findInterestingAbsoluteStrings(path)
	result.LooksLikeLinuxTarget = looksLikeLinuxTarget(result.SONAME, result.Interpreter, result.ImportedLibraries)
	result.LooksLikeRustLauncher = looksLikeRustLauncher(path)
	result.LooksLikeXtensaDynConfig = looksLikeXtensaDynConfig(path)
	if detectDelegates && result.LooksLikeRustLauncher {
		result.LauncherDelegateTargets = findRustLauncherDelegateTargets(path)
	}

	return result, nil
}

func readInterpreter(f *elf.File) string {
	for _, prog := range f.Progs {
		if prog.Type != elf.PT_INTERP {
			continue
		}

		r := prog.Open()
		data, err := io.ReadAll(r)
		if err != nil {
			return ""
		}
		return strings.TrimRight(string(data), "\x00")
	}
	return ""
}

func readDynamicString(f *elf.File, tag elf.DynTag) string {
	values, err := f.DynString(tag)
	if err != nil || len(values) == 0 {
		return ""
	}
	return strings.Join(values, ":")
}

func readDynamicStrings(f *elf.File, tag elf.DynTag) []string {
	values, err := f.DynString(tag)
	if err != nil {
		return nil
	}
	return values
}

func findInterestingAbsoluteStrings(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	seen := map[string]struct{}{}
	var results []string
	for _, s := range extractStrings(data) {
		if !strings.HasPrefix(s, "/") {
			continue
		}
		if !strings.Contains(s, "/") {
			continue
		}
		if !isInterestingPath(s) {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		results = append(results, s)
	}
	return results
}

func extractStrings(data []byte) []string {
	const minLen = 8

	var out []string
	start := -1
	for i, b := range data {
		if b >= 0x20 && b <= 0x7e {
			if start == -1 {
				start = i
			}
			continue
		}
		if start != -1 && i-start >= minLen {
			out = append(out, string(data[start:i]))
		}
		start = -1
	}
	if start != -1 && len(data)-start >= minLen {
		out = append(out, string(data[start:]))
	}
	return out
}

func isInterestingPath(path string) bool {
	markers := []string{
		"/data/data/com.termux/files/usr/glibc",
		"/data/data/com.termux/files/usr",
		"/system/bin",
		"/vendor",
		"/usr/",
	}
	for _, marker := range markers {
		if strings.Contains(path, marker) {
			return true
		}
	}
	return false
}

func looksLikeLinuxTarget(soname, interpreter string, imports []string) bool {
	if strings.Contains(soname, "ld-linux") || strings.Contains(interpreter, "ld-linux") || strings.Contains(interpreter, "linux") {
		return true
	}

	markers := map[string]struct{}{
		"libc.so.6":       {},
		"libdl.so.2":      {},
		"libm.so.6":       {},
		"libpthread.so.0": {},
		"librt.so.1":      {},
	}

	for _, lib := range imports {
		if _, ok := markers[lib]; ok {
			return true
		}
	}

	return false
}

func looksLikeRustLauncher(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var (
		foundPattern   bool
		foundDynconfig bool
		foundExecPath  bool
	)
	for _, s := range extractStrings(data) {
		lower := strings.ToLower(s)
		switch {
		case strings.Contains(s, "XTENSA_GNU_CONFIG"):
			foundDynconfig = true
		case strings.Contains(lower, "called tool must have pattern"):
			foundPattern = true
		case strings.Contains(lower, "dynconfig for target"):
			foundDynconfig = true
		case strings.Contains(lower, "get executable path"):
			foundExecPath = true
		case strings.Contains(lower, "current exe has path"):
			foundExecPath = true
		case strings.Contains(lower, "execv errno"):
			foundExecPath = true
		}
	}

	return foundPattern && foundDynconfig && foundExecPath
}

func looksLikeXtensaDynConfig(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var foundDynConfig bool
	var foundPluginHint bool
	for _, s := range extractStrings(data) {
		lower := strings.ToLower(s)
		switch {
		case strings.Contains(s, "XTENSA_GNU_CONFIG"):
			foundDynConfig = true
		case strings.Contains(lower, "-mdynconfig="):
			foundDynConfig = true
		case strings.Contains(lower, "dynconfig for target"):
			foundDynConfig = true
		case strings.Contains(lower, "xtensa_.so"):
			foundPluginHint = true
		}
	}

	return foundDynConfig && foundPluginHint
}

func findRustLauncherDelegateTargets(path string) []LauncherDelegateTarget {
	seen := map[string]struct{}{}
	var targets []LauncherDelegateTarget

	addCandidate := func(candidate, source string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		candidate = filepath.Clean(candidate)
		if candidate == "." || candidate == string(filepath.Separator) {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		targets = append(targets, inspectLauncherDelegateTarget(candidate, source))
	}

	if linkTarget, err := os.Readlink(path); err == nil {
		if !filepath.IsAbs(linkTarget) {
			linkTarget = filepath.Join(filepath.Dir(path), linkTarget)
		}
		addCandidate(linkTarget, "symlink-target")
	}

	if data, err := os.ReadFile(path); err == nil {
		for _, s := range extractStrings(data) {
			if candidate := normalizeLauncherDelegateCandidate(s); candidate != "" {
				addCandidate(candidate, "embedded-path")
			}
		}
	}

	for _, candidate := range rustLauncherDelegateCandidates(path) {
		addCandidate(candidate, "chip-plugin")
	}

	return targets
}

func normalizeLauncherDelegateCandidate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !strings.HasPrefix(value, "/") {
		return ""
	}

	lower := strings.ToLower(value)
	switch {
	case strings.HasSuffix(lower, ".a"),
		strings.HasSuffix(lower, ".h"),
		strings.HasSuffix(lower, ".md"),
		strings.HasSuffix(lower, ".txt"),
		strings.HasSuffix(lower, ".json"),
		strings.HasSuffix(lower, ".csv"),
		strings.HasSuffix(lower, ".ini"):
		return ""
	}

	base := filepath.Base(lower)
	if strings.Contains(lower, "/bin/") || strings.Contains(lower, "/tools/") || strings.Contains(lower, "/libexec/") {
		return filepath.Clean(value)
	}

	for _, marker := range []string{"gcc", "g++", "clang", "rust", "python", "perl", "xtensa", "launcher", "exec"} {
		if strings.Contains(base, marker) {
			return filepath.Clean(value)
		}
	}

	return ""
}

func rustLauncherDelegateCandidates(path string) []string {
	base := strings.ToLower(filepath.Base(path))
	chips := rustLauncherChipCandidates(base)
	if len(chips) == 0 {
		return nil
	}

	packageRoot := filepath.Clean(filepath.Join(filepath.Dir(path), ".."))
	libDir := filepath.Join(packageRoot, "lib")
	candidates := make([]string, 0, len(chips))
	for _, chip := range chips {
		candidates = append(candidates, filepath.Join(libDir, "xtensa_"+chip+".so"))
	}
	return candidates
}

func rustLauncherChipCandidates(base string) []string {
	base = strings.ToLower(base)
	switch {
	case strings.Contains(base, "esp32s3"):
		return []string{"esp32s3", "esp32"}
	case strings.Contains(base, "esp32s2"):
		return []string{"esp32s2", "esp32"}
	case strings.Contains(base, "esp32"):
		return []string{"esp32"}
	case strings.Contains(base, "esp8266"), strings.Contains(base, "lx106"):
		return []string{"esp8266"}
	default:
		return nil
	}
}

func inspectLauncherDelegateTarget(path, source string) LauncherDelegateTarget {
	target := LauncherDelegateTarget{
		Path:   path,
		Source: source,
	}

	if linkInfo, err := os.Lstat(path); err == nil {
		target.Mode = linkInfo.Mode().String()
		target.Symlink = linkInfo.Mode()&os.ModeSymlink != 0
		if target.Symlink {
			if linkTarget, err := os.Readlink(path); err == nil {
				target.SymlinkTarget = linkTarget
			}
		}
	}

	if info, err := os.Stat(path); err == nil {
		target.Exists = true
		target.Executable = info.Mode()&0o111 != 0
		if target.Mode == "" {
			target.Mode = info.Mode().String()
		}
	}

	inspection, err := inspect(path, false)
	if err != nil {
		target.InspectionError = err.Error()
		return target
	}
	target.IsELF = inspection.IsELF
	target.Class = inspection.Class
	target.Machine = inspection.Machine
	target.FileType = inspection.FileType
	target.Interpreter = inspection.Interpreter
	target.RPath = inspection.RPath
	target.RunPath = inspection.RunPath
	target.ImportedLibraries = append([]string(nil), inspection.ImportedLibraries...)
	target.Needed = append([]string(nil), inspection.Needed...)
	target.HardcodedAbsolutePaths = append([]string(nil), inspection.HardcodedAbsolutePaths...)
	target.LooksLikeLinuxTarget = inspection.LooksLikeLinuxTarget
	target.LooksLikeRustLauncher = inspection.LooksLikeRustLauncher
	target.HasProgramInterpreter = inspection.HasProgramInterpreter

	return target
}

func Format(ins Inspection) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ACL ELF scanner\n")
	fmt.Fprintf(&b, "--------------\n")
	fmt.Fprintf(&b, "File: %s\n", ins.Path)
	fmt.Fprintf(&b, "ELF: yes\n")
	fmt.Fprintf(&b, "Class: %s\n", ins.Class)
	fmt.Fprintf(&b, "Machine: %s\n", ins.Machine)
	fmt.Fprintf(&b, "Type: %s\n", ins.FileType)
	if ins.SONAME != "" {
		fmt.Fprintf(&b, "SONAME: %s\n", ins.SONAME)
	}
	if ins.Interpreter != "" {
		fmt.Fprintf(&b, "PT_INTERP: yes\n")
		fmt.Fprintf(&b, "Interpreter: %s\n", ins.Interpreter)
	} else {
		fmt.Fprintf(&b, "PT_INTERP: no\n")
	}
	if ins.RPath != "" {
		fmt.Fprintf(&b, "RPATH: %s\n", ins.RPath)
	}
	if ins.RunPath != "" {
		fmt.Fprintf(&b, "RUNPATH: %s\n", ins.RunPath)
	}
	if len(ins.Needed) > 0 {
		fmt.Fprintln(&b, "NEEDED libraries:")
		for _, lib := range ins.Needed {
			fmt.Fprintf(&b, "  - %s\n", lib)
		}
	}
	if len(ins.HardcodedAbsolutePaths) > 0 {
		fmt.Fprintln(&b, "Hardcoded absolute paths:")
		for _, path := range ins.HardcodedAbsolutePaths {
			fmt.Fprintf(&b, "  - %s\n", path)
		}
	}
	if len(ins.LauncherDelegateTargets) > 0 {
		fmt.Fprintln(&b, "Launcher delegate targets:")
		for _, target := range ins.LauncherDelegateTargets {
			fmt.Fprintf(&b, "  - %s", target.Path)
			if target.Source != "" {
				fmt.Fprintf(&b, " (%s)", target.Source)
			}
			if target.Symlink {
				fmt.Fprintf(&b, " symlink->%s", target.SymlinkTarget)
			}
			fmt.Fprintf(&b, " exists=%t executable=%t", target.Exists, target.Executable)
			if target.Mode != "" {
				fmt.Fprintf(&b, " mode=%s", target.Mode)
			}
			if target.InspectionError != "" {
				fmt.Fprintf(&b, " inspect_error=%s", target.InspectionError)
			} else {
				fmt.Fprintf(&b, " elf=%t class=%s machine=%s type=%s", target.IsELF, target.Class, target.Machine, target.FileType)
				if target.Interpreter != "" {
					fmt.Fprintf(&b, " interp=%s", target.Interpreter)
				}
				if target.RPath != "" {
					fmt.Fprintf(&b, " rpath=%s", target.RPath)
				}
				if target.RunPath != "" {
					fmt.Fprintf(&b, " runpath=%s", target.RunPath)
				}
				if len(target.Needed) > 0 {
					fmt.Fprintf(&b, " needed=%s", strings.Join(target.Needed, ","))
				}
			}
			fmt.Fprintln(&b)
		}
	}
	fmt.Fprintf(&b, "Looks like glibc/Linux-targeted: %t\n", ins.LooksLikeLinuxTarget)
	fmt.Fprintln(&b, "Imported libraries:")
	if len(ins.ImportedLibraries) == 0 {
		fmt.Fprintln(&b, "  (none)")
		return b.String()
	}
	for _, lib := range ins.ImportedLibraries {
		fmt.Fprintf(&b, "  - %s\n", lib)
	}
	return b.String()
}
