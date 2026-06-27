package termuxusb

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/acl/transport"
	"github.com/stretchr/testify/require"
)

type scriptedRunner struct {
	lookPath map[string]error
	results  map[string]CommandResult
	calls    []string
}

func (r *scriptedRunner) LookPath(name string) (string, error) {
	if r.lookPath != nil {
		if err, ok := r.lookPath[name]; ok {
			return "", err
		}
	}
	return "/usr/bin/" + name, nil
}

func (r *scriptedRunner) Run(_ context.Context, name string, args ...string) CommandResult {
	key := name + " " + strings.Join(args, " ")
	r.calls = append(r.calls, key)
	if r.results != nil {
		if result, ok := r.results[key]; ok {
			return result
		}
	}
	return CommandResult{ExitCode: 0}
}

func TestProviderDescriptorReflectsAvailability(t *testing.T) {
	provider := NewProviderWithRunner(&scriptedRunner{
		lookPath: map[string]error{
			defaultCommandName: errors.New("missing"),
		},
	})

	desc := provider.Descriptor()
	require.False(t, desc.Available)
	require.Equal(t, transport.KindAndroidUSBFD, desc.Kind)
}

func TestProviderDiscoverParsesSingleAndMultipleDevices(t *testing.T) {
	provider := NewProviderWithRunner(&scriptedRunner{
		results: map[string]CommandResult{
			defaultCommandName + " -l": {
				Stdout:   "[\"/dev/bus/usb/001/002\", \"/dev/bus/usb/001/003\"]",
				ExitCode: 0,
			},
		},
	})

	devices, err := provider.Discover(context.Background(), transport.DiscoveryRequest{})
	require.NoError(t, err)
	require.Len(t, devices, 2)
	require.Equal(t, "/dev/bus/usb/001/002", devices[0].StableID)
	require.Equal(t, "/dev/bus/usb/001/003", devices[1].StableID)
}

func TestProviderDiscoverReturnsEmptyListWhenNoDevices(t *testing.T) {
	provider := NewProviderWithRunner(&scriptedRunner{
		results: map[string]CommandResult{
			defaultCommandName + " -l": {
				Stdout:   "[]",
				ExitCode: 0,
			},
		},
	})

	devices, err := provider.Discover(context.Background(), transport.DiscoveryRequest{})
	require.NoError(t, err)
	require.Empty(t, devices)
}

func TestProviderPermissionRequestSuccessAndFailure(t *testing.T) {
	provider := NewProviderWithRunner(&scriptedRunner{
		results: map[string]CommandResult{
			defaultCommandName + " -r /dev/bus/usb/001/002": {
				Stdout:   "Access granted",
				ExitCode: 0,
			},
			defaultCommandName + " -r /dev/bus/usb/001/003": {
				Stderr:   "No such device",
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
			},
		},
	})

	granted, err := provider.RequestPermission(context.Background(), transport.PermissionRequest{
		Required: true,
		Device: transport.DiscoveredDevice{
			StableID: "/dev/bus/usb/001/002",
		},
	})
	require.NoError(t, err)
	require.Equal(t, transport.PermissionStateGranted, granted.State)
	require.Contains(t, strings.Join(granted.ProfessionalDetails(), " "), "Access granted")

	denied, err := provider.RequestPermission(context.Background(), transport.PermissionRequest{
		Required: true,
		Device: transport.DiscoveredDevice{
			StableID: "/dev/bus/usb/001/003",
		},
	})
	require.Error(t, err)
	require.Equal(t, transport.PermissionStateUnavailable, denied.State)
	require.Contains(t, denied.Reason, "stale or unavailable")
}

func TestProviderPermissionRequestSupportsCommandHandoff(t *testing.T) {
	runner := &scriptedRunner{
		results: map[string]CommandResult{
			defaultCommandName + " -r -e probe /dev/bus/usb/001/002": {
				Stdout:   "Access granted",
				ExitCode: 0,
			},
		},
	}
	provider := NewProviderWithRunner(runner)

	result, err := provider.RequestPermission(context.Background(), transport.PermissionRequest{
		Required: true,
		Device: transport.DiscoveredDevice{
			StableID: "/dev/bus/usb/001/002",
		},
		Metadata: map[string]string{
			"command": "probe",
		},
	})
	require.NoError(t, err)
	require.Equal(t, transport.PermissionStateGranted, result.State)
	require.NotEmpty(t, runner.calls)
	require.Contains(t, runner.calls[0], "-r -e probe")
}

