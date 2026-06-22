package main

import (
	"bytes"
	"errors"
	"testing"

	aclexec "github.com/arduino/arduino-cli/internal/acl/exec"
	"github.com/stretchr/testify/require"
)

type stubPlanner struct {
	plan   aclexec.ExecutionPlan
	result aclexec.Result
	err    error
}

func (s stubPlanner) Run(aclexec.Request) (aclexec.ExecutionPlan, aclexec.Result, error) {
	return s.plan, s.result, s.err
}

func TestRunRejectsMissingTarget(t *testing.T) {
	original := newPlanner
	t.Cleanup(func() { newPlanner = original })
	newPlanner = func(string) plannerRunner {
		return stubPlanner{err: errors.New("missing target executable")}
	}

	var stdout, stderr bytes.Buffer
	rc := run([]string{"--runtime-root", t.TempDir()}, &stdout, &stderr)
	require.Equal(t, 1, rc)
	require.Contains(t, stderr.String(), "missing target executable")
}

func TestRunDryRunDefault(t *testing.T) {
	original := newPlanner
	t.Cleanup(func() { newPlanner = original })
	newPlanner = func(string) plannerRunner {
		return stubPlanner{
			plan: aclexec.ExecutionPlan{
				TargetPath: "/tmp/tool",
				Allowed:    true,
				Apply:      false,
			},
		}
	}

	var stdout, stderr bytes.Buffer
	rc := run([]string{"--runtime-root", t.TempDir(), "--target", "/tmp/tool", "--", "--version"}, &stdout, &stderr)
	require.Equal(t, 0, rc)
	require.Contains(t, stdout.String(), "ACL Execution Diagnostics")
	require.Contains(t, stdout.String(), "Planner strategy:")
	require.Empty(t, stderr.String())
}

func TestRunApplyReturnsBackendExitCode(t *testing.T) {
	original := newPlanner
	t.Cleanup(func() { newPlanner = original })
	newPlanner = func(string) plannerRunner {
		return stubPlanner{
			plan: aclexec.ExecutionPlan{
				TargetPath: "/tmp/tool",
				Allowed:    true,
				Apply:      true,
			},
			result: aclexec.Result{
				Stdout:   "loader stdout",
				Stderr:   "loader stderr",
				ExitCode: 17,
			},
			err: errors.New("execution failed with exit code 17"),
		}
	}

	var stdout, stderr bytes.Buffer
	rc := run([]string{"--runtime-root", t.TempDir(), "--target", "/tmp/tool", "--apply", "--", "--version"}, &stdout, &stderr)
	require.Equal(t, 17, rc)
	require.Contains(t, stdout.String(), "ACL Execution Diagnostics")
	require.Contains(t, stderr.String(), "execution failed with exit code 17")
}
