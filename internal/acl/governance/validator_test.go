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
	writeCompliantGovernanceRepo(t, root)

	report := Validate(Options{RepoRoot: root, WorkingDir: root})
	require.False(t, report.HasFailures(), "%+v", report)
	require.Equal(t, 0, report.FailureCount)
	require.Len(t, report.Checks, 12)
	for _, check := range report.Checks {
		require.True(t, check.Passed, check.Name)
	}
}

func TestValidateReportsCanonicalDocumentationDriftCases(t *testing.T) {
	root := t.TempDir()
	oldCanonical := CanonicalRepoRoot
	CanonicalRepoRoot = root
	t.Cleanup(func() { CanonicalRepoRoot = oldCanonical })
	writeCompliantGovernanceRepo(t, root)

	cases := []struct {
		name          string
		mutate        func(t *testing.T, root string)
		wantCheck     string
		wantSubstring string
	}{
		{
			name: "missing current-state ownership",
			mutate: func(t *testing.T, root string) {
				writeFixture(t, root, "STATUS.md", strings.TrimSpace(`
# Current Project Status

## Current Mission

Mission text.

## STATUS Versus ROADMAP

Status text.

## Next Engineering Milestone

Do the next thing.
`))
			},
			wantCheck:     "STATUS structure",
			wantSubstring: "current-state authority language",
		},
		{
			name: "roadmap claims current-state ownership",
			mutate: func(t *testing.T, root string) {
				writeFixture(t, root, filepath.Join("docs", "android", "ROADMAP.md"), strings.TrimSpace(`
# Roadmap

This document is the current snapshot of project progress.

## Next Milestones

- Future work only.

## STATUS Sync

Sync text.
`))
			},
			wantCheck:     "ROADMAP structure",
			wantSubstring: "current-state language",
		},
		{
			name: "missing historical classification",
			mutate: func(t *testing.T, root string) {
				writeFixture(t, root, filepath.Join("docs", "android", "ENGINEERING_MILESTONE_SUMMARY.md"), strings.TrimSpace(`
# Engineering Milestone Summary

## Milestone
Native Full-Flash Bootloader Package Validation
`))
			},
			wantCheck:     "historical classification",
			wantSubstring: "historical classification phrase",
		},
		{
			name: "historical milestone summary missing explicit route to STATUS",
			mutate: func(t *testing.T, root string) {
				writeFixture(t, root, filepath.Join("docs", "android", "ENGINEERING_MILESTONE_SUMMARY.md"), strings.TrimSpace(`
# Engineering Milestone Summary

## Historical Classification

This document is historical evidence.
It preserves repository state at the time of the milestone.
It is not authoritative for current status.

## Milestone
Native Full-Flash Bootloader Package Validation

## Repository Footer
Repository path: /root/arduino-cli-android
Branch: android-runtime-v2
Commit destination: local git commit on android-runtime-v2
Work belongs to: arduino-cli-android
Commit recommendation: yes
Repository status: dirty working tree with tracked documentation edits and untracked EOS adoption files
`))
			},
			wantCheck:     "historical classification",
			wantSubstring: "explicit STATUS.md route",
		},
		{
			name: "historical document missing route to STATUS",
			mutate: func(t *testing.T, root string) {
				writeFixture(t, root, filepath.Join("docs", "android", "QUEUED_BRANCH_REVIEW.md"), strings.TrimSpace(`
# Queued Branch Review

## Historical Classification

This document is historical evidence.
It preserves branch-review state at the time of the cleanup.
It is not authoritative for current status.
`))
			},
			wantCheck:     "historical classification",
			wantSubstring: "STATUS.md",
		},
		{
			name: "current-state doc presents the wrong repository path",
			mutate: func(t *testing.T, root string) {
				writeFixture(t, root, "STATUS.md", strings.TrimSpace(`
# Current Project Status

This document is the current snapshot of project progress.
STATUS.md is authoritative for the current snapshot.

The canonical repository for arduino-cli-android is /root/arduino-cli-android.

## Current Mission

Mission text.

## STATUS Versus ROADMAP

Status text.

## Next Engineering Milestone

Do the next thing.
`))
			},
			wantCheck:     "stale path hygiene",
			wantSubstring: "/root/arduino-cli-android",
		},
		{
			name: "missing AGENTS workflow routing",
			mutate: func(t *testing.T, root string) {
				writeFixture(t, root, "AGENTS.md", strings.TrimSpace(`
# Repository Guidelines

## Engineering Laws

These laws are constitutional.
`))
			},
			wantCheck:     "required routing",
			wantSubstring: "operational front door",
		},
		{
			name: "missing workflow closeout routing",
			mutate: func(t *testing.T, root string) {
				writeFixture(t, root, filepath.Join("docs", "android", "DEVELOPMENT_WORKFLOW.md"), strings.TrimSpace(`
# Development Workflow

## Session Closeout Checklist

Before closing the task:

- record the evidence collected
- validate the claim at the highest supported level
`))
			},
			wantCheck:     "required routing",
			wantSubstring: "CLOSEOUT_REPORTING_STANDARD.md",
		},
		{
			name: "missing workflow intent review structure",
			mutate: func(t *testing.T, root string) {
				writeFixture(t, root, filepath.Join("docs", "android", "DEVELOPMENT_WORKFLOW.md"), strings.TrimSpace(`
# Development Workflow

## Session Closeout Checklist

Before closing the task:

- record the evidence collected
- validate the claim at the highest supported level
- determine whether validated repository state changed and record one of:
  current-state synchronization required and completed, current-state
  synchronization not required with reasoning, or current-state synchronization
  required but deferred with an explicit documented blocker and uncertainty
- prepare a closeout report that follows
  `+"`docs/android/CLOSEOUT_REPORTING_STANDARD.md`"+`
- ensure every consequential judgment in the closeout report follows
  `+"`docs/android/ENGINEERING_JUDGMENT_STANDARD.md`"+`
`))
			},
			wantCheck:     "workflow intent review",
			wantSubstring: "Canonical Document Change Review",
		},
		{
			name: "missing required document",
			mutate: func(t *testing.T, root string) {
				require.NoError(t, os.Remove(filepath.Join(root, "docs", "android", "ENGINEERING_JUDGMENT_STANDARD.md")))
			},
			wantCheck:     "required documents exist",
			wantSubstring: "required document docs/android/ENGINEERING_JUDGMENT_STANDARD.md is missing",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mutate(t, root)

			report := Validate(Options{RepoRoot: root, WorkingDir: root})
			require.True(t, report.HasFailures(), "%+v", report)
			checks := map[string]CheckResult{}
			for _, check := range report.Checks {
				checks[check.Name] = check
			}
			require.False(t, checks[tc.wantCheck].Passed, tc.name)
			require.Contains(t, strings.Join(checks[tc.wantCheck].Messages, "\n"), tc.wantSubstring)
		})
	}
}

