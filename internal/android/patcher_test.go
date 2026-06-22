package android

import (
	"debug/elf"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeOSSuffix(t *testing.T) {
	require.Equal(t, "linux", RuntimeOSSuffix("android"))
}

func TestInstallRuntimeCopiesEmbeddedFiles(t *testing.T) {
	root := t.TempDir()
	runtimeDir, err := installRuntime(root)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(runtimeDir, "ld-linux-aarch64.so.1"))
	require.FileExists(t, filepath.Join(runtimeDir, "libc.so.6"))
	require.FileExists(t, filepath.Join(runtimeDir, "libm.so.6"))
	require.FileExists(t, filepath.Join(runtimeDir, "libgcc_s.so.1"))
	info, err := os.Stat(filepath.Join(runtimeDir, "ld-linux-aarch64.so.1"))
	require.NoError(t, err)
	require.NotZero(t, info.Mode().Perm()&0o111)

	linkInfo, err := os.Lstat(filepath.Join(runtimeDir, "libc.so"))
	require.NoError(t, err)
	require.True(t, linkInfo.Mode()&os.ModeSymlink != 0)
}

func TestPatchInstallTreePatchesPlatformTxtForAndroid(t *testing.T) {
	root := t.TempDir()
	platformTxt := filepath.Join(root, "platform.txt")
	require.NoError(t, os.WriteFile(platformTxt, []byte("/usr/bin/env python3\n"), 0o644))

	require.NoError(t, patchInstallTree(root, true, "android"))

	data, err := os.ReadFile(platformTxt)
	require.NoError(t, err)
	require.Equal(t, "env python3\n", string(data))
	require.DirExists(t, filepath.Join(root, aclDirName, aclRuntimeName))
}

func TestPatchInstallTreeLeavesPlatformTxtUntouchedForTools(t *testing.T) {
	root := t.TempDir()
	platformTxt := filepath.Join(root, "platform.txt")
	require.NoError(t, os.WriteFile(platformTxt, []byte("/usr/bin/env python3\n"), 0o644))

	require.NoError(t, patchInstallTree(root, false, "android"))

	data, err := os.ReadFile(platformTxt)
	require.NoError(t, err)
	require.Equal(t, "/usr/bin/env python3\n", string(data))
}

func TestPatchInstallTreeIgnoresScripts(t *testing.T) {
	root := t.TempDir()
	scriptPath := filepath.Join(root, "tool.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/sh\necho ok\n"), 0o755))
	require.NoError(t, patchInstallTree(root, false, "android"))
}

func TestPatchInstallTreeRepairsMissingExecuteBitsForExecutableFiles(t *testing.T) {
	root := t.TempDir()
	hostExe, err := os.Executable()
	require.NoError(t, err)

	elfPath := filepath.Join(root, "bin", "xtensa-esp-elf-gcc")
	require.NoError(t, os.MkdirAll(filepath.Dir(elfPath), 0o755))
	copyFile(t, hostExe, elfPath, 0o600)

	scriptPath := filepath.Join(root, "bin", "tool.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/sh\necho ok\n"), 0o600))

	require.NoError(t, patchInstallTree(root, false, "android"))

	requireExecutableMode(t, elfPath)
	requireExecutableMode(t, scriptPath)
}

func TestPatchInstallTreeSkipsEmptyFiles(t *testing.T) {
	root := t.TempDir()
	empty := filepath.Join(root, "bin", "empty")
	require.NoError(t, os.MkdirAll(filepath.Dir(empty), 0o755))
	require.NoError(t, os.WriteFile(empty, nil, 0o600))

	require.NoError(t, patchInstallTree(root, false, "android"))
}

func TestEnsureRuntimeDependenciesAvailableAcceptsCompleteRuntime(t *testing.T) {
	runtimeDir := t.TempDir()
	for _, name := range []string{"libc.so.6", "libdl.so.2", "libm.so.6"} {
		require.NoError(t, os.WriteFile(filepath.Join(runtimeDir, name), []byte("x"), 0o644))
	}

	err := ensureRuntimeDependenciesAvailable("/tmp/cc1plus", runtimeDir, []string{"libc.so.6", "libdl.so.2", "libm.so.6"})
	require.NoError(t, err)
}

func TestEnsureRuntimeDependenciesAvailableReportsMissingLibrary(t *testing.T) {
	runtimeDir := t.TempDir()
	for _, name := range []string{"libc.so.6", "libdl.so.2"} {
		require.NoError(t, os.WriteFile(filepath.Join(runtimeDir, name), []byte("x"), 0o644))
	}

	err := ensureRuntimeDependenciesAvailable("/tmp/cc1plus", runtimeDir, []string{"libc.so.6", "libdl.so.2", "libm.so.6"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "/tmp/cc1plus")
	require.Contains(t, err.Error(), "libm.so.6")
}

func TestPatchExecutableWrapsFilePathOnMalformedELF(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "bin", "broken")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("\x7fELF"), 0o755))

	err := patchExecutable(path, filepath.Join(root, ".acl", "runtime"))
	require.Error(t, err)
	require.Contains(t, err.Error(), path)
}

func TestPatchSpecForELFSkipsNonAArch64(t *testing.T) {
	spec, ok := patchSpecForELFFields(elf.EM_X86_64, "EXEC", "/lib64/ld-linux-x86-64.so.2", []string{"libc.so.6"}, "/tmp/runtime")
	require.False(t, ok)
	require.Equal(t, patchSpec{}, spec)
}

func TestPatchSpecForELFUsesInterpreterForExecutables(t *testing.T) {
	spec, ok := patchSpecForELFFields(elf.EM_AARCH64, "EXEC", "/lib/ld-linux-aarch64.so.1", []string{"libc.so.6"}, "/tmp/runtime")
	require.True(t, ok)
	require.True(t, spec.setInterpreter)
	require.Equal(t, filepath.Join("/tmp/runtime", "ld-linux-aarch64.so.1"), spec.interpreter)
	require.Contains(t, spec.rpath, "/tmp/runtime")
}

func TestPatchSpecForELFSkipsInterpreterForSharedLibraries(t *testing.T) {
	spec, ok := patchSpecForELFFields(elf.EM_AARCH64, "DYN", "", []string{"libc.so.6"}, "/tmp/runtime")
	require.True(t, ok)
	require.False(t, spec.setInterpreter)
	require.Empty(t, spec.interpreter)
	require.Contains(t, spec.rpath, "/tmp/runtime")
}

func copyFile(t *testing.T, src, dst string, mode os.FileMode) {
	t.Helper()
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(dst, data, mode))
}

func requireExecutableMode(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&0o111, "expected executable bits for %s", path)
}
