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

func TestLooksLikeXtensaDynConfig(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "xtensa-esp32s3-elf-gcc")
	data := []byte("XTENSA_GNU_CONFIG\x00-mdynconfig=\x00xtensa_.so\x00")
	require.NoError(t, os.WriteFile(path, data, 0o644))

	require.True(t, looksLikeXtensaDynConfig(path))
}

func TestRustLauncherDelegateTargets(t *testing.T) {
	root := t.TempDir()
	launcherDir := filepath.Join(root, "bin")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "lib"), 0o755))
	require.NoError(t, os.MkdirAll(launcherDir, 0o755))
	launcher := filepath.Join(launcherDir, "xtensa-esp32s3-elf-gcc")
	backend := filepath.Join(root, "lib", "xtensa_esp32s3.so")
	copyExecutable(t, backend)
	require.NoError(t, os.WriteFile(launcher, []byte("Get executable path\x00Current exe has path\x00Called tool must have pattern \"xtensa-esp*-elf-*\"\x00Dynconfig for target\x00XTENSA_GNU_CONFIG\x00mdynconfig=\x00xtensa_.so\x00execv errno\x00"), 0o644))

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
		require.Empty(t, target.InspectionError)
		require.True(t, target.IsELF)
		require.Equal(t, "xtensa_esp32s3.so", filepath.Base(target.Path))
		require.NotEmpty(t, target.FileType)
		require.NotEmpty(t, target.Interpreter)
		require.NotEmpty(t, target.Needed)
	}
	require.True(t, found, "expected a chip plugin delegate candidate to be identified")
}

func copyExecutable(t *testing.T, dst string) {
	t.Helper()
	src, err := os.Executable()
	require.NoError(t, err)
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(dst, data, 0o755))
}
