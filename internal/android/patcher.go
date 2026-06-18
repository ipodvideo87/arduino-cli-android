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
	// Nothing to do unless we're running on Android.
	if runtime.GOOS != "android" {
		return nil
	}

	// Walk the installed platform.
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Patch platform.txt
		if filepath.Base(path) == "platform.txt" {
			if err := patchPlatform(path); err != nil {
				return err
			}
			return nil
		}

		// Patch executable binaries.
		if info.Mode()&0111 != 0 {
			if err := patchExecutable(path); err != nil {
				return err
			}
		}

		return nil
	})
}

// patchPlatform modifies platform.txt for Android compatibility.
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
