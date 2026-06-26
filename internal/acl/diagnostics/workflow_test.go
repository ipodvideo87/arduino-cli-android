package diagnostics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkflowTracksStatusesAndSummary(t *testing.T) {
	w := NewWorkflow("USB diagnostics", "usb")
	w.AddStep("discover")
	w.AddStep("open")
	w.AddStep("bridge")

	w.SetStatus("discover", StatusPassed, "device listed")
	w.SetStatus("open", StatusWarning, "permission prompt was slow")
	w.SetStatus("bridge", StatusRunning, "building PTY")

	require.Equal(t, 4, len(w.Steps))
	require.Equal(t, StatusPassed, mustStep(t, w, "discover").Status)
	require.Equal(t, StatusWarning, mustStep(t, w, "open").Status)
	require.Equal(t, StatusRunning, mustStep(t, w, "bridge").Status)
	require.Greater(t, w.Progress(), 0.0)
	require.Contains(t, w.Summary(), "permission prompt was slow")
	require.Contains(t, w.Summary(), "USB diagnostics")
}

func TestWorkflowCountsPendingSteps(t *testing.T) {
	w := NewWorkflow("Build")
	w.AddStep("compile")
	w.AddStep("package")

	counts := w.Counts()
	require.Equal(t, 2, counts[StatusPending])
	require.Equal(t, 0, counts[StatusPassed])
}

func mustStep(t *testing.T, w *Workflow, name string) Step {
	t.Helper()
	step, ok := w.Step(name)
	require.True(t, ok)
	return step
}
