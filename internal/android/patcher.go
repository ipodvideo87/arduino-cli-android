package android

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// PatchPlatformForAndroid patches an installed Arduino platform
// so it works correctly under Android/Termux.
func PatchPlatformForAndroid(root string) error {
	// Only run on Android.
	if runtime.GOOS != "android" {
		return nil
	}

	// Walk the platform directory looking for files to patch.
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Patch platform.txt files.
		if !info.IsDir() && filepath.Base(path) == "platform.txt" {
			if err := patchPlatform(path); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	// Patch Linux executables.
	if err := patchExecutables(root); err != nil {
		return err
	}

	return nil
}

// patchPlatform modifies platform.txt so it works under Termux.
func patchPlatform(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	text := string(data)

	// Replace hardcoded /usr/bin/env with env.
	text = strings.ReplaceAll(
		text,
		"/usr/bin/env",
		"env",
	)

	return os.WriteFile(filename, []byte(text), 0644)
}

// patchExecutables walks the installed platform looking for
// Linux executables that need Android compatibility fixes.
func patchExecutables(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories.
		if info.IsDir() {
			return nil
		}

		// Skip files that aren't executable.
		if info.Mode()&0111 == 0 {
			return nil
		}

		// Only process ELF executables.
		ok, err := isELF(path)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		// TODO:
		// Run:
		//     glibc-runner -c <path>
		//
		// or eventually patch the ELF directly with patchelf.
		//
		// We'll implement that next.

		return nil
	}
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
