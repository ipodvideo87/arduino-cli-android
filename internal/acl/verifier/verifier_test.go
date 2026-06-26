package verifier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifierProducesJSONAndBeginnerSummary(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "tool.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\necho hi\n"), 0o755))

	target, err := os.Executable()
	require.NoError(t, err)

	report, err := New("").Verify(Request{
		Root:       root,
		TargetPath: target,
		Args:       []string{"--version"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, report.Beginner)
	require.NotEmpty(t, report.Professional)

	data, err := report.JSON()
	require.NoError(t, err)
	require.Contains(t, string(data), `"scan"`)
	require.Contains(t, string(data), `"execution"`)
}