func TestValidateAllowsHistoricalOldPathWhileEnforcingCurrentStateStalePathHygiene(t *testing.T) {
	root := t.TempDir()
	oldCanonical := CanonicalRepoRoot
	CanonicalRepoRoot = root
	t.Cleanup(func() { CanonicalRepoRoot = oldCanonical })

	t.Run("historical evidence preserves old path", func(t *testing.T) {
		writeCompliantGovernanceRepo(t, root)
		writeFixture(t, root, filepath.Join("docs", "android", "ENGINEERING_MILESTONE_SUMMARY.md"), strings.TrimSpace(`
# Engineering Milestone Summary

## Historical Classification

This document is historical evidence.
It preserves repository state at the time of the milestone.
It is not authoritative for current status.
For the current snapshot, see [STATUS.md](../../STATUS.md).

Historical repository path: /root/arduino-cli-android
`))

		report := Validate(Options{RepoRoot: root, WorkingDir: root})
		require.False(t, report.HasFailures(), "%+v", report)

		checks := map[string]CheckResult{}
		for _, check := range report.Checks {
			checks[check.Name] = check
		}
		require.True(t, checks["historical classification"].Passed)
		require.True(t, checks["stale path hygiene"].Passed)
	})

	t.Run("historical milestone summary route removal fails", func(t *testing.T) {
		writeCompliantGovernanceRepo(t, root)
		writeFixture(t, root, filepath.Join("docs", "android", "ENGINEERING_MILESTONE_SUMMARY.md"), strings.TrimSpace(`
# Engineering Milestone Summary

## Historical Classification

This document is historical evidence.
It preserves repository state at the time of the milestone.
It is not authoritative for current status.

## Milestone
Native Full-Flash Bootloader Package Validation

## Repository Footer
Repository path: /root/arduino-cli-android
Branch: android-runtime-v2
Commit destination: local git commit on android-runtime-v2
Work belongs to: arduino-cli-android
Commit recommendation: yes
Repository status: dirty working tree with tracked documentation edits and untracked EOS adoption files
`))

		report := Validate(Options{RepoRoot: root, WorkingDir: root})
		require.True(t, report.HasFailures(), "%+v", report)
		checks := map[string]CheckResult{}
		for _, check := range report.Checks {
			checks[check.Name] = check
		}
		require.False(t, checks["historical classification"].Passed)
		require.Contains(t, strings.Join(checks["historical classification"].Messages, "\n"), "explicit STATUS.md route")
	})

	t.Run("current-state docs still reject stale path", func(t *testing.T) {
		writeCompliantGovernanceRepo(t, root)
		writeFixture(t, root, "STATUS.md", strings.TrimSpace(`
# Current Project Status

This document is the current snapshot of project progress.
STATUS.md is authoritative for the current snapshot.

The canonical repository for arduino-cli-android is /root/arduino-cli-android.

## Current Mission

Mission text.

## STATUS Versus ROADMAP

Status text.

## Next Engineering Milestone

Do the next thing.
`))

		report := Validate(Options{RepoRoot: root, WorkingDir: root})
		require.True(t, report.HasFailures(), "%+v", report)

		checks := map[string]CheckResult{}
		for _, check := range report.Checks {
			checks[check.Name] = check
		}
		require.False(t, checks["stale path hygiene"].Passed)
		require.Contains(t, strings.Join(checks["stale path hygiene"].Messages, "\n"), "/root/arduino-cli-android")
	})
}

