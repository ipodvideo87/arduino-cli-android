package android

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// PatchPlatformForAndroid patches an installed Arduino platform
// so it can execute correctly under Android/Termux.
func PatchPlatformForAndroid(root string) error {
	// Only do anything on Android.
	if runtime.GOOS != "android" {
		return nil
	}

	// Patch all platform.txt files.
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Base(path) == "platform.txt" {
			if err := patchPlatform(path); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	// Patch Linux executables (implementation comes next).
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

		// TODO:
		// Detect ELF files.
		// Run grun --set <path>
		// (We'll implement this next.)

		return nil
	})
}
func patchPlatform(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	text := string(data)

	text = strings.ReplaceAll(
		text,
		"/usr/bin/env",
		"env",
	)

	return os.WriteFile(filename, []byte(text), 0644)
}
