package governance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatePassesForCompliantRepo(t *testing.T) {
	root := t.TempDir()
	oldCanonical := CanonicalRepoRoot
	CanonicalRepoRoot = root
	t.Cleanup(func() { CanonicalRepoRoot = oldCanonical })
	writeFixture(t, root, "README.md", strings.TrimSpace(`
# Arduino CLI Android

For the current progress snapshot, see [STATUS.md](STATUS.md).
For future milestone ordering, see [docs/android/ROADMAP.md](docs/android/ROADMAP.md).

## Canonical Working Copy

Use this checkout for active work.
`))
	writeFixture(t, root, "STATUS.md", strings.TrimSpace(`
# Status

## Current Mission

Mission text.

## STATUS Versus ROADMAP

Status text.

## Next Engineering Milestone

Do the next thing.
`))
	writeFixture(t, root, filepath.Join("docs", "android", "ROADMAP.md"), strings.TrimSpace(`
# Roadmap

## Next Milestones

- Future work only.

## STATUS Sync

Sync text.
`))
	writeFixture(t, root, filepath.Join("docs", "android", "DEVELOPMENT_WORKFLOW.md"), "Workflow text.\n")
	writeFixture(t, root, filepath.Join("docs", "android", "TASK_RECOVERY.md"), strings.TrimSpace(`
# Task Recovery

## Active Task

- none

## Intended Plan

- none

## Progress

- Completed:
  - none
- In progress:
  - none
- Remaining:
  - none

## Files

- Touched:
  - none
- Intended:
  - none

## Validation

- Status: idle
- Evidence collected:
  - none
- Evidence still needed:
  - none

## Safest Next Action

- Next action: start the next task from a fresh live snapshot.

## Canonical Follow-Through

- STATUS.md:
- VALIDATED_FINDINGS.md:
- DECISION_LOG.md:
- LESSONS_LEARNED.md:
- UNCERTAINTY_REGISTER.md:

## Reset State

When there is no active task, leave this file in place with only the template
above and blank fields.
`))

	report := Validate(Options{RepoRoot: root, WorkingDir: root})
	require.False(t, report.HasFailures(), "%+v", report)
	require.Equal(t, 0, report.FailureCount)
	require.Equal(t, 7, len(report.Checks))
	for _, check := range report.Checks {
		require.True(t, check.Passed, check.Name)
	}
}

func TestValidateReportsAllFailuresTogether(t *testing.T) {
	root := t.TempDir()
	oldCanonical := CanonicalRepoRoot
	CanonicalRepoRoot = root
	t.Cleanup(func() { CanonicalRepoRoot = oldCanonical })
	writeFixture(t, root, "README.md", strings.TrimSpace(`
# Arduino CLI Android

## Current Validated State

## What Is Still Experimental

## Current Focus

## Recent Work

## Future Work
`))
	writeFixture(t, root, "STATUS.md", strings.TrimSpace(`
# Status

## Current Mission

Mission text.

## STATUS Versus ROADMAP

Status text.

## Next Engineering Milestone

Do the next thing.

target chip metadata is not set
`))
	writeFixture(t, root, filepath.Join("docs", "android", "ROADMAP.md"), strings.TrimSpace(`
# Roadmap

## Next Milestones

- Validate the Termux USB provider and file-descriptor handoff path on native Termux
- Validate the TERMUX_USB_FD probe surface on native Termux
- Validate upload prepare-only planning on native Termux and keep the CLI/report contract explicit

## STATUS Sync

Sync text.
`))
	writeFixture(t, root, filepath.Join("docs", "android", "DEVELOPMENT_WORKFLOW.md"), "Workflow text.\n")
	writeFixture(t, root, filepath.Join("docs", "android", "TASK_RECOVERY.md"), strings.TrimSpace(`
# Task Recovery

## Active Task

- Objective: stale completed task residue
`))

	report := Validate(Options{RepoRoot: root, WorkingDir: filepath.Join(root, "other")})
	require.True(t, report.HasFailures())
	require.Greater(t, report.FailureCount, 0)
	require.Len(t, report.FailedChecks(), 6)

	checks := map[string]CheckResult{}
	for _, check := range report.Checks {
		checks[check.Name] = check
	}
	require.False(t, checks["canonical repo root"].Passed)
	require.False(t, checks["README overview-only"].Passed)
	require.False(t, checks["STATUS structure"].Passed)
	require.False(t, checks["TASK_RECOVERY state"].Passed)
	require.False(t, checks["ROADMAP future items"].Passed)
	require.Contains(t, strings.Join(checks["README overview-only"].Messages, "\n"), "Current Validated State")
	require.Contains(t, strings.Join(checks["STATUS structure"].Messages, "\n"), "resolved target-chip unresolved wording")
	require.Contains(t, strings.Join(checks["TASK_RECOVERY state"].Messages, "\n"), "neither idle nor explicitly active")
	require.Contains(t, strings.Join(checks["ROADMAP future items"].Messages, "\n"), "Validate the TERMUX_USB_FD probe surface on native Termux")
	require.False(t, checks["stale phrases"].Passed)
	require.Contains(t, strings.Join(checks["stale phrases"].Messages, "\n"), "target chip metadata is not set")
}

func TestValidateAcceptsIdleOrExplicitActiveTaskRecovery(t *testing.T) {
	root := t.TempDir()
	oldCanonical := CanonicalRepoRoot
	CanonicalRepoRoot = root
	t.Cleanup(func() { CanonicalRepoRoot = oldCanonical })
	writeFixture(t, root, "README.md", "# Arduino CLI Android\n")
	writeFixture(t, root, "STATUS.md", "## Current Mission\n\n## STATUS Versus ROADMAP\n\n## Next Engineering Milestone\n")
	writeFixture(t, root, filepath.Join("docs", "android", "ROADMAP.md"), "## Next Milestones\n\n## STATUS Sync\n")
	writeFixture(t, root, filepath.Join("docs", "android", "DEVELOPMENT_WORKFLOW.md"), "Workflow text.\n")
	writeFixture(t, root, filepath.Join("docs", "android", "TASK_RECOVERY.md"), strings.TrimSpace(`
# Task Recovery

## Active Task

- Objective: add governance automation

## Intended Plan

- Step 1: implement the validator
`))

	report := Validate(Options{RepoRoot: root, WorkingDir: root})
	require.False(t, report.HasFailures())
	checks := map[string]CheckResult{}
	for _, check := range report.Checks {
		checks[check.Name] = check
	}
	require.True(t, checks["TASK_RECOVERY state"].Passed)
}

func writeFixture(t *testing.T, root, rel string, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content+"\n"), 0o644))
}
