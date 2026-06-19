package android

import (
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
