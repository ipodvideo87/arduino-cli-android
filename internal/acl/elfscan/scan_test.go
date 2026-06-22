package elfscan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLooksLikeRustLauncher(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "xtensa-esp32s3-elf-gcc")
	data := []byte("Get executable path\x00Current exe has path\x00Called tool must have pattern \"xtensa-esp*-elf-*\"\x00Dynconfig for target\x00XTENSA_GNU_CONFIG\x00execv errno\x00")
	require.NoError(t, os.WriteFile(path, data, 0o644))

	require.True(t, looksLikeRustLauncher(path))
}

func TestLooksLikeRustLauncherRejectsUnrelatedBinary(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "tool")
	require.NoError(t, os.WriteFile(path, []byte("plain data with no markers"), 0o644))

	require.False(t, looksLikeRustLauncher(path))
}

func TestRustLauncherDelegateTargets(t *testing.T) {
	root := t.TempDir()
	launcher := filepath.Join(root, "xtensa-esp32s3-elf-gcc")
	backend := filepath.Join(root, "xtensa-esp32s3-elf-gcc.real")
	require.NoError(t, os.WriteFile(backend, []byte("backend"), 0o755))
	require.NoError(t, os.WriteFile(launcher, []byte("Get executable path\x00Current exe has path\x00Called tool must have pattern \"xtensa-esp*-elf-*\"\x00Dynconfig for target\x00XTENSA_GNU_CONFIG\x00execv errno\x00"), 0o644))

	targets := findRustLauncherDelegateTargets(launcher)
	require.NotEmpty(t, targets)

	found := false
	for _, target := range targets {
		if target.Path != backend {
			continue
		}
		found = true
		require.True(t, target.Exists)
		require.True(t, target.Executable)
		require.NotEmpty(t, target.Source)
	}
	require.True(t, found, "expected a .real delegate candidate to be identified")
}
