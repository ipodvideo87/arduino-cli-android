package android

import (
	"debug/elf"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/arduino/arduino-cli/internal/acl/toolcompat"
)

type patchSpec struct {
	setInterpreter bool
	interpreter    string
	rpath          string
}

// patchELFs walks an installed tree looking for host executables that ACL may
// need to rewrite for Android compatibility.
func patchELFs(root, runtimeDir string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}
		if info.IsDir() {
			if path == runtimeDir {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Mode()&0o111 == 0 {
			return nil
		}
		return patchExecutable(path, runtimeDir)
	})
}

// patchExecutable patches a single executable if needed.
func patchExecutable(path, runtimeDir string) error {
	isElf, err := isELF(path)
	if err != nil {
		return fmt.Errorf("detect ELF %s: %w", path, err)
	}
	if !isElf {
		return nil
	}

	f, err := elf.Open(path)
	if err != nil {
		return fmt.Errorf("open ELF %s: %w", path, err)
	}
	defer f.Close()

	spec, shouldPatch := patchSpecForELF(f, runtimeDir)
	if !shouldPatch {
		return nil
	}

	libs, err := f.ImportedLibraries()
	if err != nil {
		return fmt.Errorf("read imports %s: %w", path, err)
	}
	if err := ensureRuntimeDependenciesAvailable(path, runtimeDir, libs); err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	originalMode := info.Mode()

	if err := patchWithPatchelf(path, spec); err != nil {
		return err
	}
	return os.Chmod(path, originalMode)
}

func ensureRuntimeDependenciesAvailable(path, runtimeDir string, libs []string) error {
	missing := make([]string, 0)
	for _, lib := range libs {
		lib = strings.TrimSpace(lib)
		if lib == "" {
			continue
		}
		candidate := filepath.Join(runtimeDir, filepath.Base(lib))
		if info, err := os.Lstat(candidate); err == nil && !info.IsDir() {
			continue
		}
		missing = append(missing, lib)
	}

	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("runtime dependency closure incomplete for %s: missing %s in %s", path, strings.Join(missing, ", "), runtimeDir)
}

func patchSpecForELF(f *elf.File, runtimeDir string) (patchSpec, bool) {
	interp, err := elfInterpreter(f)
	if err != nil {
		interp = ""
	}
	libs, err := f.ImportedLibraries()
	if err != nil {
		return patchSpec{}, false
	}
	fileType := strings.TrimPrefix(f.FileHeader.Type.String(), "ET_")
	return patchSpecForELFFields(f.FileHeader.Machine, fileType, interp, libs, runtimeDir)
}

func patchSpecForELFFields(machine elf.Machine, fileType string, interp string, libs []string, runtimeDir string) (patchSpec, bool) {
	if machine != elf.EM_AARCH64 {
		return patchSpec{}, false
	}
	if fileType != "EXEC" && fileType != "DYN" {
		return patchSpec{}, false
	}

	class := toolcompat.PatchClassUnsupported
	if fileType == "EXEC" || fileType == "DYN" {
		class = toolcompat.PatchClassForELFFields(fileType, interp, libs)
	}
	switch class {
	case toolcompat.PatchClassLoaderAndRPath:
		return patchSpec{
			setInterpreter: true,
			interpreter:    filepath.Join(runtimeDir, "ld-linux-aarch64.so.1"),
			rpath:          buildRPath(runtimeDir),
		}, true
	case toolcompat.PatchClassRPathOnly:
		return patchSpec{rpath: buildRPath(runtimeDir)}, true
	default:
		return patchSpec{}, false
	}
}

func elfInterpreter(f *elf.File) (string, error) {
	for _, prog := range f.Progs {
		if prog.Type != elf.PT_INTERP {
			continue
		}
		data := make([]byte, prog.Filesz)
		if _, err := prog.ReadAt(data, 0); err != nil {
			return "", fmt.Errorf("read PT_INTERP: %w", err)
		}
		return strings.TrimRight(string(data), "\x00"), nil
	}
	return "", fmt.Errorf("no PT_INTERP")
}

func buildRPath(runtimeDir string) string {
	return strings.Join([]string{
		runtimeDir,
		"$ORIGIN",
		"$ORIGIN/../lib",
		"$ORIGIN/../lib64",
		"$ORIGIN/../libs",
		"$ORIGIN/..",
	}, ":")
}

func patchWithPatchelf(path string, spec patchSpec) error {
	args := []string{}
	if spec.setInterpreter {
		args = append(args, "--set-interpreter", spec.interpreter)
	}
	if spec.rpath != "" {
		args = append(args, "--set-rpath", spec.rpath)
	}
	args = append(args, path)
	cmd := exec.Command("patchelf", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = "patchelf failed"
		}
		return fmt.Errorf("patching %s: %w: %s", path, err, msg)
	}
	return nil
}

// isELF returns true if the file is an ELF executable.
func isELF(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	header := make([]byte, 4)
	if _, err := f.Read(header); err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}

	return header[0] == 0x7f &&
		header[1] == 'E' &&
		header[2] == 'L' &&
		header[3] == 'F', nil
}
