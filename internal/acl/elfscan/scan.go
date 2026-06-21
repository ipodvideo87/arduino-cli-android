package elfscan

import (
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var ErrNotELF = errors.New("not an ELF file")

type Inspection struct {
	Path                   string
	Exists                 bool
	IsELF                  bool
	Class                  string
	Machine                string
	FileType               string
	SONAME                 string
	Interpreter            string
	RPath                  string
	RunPath                string
	ImportedLibraries      []string
	Needed                 []string
	HardcodedAbsolutePaths []string
	LooksLikeLinuxTarget   bool
	LooksLikeRustLauncher  bool
	HasProgramInterpreter  bool
}

func Inspect(path string) (Inspection, error) {
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
