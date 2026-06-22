package android

import (
	"debug/elf"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// PatchPlatformForAndroid patches an installed platform tree using the ACL
// integration entrypoint.
func PatchPlatformForAndroid(root string) error {
	return patchInstallTree(root, true, runtime.GOOS)
}

// PatchToolForAndroid patches an installed tool tree using the ACL integration
// entrypoint.
func PatchToolForAndroid(root string) error {
	return patchInstallTree(root, false, runtime.GOOS)
}

func patchInstallTree(root string, patchPlatformTxt bool, goos string) error {
	if goos != "android" {
		return nil
	}
	runtimeDir, err := installRuntime(root)
	if err != nil {
		return err
	}
	if err := repairExecutableModes(root); err != nil {
		return err
	}

	if patchPlatformTxt {
		if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || filepath.Base(path) != "platform.txt" {
				return nil
			}
			return patchPlatform(path)
		}); err != nil {
			return err
		}
	}

	return patchELFs(root, runtimeDir)
}

// patchPlatform modifies platform.txt for Android compatibility.
func patchPlatform(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	text := string(data)
	text = strings.ReplaceAll(text, "/usr/bin/env", "env")

	return os.WriteFile(filename, []byte(text), 0o644)
}

func repairExecutableModes(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		isExec, err := looksExecutableByContent(path)
		if err != nil {
			return err
		}
		if !isExec || info.Mode()&0o111 != 0 {
			return nil
		}
		return os.Chmod(path, info.Mode()|0o111)
	})
}

func looksExecutableByContent(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	header := make([]byte, 4)
	n, err := file.Read(header)
	if err != nil && n == 0 {
		if err == io.EOF {
			return false, nil
		}
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	if n >= 4 && header[0] == 0x7f && header[1] == 'E' && header[2] == 'L' && header[3] == 'F' {
		elfFile, err := elf.NewFile(file)
		if err != nil {
			return false, nil
		}
		defer elfFile.Close()
		if elfFile.FileHeader.Type == elf.ET_EXEC {
			return true, nil
		}
		for _, prog := range elfFile.Progs {
			if prog.Type == elf.PT_INTERP {
				return true, nil
			}
		}
		return false, nil
	}

	if n < 2 || header[0] != '#' || header[1] != '!' {
		return false, nil
	}
	return true, nil
}
