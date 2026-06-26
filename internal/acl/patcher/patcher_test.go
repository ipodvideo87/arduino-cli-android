package patcher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDryRunReportsPermissionRepair(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "tool.sh")
	require.NoError(t, os.WriteFile(target, []byte("#!/bin/sh\necho hi\n"), 0o600))

	report, err := DryRun(root)
	require.NoError(t, err)
	require.NotEmpty(t, report.Entries)
	require.Equal(t, "permission-repair", report.Entries[0].Action)
	require.True(t, report.Entries[0].WouldModify)
	data, err := report.JSON()
	require.NoError(t, err)
	require.Contains(t, string(data), `"runtime_dir"`)
}
