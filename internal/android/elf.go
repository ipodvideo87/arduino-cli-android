package android

import (
	"debug/elf"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var glibcSONames = map[string]struct{}{
	"libc.so.6":       {},
	"libpthread.so.0": {},
	"libdl.so.2":      {},
	"librt.so.1":      {},
	"libm.so.6":       {},
	"libstdc++.so.6":  {},
	"libgcc_s.so.1":   {},
	"libz.so.1":       {},
}

// patchELFs walks an installed tree looking for host executables that ACL may
// need to rewrite for Android compatibility.
func patchELFs(root, runtimeDir string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
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
		return err
	}
	if !isElf {
		return nil
	}

	f, err := elf.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if !shouldPatchELF(f) {
		return nil
	}

	return patchWithPatchelf(path, filepath.Join(runtimeDir, "ld-linux-aarch64.so.1"), buildRPath(runtimeDir))
}

func shouldPatchELF(f *elf.File) bool {
	if f.FileHeader.Machine != elf.EM_AARCH64 {
		return false
	}
	if f.FileHeader.Type != elf.ET_EXEC && f.FileHeader.Type != elf.ET_DYN {
		return false
	}

	interp, err := elfInterpreter(f)
	if err == nil && filepath.Base(interp) == "ld-linux-aarch64.so.1" {
		return true
	}

	libs, err := f.ImportedLibraries()
	if err != nil {
		return false
	}
	for _, lib := range libs {
		if _, ok := glibcSONames[lib]; ok {
			return true
		}
	}
	return false
}

func elfInterpreter(f *elf.File) (string, error) {
	for _, prog := range f.Progs {
		if prog.Type != elf.PT_INTERP {
			continue
		}
		data := make([]byte, prog.Filesz)
		if _, err := prog.ReadAt(data, 0); err != nil {
			return "", err
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

func patchWithPatchelf(path, interpreter, rpath string) error {
	cmd := exec.Command("patchelf", "--set-interpreter", interpreter, "--set-rpath", rpath, path)
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
		return false, err
	}

	return header[0] == 0x7f &&
		header[1] == 'E' &&
		header[2] == 'L' &&
		header[3] == 'F', nil
}
