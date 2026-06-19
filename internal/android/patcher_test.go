package android

import (
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
