package acl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestACLGovernanceValidateCommandReportsFailures(t *testing.T) {
	rootDir := t.TempDir()
	writeGovernanceFixture(t, rootDir, "README.md", strings.TrimSpace(`
# Arduino CLI Android

## Current Validated State

## What Is Still Experimental
`))
	writeGovernanceFixture(t, rootDir, "STATUS.md", strings.TrimSpace(`
# Status

## Current Mission

## STATUS Versus ROADMAP

## Next Engineering Milestone
`))
	writeGovernanceFixture(t, rootDir, filepath.Join("docs", "android", "ROADMAP.md"), strings.TrimSpace(`
# Roadmap

## Next Milestones

- Validate the Termux USB provider and file-descriptor handoff path on native Termux

## STATUS Sync
`))
	writeGovernanceFixture(t, rootDir, filepath.Join("docs", "android", "TASK_RECOVERY.md"), strings.TrimSpace(`
# Task Recovery

## Active Task

- none
`))

	oldRepoRoot := governanceRepoRootFunc
	oldWorkingDir := governanceWorkingDirFunc
	governanceRepoRootFunc = func() string { return rootDir }
	governanceWorkingDirFunc = func() string { return rootDir }
	t.Cleanup(func() {
		governanceRepoRootFunc = oldRepoRoot
		governanceWorkingDirFunc = oldWorkingDir
	})

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "governance", "validate"})

	err := root.Execute()
	require.Error(t, err)
	require.Contains(t, buf.String(), "ACL Governance Validation")
	require.Contains(t, buf.String(), "README overview-only")
	require.Contains(t, buf.String(), "ROADMAP future items")
}

func writeGovernanceFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content+"\n"), 0o644))
}
