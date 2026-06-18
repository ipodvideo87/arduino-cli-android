package android

import (
	"os"
)

// patchELFs walks a platform looking for Linux ELF executables.
func patchELFs(root string) error {
	return nil
}

// patchExecutable patches a single executable if needed.
func patchExecutable(path string) error {
	isElf, err := isELF(path)
	if err != nil {
		return err
	}

	// Ignore scripts and other non-ELF executables.
	if !isElf {
		return nil
	}

	// TODO:
	// This is where we'll eventually:
	//   - Patch the ELF interpreter
	//   - Patch the runtime search path
	//   - Remove the need for grun/patchelf
	//
	// For now we simply detect the executable.

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