func TestProviderDiagnosticsReportsTracesAndEndpointAvailability(t *testing.T) {
	oldFD := os.Getenv("TERMUX_USB_FD")
	t.Cleanup(func() {
		_ = os.Setenv("TERMUX_USB_FD", oldFD)
	})
	require.NoError(t, os.Setenv("TERMUX_USB_FD", "17"))

	provider := NewProviderWithRunner(&scriptedRunner{
		results: map[string]CommandResult{
			defaultCommandName + " -l": {
				Stdout:   "[\"/dev/bus/usb/001/002\"]",
				ExitCode: 0,
			},
		},
	})

	report, err := provider.Diagnostics(context.Background(), transport.DiagnosticsRequest{
		Metadata: map[string]string{
			"device_path": "/dev/bus/usb/001/002",
		},
	})
	require.NoError(t, err)
	require.Equal(t, diagnostics.StatusPassed, report.Status)
	require.Len(t, report.Traces, 1)
	require.Equal(t, transport.EndpointExportFileDescriptor, report.SelectedEndpoint.Kind)
	require.Equal(t, 17, report.SelectedEndpoint.FileDescriptor)
	require.Equal(t, "/dev/bus/usb/001/002", report.Device.StableID)
}

func TestProviderDiagnosticsReportsUnsupportedEndpointWithoutFD(t *testing.T) {
	require.NoError(t, os.Unsetenv("TERMUX_USB_FD"))

	provider := NewProviderWithRunner(&scriptedRunner{
		results: map[string]CommandResult{
			defaultCommandName + " -l": {
				Stdout:   "[]",
				ExitCode: 0,
			},
		},
	})

	report, err := provider.Diagnostics(context.Background(), transport.DiagnosticsRequest{})
	require.NoError(t, err)
	require.Equal(t, transport.EndpointExportUnsupported, report.SelectedEndpoint.Kind)
	require.False(t, report.SelectedEndpoint.Supported)
	require.Contains(t, report.SelectedEndpoint.Reason, "TERMUX_USB_FD")
}

func TestProviderProbeReportsMissingFD(t *testing.T) {
	require.NoError(t, os.Unsetenv("TERMUX_USB_FD"))

	report := HelperStreamProbeReportFromEnv()
	require.False(t, report.FDObserved)
	require.Equal(t, transport.StreamObservationUnsupported, report.ReadState)
	require.Contains(t, report.BeginnerSummary(), "TERMUX_USB_FD is not set")
}

func TestProviderProbeReportsInvalidFD(t *testing.T) {
	oldFD := os.Getenv("TERMUX_USB_FD")
	t.Cleanup(func() {
		_ = os.Setenv("TERMUX_USB_FD", oldFD)
	})
	require.NoError(t, os.Setenv("TERMUX_USB_FD", "not-a-number"))

	report := HelperStreamProbeReportFromEnv()
	require.True(t, report.FDObserved)
	require.False(t, report.FDValid)
	require.Equal(t, transport.StreamObservationUnsupported, report.ReadState)
	require.Contains(t, report.BeginnerSummary(), "invalid")
}

func TestProviderProbeReportsInvalidFDArgument(t *testing.T) {
	require.NoError(t, os.Unsetenv("TERMUX_USB_FD"))

	report := HelperStreamProbeReportFromInvocation([]string{"not-a-number"})
	require.True(t, report.FDObserved)
	require.False(t, report.FDValid)
	require.Equal(t, "argument", report.FDSource)
	require.Equal(t, diagnostics.StatusFailed, report.Status)
	require.Contains(t, report.BeginnerSummary(), "invalid")
}

func TestProviderProbeParsesHelperReport(t *testing.T) {
	oldFD := os.Getenv("TERMUX_USB_FD")
	t.Cleanup(func() {
		_ = os.Setenv("TERMUX_USB_FD", oldFD)
	})
	require.NoError(t, os.Setenv("TERMUX_USB_FD", "17"))

	provider := NewProviderWithRunner(&scriptedRunner{
		results: map[string]CommandResult{
			defaultCommandName + " -r -E -e helper /dev/bus/usb/001/002": {
				Stdout:   `{"schema_version":"1","status":"warning","provider":"termuxusb","provider_kind":"android-usb-fd","fd_env_present":true,"fd_observed":true,"fd_valid":true,"fd_inspectable":true,"fd_source":"environment","handoff_mode":"env","helper_args":[],"stream_supported":false,"stream_proven":false,"read_state":"unsupported","write_state":"unsupported","close_state":"unsupported","eof_state":"unsupported","disconnect_state":"unsupported","beginner_summary":"TERMUX_USB_FD observed via environment; stream support remains experimental","professional_details":["fd handoff observed"],"next_step":"add a bounded byte-stream bridge or transport stream adapter"}`,
				ExitCode: 0,
			},
		},
	})

	report, err := provider.Probe(context.Background(), transport.StreamProbeRequest{
		Device: transport.DiscoveredDevice{
			StableID: "/dev/bus/usb/001/002",
		},
		HelperCommand: "helper",
	})
	require.NoError(t, err)
	require.True(t, report.FDObserved)
	require.True(t, report.FDValid)
	require.True(t, report.FDInspectable)
	require.Equal(t, "environment", report.FDSource)
	require.Equal(t, "env", report.HandoffMode)
	require.Equal(t, "termux-usb -r -E -e helper /dev/bus/usb/001/002", report.TermuxUSBCommand)
	require.Equal(t, transport.StreamObservationUnsupported, report.ReadState)
	require.Contains(t, report.BeginnerSummary(), "TERMUX_USB_FD observed")
	require.Contains(t, report.ProfessionalDetails(), "fd handoff observed")
}

