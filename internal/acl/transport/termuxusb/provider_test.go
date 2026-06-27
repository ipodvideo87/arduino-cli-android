package termuxusb

import (
	"context"
	"errors"
	"os"
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
