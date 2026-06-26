package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScannerValidateAndJSON(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "tool.sh")
	require.NoError(t, os.WriteFile(target, []byte("#!/bin/sh\necho hi\n"), 0o755))

	report, err := New().Scan(root)
	require.NoError(t, err)
	require.Len(t, report.Entries, 1)

	data, err := report.JSON()
	require.NoError(t, err)
	require.Contains(t, string(data), `"root"`)
	require.NotEmpty(t, report.BeginnerSummary())

	validation := Validate(report)
	vdata, err := validation.JSON()
	require.NoError(t, err)
	require.Contains(t, string(vdata), `"summary"`)
	require.NotEmpty(t, validation.BeginnerSummary())
}
