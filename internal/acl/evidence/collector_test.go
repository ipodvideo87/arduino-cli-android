package evidence

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeCollectorRunner struct {
	calls   []CommandSpec
	results map[string]CommandResult
}

func (r *fakeCollectorRunner) Run(_ context.Context, spec CommandSpec) CommandResult {
	r.calls = append(r.calls, spec)
	if result, ok := r.results[specKey(spec)]; ok {
		return result
	}
	return CommandResult{ExitCode: 127, Err: errors.New("unexpected command")}
}

func specKey(spec CommandSpec) string {
	return spec.Name + " " + strings.Join(spec.Args, " ")
}

func TestCollectorShapesEvidenceEnvelope(t *testing.T) {
	tempDir := t.TempDir()
	runner := &fakeCollectorRunner{
		results: map[string]CommandResult{
			specKey(CommandSpec{Name: "/tmp/arduino-cli", Args: []string{"acl", "transport", "list", "--json"}}): {
				Stdout:   "{\n  \"status\": \"warning\",\n  \"provider\": \"termuxusb\"\n}",
				ExitCode: 0,
			},
			specKey(CommandSpec{Name: "/tmp/arduino-cli", Args: []string{"acl", "transport", "diagnose", "--json", "--details", "--device", "/dev/bus/usb/001/002"}}): {
				Stdout:   "{\n  \"status\": \"warning\",\n  \"device\": {\"stable_id\": \"/dev/bus/usb/001/002\"}\n}",
				ExitCode: 0,
			},
			specKey(CommandSpec{Name: "/tmp/arduino-cli", Args: []string{"acl", "transport", "probe-fd", "--json", "--device", "/dev/bus/usb/001/002"}}): {
				Stdout:   "{\n  \"fd_source\": \"environment\",\n  \"fd_valid\": true\n}",
				ExitCode: 0,
			},
			specKey(CommandSpec{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}}): {
				Stdout:   tempDir + "\n",
				ExitCode: 0,
			},
			specKey(CommandSpec{Name: "git", Args: []string{"branch", "--show-current"}}): {
				Stdout:   "android-runtime-v2\n",
				ExitCode: 0,
			},
			specKey(CommandSpec{Name: "git", Args: []string{"rev-parse", "HEAD"}}): {
				Stdout:   "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef\n",
				ExitCode: 0,
			},
			specKey(CommandSpec{Name: "git", Args: []string{"status", "--short", "--branch"}}): {
				Stdout:   "## android-runtime-v2...origin/android-runtime-v2 [ahead 4]\n",
				ExitCode: 0,
			},
		},
	}
	collector := NewCollectorWithRunner(runner)
	collector.binaryPath = "/tmp/arduino-cli"

	bundle, err := collector.Collect(context.Background(), CollectOptions{
		DevicePath: "/dev/bus/usb/001/002",
		OutputDir:  filepath.Join(tempDir, "evidence"),
	})
	require.NoError(t, err)
	require.NotEmpty(t, bundle.OutputPath)
	require.FileExists(t, bundle.OutputPath)
	require.Equal(t, tempDir, bundle.Repository.Root)
	require.Equal(t, "android-runtime-v2", strings.TrimSpace(bundle.Repository.Branch))
	require.Equal(t, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", strings.TrimSpace(bundle.Repository.Commit))
	require.Contains(t, bundle.Repository.Status, "ahead 4")
	require.Equal(t, "/dev/bus/usb/001/002", bundle.DevicePath)
	require.Equal(t, 7, len(bundle.Commands))
	require.Equal(t, "json", bundle.Commands[4].NormalizedReportType)
	require.Equal(t, "json", bundle.Commands[5].NormalizedReportType)
	require.Equal(t, "json", bundle.Commands[6].NormalizedReportType)
	require.Equal(t, "{\n  \"status\": \"warning\",\n  \"provider\": \"termuxusb\"\n}", strings.TrimSpace(bundle.Commands[4].Stdout))
	require.Equal(t, "warning", strings.ToLower(string(bundle.Status)))
	for _, command := range bundle.Commands {
		require.False(t, command.Mutating)
		require.True(t, command.Allowlisted)
	}
}

func TestCollectorCapturesRawTraceAndWarnings(t *testing.T) {
	tempDir := t.TempDir()
	runner := &fakeCollectorRunner{
		results: map[string]CommandResult{
			specKey(CommandSpec{Name: "/tmp/arduino-cli", Args: []string{"acl", "transport", "list", "--json"}}): {
				Stdout:   "{\"status\":\"warning\"}",
				ExitCode: 0,
			},
			specKey(CommandSpec{Name: "/tmp/arduino-cli", Args: []string{"acl", "transport", "diagnose", "--json", "--details", "--device", "/dev/bus/usb/001/002"}}): {
				Stdout:   "{\"status\":\"warning\"}",
				ExitCode: 0,
			},
			specKey(CommandSpec{Name: "/tmp/arduino-cli", Args: []string{"acl", "transport", "probe-fd", "--json", "--device", "/dev/bus/usb/001/002"}}): {
				Stdout:   "{\"status\":\"warning\"}",
				Stderr:   "probe failed",
				ExitCode: 3,
				Err:      errors.New("exit status 3"),
			},
			specKey(CommandSpec{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}}): {
				Stdout:   tempDir + "\n",
				ExitCode: 0,
			},
			specKey(CommandSpec{Name: "git", Args: []string{"branch", "--show-current"}}): {
				Stdout:   "android-runtime-v2\n",
				ExitCode: 0,
			},
			specKey(CommandSpec{Name: "git", Args: []string{"rev-parse", "HEAD"}}): {
				Stdout:   "deadbeef\n",
				ExitCode: 0,
			},
			specKey(CommandSpec{Name: "git", Args: []string{"status", "--short", "--branch"}}): {
				Stdout:   "## android-runtime-v2...origin/android-runtime-v2 [ahead 4]\n",
				ExitCode: 0,
			},
		},
	}
	collector := NewCollectorWithRunner(runner)
	collector.binaryPath = "/tmp/arduino-cli"

	bundle, err := collector.Collect(context.Background(), CollectOptions{
		DevicePath: "/dev/bus/usb/001/002",
		OutputDir:  filepath.Join(tempDir, "evidence"),
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(bundle.Warnings))
	require.Equal(t, "probe failed", bundle.Commands[6].Stderr)
	require.Equal(t, 3, bundle.Commands[6].ExitCode)
	require.NotEmpty(t, bundle.Commands[6].StartedAtUTC)
	require.NotEmpty(t, bundle.Commands[6].FinishedAtUTC)
	require.GreaterOrEqual(t, bundle.Commands[6].DurationMS, int64(0))
	require.Equal(t, "warning", strings.ToLower(string(bundle.Status)))
}

func TestValidatePlanRejectsNonAllowlistedCommand(t *testing.T) {
	err := validatePlan([]CommandSpec{
		{Name: "touch", Args: []string{"/tmp/should-not-run"}, Allowlisted: false},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowlisted")
}

func TestValidatePlanRejectsMutatingCommand(t *testing.T) {
	err := validatePlan([]CommandSpec{
		{Name: "git", Args: []string{"status"}, Allowlisted: true, Mutating: true},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "mutating")
}
