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
	if runtime.GOOS != "android" {
		return nil
	}

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		switch filepath.Base(path) {

		case "platform.txt":
			if err := patchPlatform(path); err != nil {
				return err
			}
		}

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
