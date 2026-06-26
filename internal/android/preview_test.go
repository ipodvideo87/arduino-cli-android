package android

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreviewPatchTreeReportsPermissionAndELFChangesWithoutMutatingFiles(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "tool.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\necho ok\n"), 0o600))

	hostExe, err := os.Executable()
	require.NoError(t, err)
	elfPath := filepath.Join(root, "bin", "tool")
	require.NoError(t, os.MkdirAll(filepath.Dir(elfPath), 0o755))
	copyFile(t, hostExe, elfPath, 0o600)

	report, err := previewPatchTreeWithAnalyzer(root, filepath.Join(root, ".acl", "runtime"), func(path string) (elfAnalysis, error) {
		return sampleAArch64ELFAnalysis(path, "bin"), nil
	})
	require.NoError(t, err)
	require.Len(t, report.Entries, 2)
	require.Equal(t, "bin/tool", report.Entries[0].RelativePath)
	require.Contains(t, report.Entries[0].Action, string(patchActionLoaderAndPath))
	require.True(t, report.Entries[0].WouldModify)
	require.Contains(t, report.Entries[0].ELF.InterpreterAfter, "ld-linux-aarch64.so.1")
	require.Equal(t, "tool.sh", report.Entries[1].RelativePath)
	require.Equal(t, "permission-repair", report.Entries[1].Action)
	require.True(t, report.Entries[1].WouldModify)

	scriptInfo, err := os.Stat(script)
	require.NoError(t, err)
	require.Zero(t, scriptInfo.Mode()&0o111)

	elfInfo, err := os.Stat(elfPath)
	require.NoError(t, err)
	require.Zero(t, elfInfo.Mode()&0o111)

	data, err := report.JSON()
	require.NoError(t, err)
	require.Contains(t, string(data), `"entries"`)
}