func writeFixture(t *testing.T, root, rel string, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content+"\n"), 0o644))
}

func writeCompliantGovernanceRepo(t *testing.T, root string) {
	t.Helper()
	writeFixture(t, root, "README.md", strings.TrimSpace(`
# Arduino CLI Android

For the current progress snapshot, see [STATUS.md](STATUS.md).
For future milestone ordering, see [docs/android/ROADMAP.md](docs/android/ROADMAP.md).

## Canonical Working Copy

Use this checkout for active work.
`))
	writeFixture(t, root, "AGENTS.md", strings.TrimSpace(`
# Repository Guidelines

Use [docs/android/DEVELOPMENT_WORKFLOW.md](docs/android/DEVELOPMENT_WORKFLOW.md) as the operational front door for task execution.

Use [docs/android/ENGINEERING_KNOWLEDGE_FRAMEWORK.md](docs/android/ENGINEERING_KNOWLEDGE_FRAMEWORK.md) as the canonical knowledge lifecycle route.
`))
	writeFixture(t, root, "STATUS.md", strings.TrimSpace(`
# Current Project Status

This document is the current snapshot of project progress.
STATUS.md is authoritative for the current snapshot.

## Current Mission

Mission text.

## STATUS Versus ROADMAP

Status text.

## Next Engineering Milestone

Do the next thing.
`))
	writeFixture(t, root, filepath.Join("docs", "android", "ROADMAP.md"), strings.TrimSpace(`
# Roadmap

ROADMAP.md is authoritative for future ordering.
docs/android/ROADMAP.md is authoritative for future ordering.

## Next Milestones

- Future work only.

## STATUS Sync

Sync text.
`))
	writeFixture(t, root, filepath.Join("docs", "android", "DEVELOPMENT_WORKFLOW.md"), strings.TrimSpace(`
# Development Workflow

Use `+"`docs/android/ENGINEERING_KNOWLEDGE_FRAMEWORK.md`"+` as the canonical knowledge route.
Use `+"`docs/android/CLOSEOUT_REPORTING_STANDARD.md`"+` for closeout shape.
Use `+"`docs/android/ENGINEERING_JUDGMENT_STANDARD.md`"+` for consequential judgments.

## Session Closeout Checklist

Before closing the task:

- record the evidence collected
- validate the claim at the highest supported level
- determine whether validated repository state changed and record one of:
  current-state synchronization required and completed, current-state
  synchronization not required with reasoning, or current-state synchronization
  required but deferred with an explicit documented blocker and uncertainty
- prepare a closeout report that follows
  `+"`docs/android/CLOSEOUT_REPORTING_STANDARD.md`"+`
- ensure every consequential judgment in the closeout report follows
  `+"`docs/android/ENGINEERING_JUDGMENT_STANDARD.md`"+`
- update the canonical docs that own the result
- confirm `+"`STATUS.md`"+` still owns current state and `+"`docs/android/ROADMAP.md`"+` still owns future ordering, or record a documented deferral

### Canonical Document Change Review

Before editing an important canonical document:

- identify the target document
- state the purpose of the change
- name the document's current canonical role
- list the validated engineering behaviors affected
- list the dependent documents affected
- cite the evidence authorizing the change
- decide whether current-state synchronization may be required

After editing an important canonical document:

- classify the behavior change as preserved, strengthened, weakened, replaced,
  removed, contradiction introduced, or contradiction resolved
- note the routing or discoverability effect
- note the ownership effect
- state the current-state synchronization result
- state the historical/current classification result
- cite evidence authorizing any strengthening, weakening, replacement, or
  removal
- give a final compliance judgment
`))
	writeFixture(t, root, filepath.Join("docs", "android", "DOCUMENTATION_ARCHITECTURE.md"), strings.TrimSpace(`
# Documentation Architecture

## Ownership Rules

## Contract Categories

- Current-state contract owner: STATUS.md
- Future-ordering contract owner: ROADMAP.md
- Workflow contract owner: DEVELOPMENT_WORKFLOW.md
- Historical classification contract owner: this document
`))
	writeFixture(t, root, filepath.Join("docs", "android", "ENGINEERING_MILESTONE_SUMMARY.md"), strings.TrimSpace(`
# Engineering Milestone Summary

## Historical Classification

This document is historical evidence.
It preserves repository state at the time of the milestone.
It is not authoritative for current status.
For the current snapshot, see [STATUS.md](../../STATUS.md).
`))
	writeFixture(t, root, filepath.Join("docs", "android", "QUEUED_BRANCH_REVIEW.md"), strings.TrimSpace(`
# Queued Branch Review

## Historical Classification

This document is historical evidence.
It preserves branch-review state at the time of the cleanup.
It is not authoritative for current status.
For the current snapshot, see [STATUS.md](../../STATUS.md).
`))
	writeFixture(t, root, filepath.Join("docs", "android", "CLOSEOUT_REPORTING_STANDARD.md"), strings.TrimSpace(`
# Closeout Reporting Standard

This document defines the minimum closeout report shape for Codex tasks in this repository.

When a closeout includes a final verdict, confidence statement, recommendation,
or risk assessment, that judgment must also follow
`+"`ENGINEERING_JUDGMENT_STANDARD.md`"+`.
`))
	writeFixture(t, root, filepath.Join("docs", "android", "ENGINEERING_JUDGMENT_STANDARD.md"), strings.TrimSpace(`
# Engineering Judgment Standard

## Required Reporting Contract

- Judgment
- Evidence
- Reasoning
- Assumptions
- Confidence
- Remaining uncertainty
`))
	writeFixture(t, root, filepath.Join("docs", "android", "GOVERNANCE_COVERAGE_MATRIX.md"), strings.TrimSpace(`
# Governance Coverage Matrix

## Coverage Matrix

| Engineering rule | Current coverage | Evidence or tooling | Gap | Recommended next check |
| --- | --- | --- | --- | --- |
| Current-state synchronization and historical labeling | Partial | STATUS.md, DEVELOPMENT_WORKFLOW.md, DOCUMENTATION_ARCHITECTURE.md | Manual review needed | Require closeout sync determination and explicit historical classification |
`))
	writeFixture(t, root, filepath.Join("docs", "android", "ENGINEERING_DEBT_REGISTER.md"), strings.TrimSpace(`
# Engineering Debt Register

## Register Status

Current entries:

## D-0001 - canonical documentation drift

- Class: documentation
- Root cause class: workflow
- Reason: validated state can drift from canonical docs without an explicit synchronization gate.
- Impact: current-state docs can go stale while still looking authoritative.
- Evidence: Phase 6 after-state audit.
- Smallest durable improvement: require current-state synchronization determination and explicit historical classification.
- Exit criteria: deterministic checks catch the defined stale-state failures and the next milestone closes without recurrence.
- Owner: Codex governance workflow
- Related docs: DEVELOPMENT_WORKFLOW.md, DOCUMENTATION_ARCHITECTURE.md
`))
	writeFixture(t, root, filepath.Join("docs", "android", "ENGINEERING_KNOWLEDGE_FRAMEWORK.md"), strings.TrimSpace(`
# Engineering Knowledge Framework

This document defines how engineering work becomes permanent repository
knowledge.

## Canonical Ownership

- ENGINEERING_KNOWLEDGE_FRAMEWORK.md owns the policy that connects the artifacts.
`))
	writeFixture(t, root, filepath.Join("docs", "android", "TASK_RECOVERY.md"), strings.TrimSpace(`
# Task Recovery

This file is the live recovery snapshot for the current Codex task.

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
}
