package android

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// PatchPlatformForAndroid patches an installed Arduino platform
// so it works correctly under Android/Termux.
func PatchPlatformForAndroid(root string) error {
	// Only patch on Android.
	if runtime.GOOS != "android" {
		return nil
	}

	fmt.Printf("Android: patching platform %s\n", root)

	// Walk the platform looking for platform.txt files.
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Base(path) == "platform.txt" {
			fmt.Printf("Android: patching %s\n", path)
			return patchPlatform(path)
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Patch executables.
	if err := patchExecutables(root); err != nil {
		return err
	}

	return nil
}

// patchPlatform patches platform.txt for Android.
func patchPlatform(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	text := string(data)

	// Replace hardcoded /usr/bin/env.
	text = strings.ReplaceAll(text, "/usr/bin/env", "env")

	return os.WriteFile(filename, []byte(text), 0644)
}

// patchExecutables patches every ELF executable in the platform.
func patchExecutables(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Skip non-executable files.
		if info.Mode()&0111 == 0 {
			return nil
		}

		ok, err := isELF(path)
		if err != nil {
			return err
		}

		if !ok {
			return nil
		}

		fmt.Printf("Android: patching executable %s\n", path)

		if err := runPatcher(path); err != nil {
			fmt.Printf("Android: warning: %v\n", err)
		}

		return nil
	})
}

// runPatcher attempts to configure a Linux executable for Android.
func runPatcher(path string) error {
	commands := [][]string{
		{"glibc-runner", "-c", path},
		{"grun", "-c", path},
	}

	for _, command := range commands {
		cmd := exec.Command(command[0], command[1:]...)

		output, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}

		if _, ok := err.(*exec.Error); ok {
			// Command not installed.
			continue
		}

		return fmt.Errorf("%s failed:\n%s", command[0], string(output))
	}

	return fmt.Errorf("neither glibc-runner nor grun was found")
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