func TestHelperSupportsArgumentHandoff(t *testing.T) {
	oldFD := os.Getenv("TERMUX_USB_FD")
	t.Cleanup(func() {
		_ = os.Setenv("TERMUX_USB_FD", oldFD)
	})
	require.NoError(t, os.Unsetenv("TERMUX_USB_FD"))

	fdFile, err := os.CreateTemp(t.TempDir(), "termux-usb-fd-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = fdFile.Close() })
	fdArg := strconv.FormatUint(uint64(fdFile.Fd()), 10)

	report := HelperStreamProbeReportFromInvocation([]string{fdArg})
	require.Equal(t, "argument", report.FDSource)
	require.Equal(t, "argv", report.HandoffMode)
	require.Contains(t, report.BeginnerSummary(), "command-line argument")
}

func TestProviderProbeHandlesMalformedHelperOutput(t *testing.T) {
	oldFD := os.Getenv("TERMUX_USB_FD")
	t.Cleanup(func() {
		_ = os.Setenv("TERMUX_USB_FD", oldFD)
	})
	require.NoError(t, os.Setenv("TERMUX_USB_FD", "17"))

	provider := NewProviderWithRunner(&scriptedRunner{
		results: map[string]CommandResult{
			defaultCommandName + " -r -E -e helper /dev/bus/usb/001/002": {
				Stdout:   "not-json",
				ExitCode: 0,
			},
		},
	})

	report, err := provider.Probe(context.Background(), transport.StreamProbeRequest{
		Device: transport.DiscoveredDevice{
			StableID: "/dev/bus/usb/001/002",
		},
		HelperCommand: "helper",
	})
	require.NoError(t, err)
	require.Equal(t, diagnostics.StatusFailed, report.Status)
	require.Contains(t, report.BeginnerSummary(), "could not be parsed")
	require.Contains(t, strings.Join(report.ProfessionalDetails(), " "), "helper stdout: not-json")
}

func TestProviderProbeHandlesHelperFailure(t *testing.T) {
	oldFD := os.Getenv("TERMUX_USB_FD")
	t.Cleanup(func() {
		_ = os.Setenv("TERMUX_USB_FD", oldFD)
	})
	require.NoError(t, os.Setenv("TERMUX_USB_FD", "17"))

	provider := NewProviderWithRunner(&scriptedRunner{
		results: map[string]CommandResult{
			defaultCommandName + " -r -E -e helper /dev/bus/usb/001/002": {
				Stderr:   "permission denied",
				Stdout:   "No such device",
				ExitCode: 1,
				Err:      errors.New("exit status 1"),
			},
		},
	})

	report, err := provider.Probe(context.Background(), transport.StreamProbeRequest{
		Device: transport.DiscoveredDevice{
			StableID: "/dev/bus/usb/001/002",
		},
		HelperCommand: "helper",
	})
	require.NoError(t, err)
	require.Contains(t, report.Warnings, "exit status 1")
	require.Contains(t, strings.Join(report.ProfessionalDetails(), " "), "permission denied")
	require.Contains(t, report.BeginnerSummary(), "handoff failed")
}

func TestProviderProbeUsesExpectedCommandShape(t *testing.T) {
	runner := &scriptedRunner{
		results: map[string]CommandResult{
			defaultCommandName + " -r -E -e helper /dev/bus/usb/001/002": {
				Stdout:   `{"schema_version":"1","status":"warning","provider":"termuxusb","provider_kind":"android-usb-fd","fd_env_present":true,"fd_observed":true,"fd_valid":true,"fd_inspectable":true,"fd_source":"environment","handoff_mode":"env","helper_args":[],"stream_supported":false,"stream_proven":false,"read_state":"unsupported","write_state":"unsupported","close_state":"unsupported","eof_state":"unsupported","disconnect_state":"unsupported","beginner_summary":"TERMUX_USB_FD observed via environment; stream support remains experimental","professional_details":["fd handoff observed"],"next_step":"add a bounded byte-stream bridge or transport stream adapter"}`,
				ExitCode: 0,
			},
		},
	}
	provider := NewProviderWithRunner(runner)

	report, err := provider.Probe(context.Background(), transport.StreamProbeRequest{
		Device: transport.DiscoveredDevice{
			StableID: "/dev/bus/usb/001/002",
		},
		HelperCommand: "helper",
	})
	require.NoError(t, err)
	require.Equal(t, "termux-usb -r -E -e helper /dev/bus/usb/001/002", report.TermuxUSBCommand)
	require.NotEmpty(t, runner.calls)
	require.Equal(t, defaultCommandName+" -r -E -e helper /dev/bus/usb/001/002", runner.calls[0])
}
