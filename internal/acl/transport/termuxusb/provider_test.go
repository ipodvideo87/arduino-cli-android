package termuxusb

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/acl/transport"
	"github.com/stretchr/testify/require"
)

type scriptedRunner struct {
	lookPath map[string]error
	results  map[string]CommandResult
	calls    []string
}

var _ transport.StreamSession = (*Session)(nil)

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
	fdFile, err := os.CreateTemp(t.TempDir(), "termux-usb-fd-*")
	require.NoError(t, err)
	oldFD := os.Getenv("TERMUX_USB_FD")
	t.Cleanup(func() {
		_ = os.Setenv("TERMUX_USB_FD", oldFD)
		_ = fdFile.Close()
	})
	require.NoError(t, os.Setenv("TERMUX_USB_FD", strconv.FormatUint(uint64(fdFile.Fd()), 10)))

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
	require.Equal(t, diagnostics.StatusWarning, report.Status)
	require.Len(t, report.Traces, 2)
	require.Equal(t, transport.EndpointExportFileDescriptor, report.SelectedEndpoint.Kind)
	require.Equal(t, int(fdFile.Fd()), report.SelectedEndpoint.FileDescriptor)
	require.Equal(t, "/dev/bus/usb/001/002", report.Device.StableID)
}

func TestHelperUSBTopologyReportUsesInspector(t *testing.T) {
	oldInspector := inspectUSBTopology
	inspectUSBTopology = func(fd int, args []string) (transport.DiscoveredDevice, error) {
		require.NotZero(t, fd)
		require.Contains(t, strings.Join(args, " "), "/dev/bus/usb/001/002")
		return transport.DiscoveredDevice{
			StableID:        "/dev/bus/usb/001/002",
			DisplayName:     "/dev/bus/usb/001/002",
			TransportFamily: transport.TransportFamilyUSBSerial,
			VID:             0x1234,
			PID:             0x5678,
			Manufacturer:    "Acme",
			Product:         "Widget",
			SerialNumber:    "SN123",
			Interfaces: []transport.InterfaceSummary{{
				Number:      2,
				Alternate:   0,
				Class:       "vendor-specific",
				Subclass:    "0xff",
				Protocol:    "0x01",
				Description: "CDC function",
				Endpoints: []transport.EndpointSummary{{
					Address:       0x81,
					Direction:     "in",
					Type:          "bulk",
					MaxPacketSize: 64,
					Usage:         "bulk",
				}},
			}},
			Metadata: map[string]string{
				"claim_release_state": "not_attempted",
				"bridge_state":        "experimental",
			},
		}, nil
	}
	t.Cleanup(func() { inspectUSBTopology = oldInspector })

	fdFile, err := os.CreateTemp(t.TempDir(), "termux-usb-topology-*")
	require.NoError(t, err)
	oldFD := os.Getenv("TERMUX_USB_FD")
	t.Cleanup(func() {
		_ = os.Setenv("TERMUX_USB_FD", oldFD)
		_ = fdFile.Close()
	})
	require.NoError(t, os.Setenv("TERMUX_USB_FD", strconv.FormatUint(uint64(fdFile.Fd()), 10)))

	report := HelperUSBTopologyReportFromInvocation([]string{"/dev/bus/usb/001/002"})
	require.Equal(t, diagnostics.StatusWarning, report.Status)
	require.True(t, report.FDObserved)
	require.True(t, report.FDInspectable)
	require.Equal(t, "experimental", report.BridgeState)
	require.Equal(t, "not_attempted", report.ClaimReleaseState)
	require.Equal(t, uint16(0x1234), report.Device.VID)
	require.Len(t, report.Device.Interfaces, 1)
	require.Len(t, report.Device.Interfaces[0].Endpoints, 1)
	require.Equal(t, "/dev/bus/usb/001/002", report.Device.StableID)
}

func TestProviderDiscoverEnrichesTopologyEvidence(t *testing.T) {
	helperCommand := defaultUSBTopologyHelperCommand()
	provider := NewProviderWithRunner(&scriptedRunner{
		results: map[string]CommandResult{
			defaultCommandName + " -l": {
				Stdout:   "[\"/dev/bus/usb/001/002\"]",
				ExitCode: 0,
			},
			fmt.Sprintf("%s -r -E -e %s /dev/bus/usb/001/002", defaultCommandName, helperCommand): {
				Stdout:   `{"schema_version":"1","status":"warning","provider":"termuxusb","provider_kind":"android-usb-fd","bridge_state":"experimental","claim_release_state":"not_attempted","device":{"stable_id":"/dev/bus/usb/001/002","display_name":"/dev/bus/usb/001/002","transport_family":"usb-serial","vid":4660,"pid":22136,"manufacturer":"Acme","product":"Widget","serial_number":"SN123","interfaces":[{"number":2,"alternate":0,"class":"vendor-specific","subclass":"0xff","protocol":"0x01","description":"CDC function","endpoints":[{"address":129,"direction":"in","type":"bulk","max_packet_size":64,"usage":"bulk"}]}],"metadata":{"bridge_state":"experimental","claim_release_state":"not_attempted","topology_source":"libusb"}},"limitations":["no payload transfers were attempted","claim/release diagnostics were not attempted"],"beginner_summary":"TERMUX_USB_FD observed via environment; USB topology bridge remains experimental","professional_details":["TERMUX_USB_FD was observed and the USB topology could be inspected","descriptors, interfaces, and endpoints were collected without payload transfers"],"next_step":"use the topology evidence to decide whether a later transfer milestone is safe","metadata":{"device_path":"/dev/bus/usb/001/002","helper_args":"","bridge_state":"experimental","claim_release_state":"not_attempted","fd_source":"environment","handoff_mode":"env","topology_source":"libusb"}}`,
				ExitCode: 0,
			},
		},
	})

	devices, err := provider.Discover(context.Background(), transport.DiscoveryRequest{})
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Equal(t, uint16(0x1234), devices[0].VID)
	require.Len(t, devices[0].Interfaces, 1)
	require.Len(t, devices[0].Interfaces[0].Endpoints, 1)
	require.Equal(t, "experimental", devices[0].Metadata["bridge_state"])
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

func TestSessionStreamWrapsFileDescriptorExperimentally(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "termux-usb-stream-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })

	session := &Session{
		endpoint: transport.EndpointExport{
			Kind:           transport.EndpointExportFileDescriptor,
			FileDescriptor: int(file.Fd()),
			Supported:      true,
		},
		report: transport.TransportDiagnosticsReport{
			SchemaVersion: "1",
			Status:        diagnostics.StatusWarning,
			StreamProbe: transport.TransportStreamDiagnosticsReport{
				SchemaVersion: "1",
				Status:        diagnostics.StatusWarning,
				State:         transport.TransportStreamStateExperimental,
				Beginner:      "stream is experimental",
			},
		},
	}

	stream, err := session.Stream()
	require.NoError(t, err)
	require.NotNil(t, stream)
	require.Equal(t, transport.TransportStreamStateExperimental, stream.Diagnostics().State)
	require.True(t, stream.Capabilities().Read)
	require.True(t, stream.Capabilities().Write)
}

func TestSessionStreamRequiresFileDescriptorEndpoint(t *testing.T) {
	session := &Session{}

	stream, err := session.Stream()
	require.Error(t, err)
	require.Nil(t, stream)
	require.Contains(t, err.Error(), "byte-stream export is unavailable")
}

func TestProviderProbeReportsMissingFD(t *testing.T) {
	require.NoError(t, os.Unsetenv("TERMUX_USB_FD"))

	report := HelperStreamProbeReportFromEnv()
	require.False(t, report.FDObserved)
	require.Equal(t, transport.StreamObservationUnsupported, report.ReadState)
	require.Equal(t, transport.TransportStreamStateUnavailable, report.State)
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
	require.Equal(t, transport.TransportStreamStateFailed, report.State)
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
				Stdout:   `{"schema_version":"1","status":"warning","provider":"termuxusb","provider_kind":"android-usb-fd","state":"experimental","fd_env_present":true,"fd_observed":true,"fd_valid":true,"fd_inspectable":true,"fd_source":"environment","handoff_mode":"env","helper_args":[],"stream_supported":false,"stream_proven":false,"read_state":"experimental","write_state":"experimental","close_state":"experimental","eof_state":"experimental","disconnect_state":"experimental","beginner_summary":"TERMUX_USB_FD observed via environment; stream support remains experimental","professional_details":["fd handoff observed"],"next_step":"add a bounded byte-stream bridge or transport stream adapter"}`,
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
	require.Equal(t, transport.TransportStreamStateExperimental, report.State)
	require.Equal(t, transport.StreamObservationExperimental, report.ReadState)
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
	require.Equal(t, transport.TransportStreamStateExperimental, report.State)
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
	require.Equal(t, transport.TransportStreamStateFailed, report.State)
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
	require.Equal(t, transport.TransportStreamStateFailed, report.State)
}

func TestProviderProbeUsesExpectedCommandShape(t *testing.T) {
	runner := &scriptedRunner{
		results: map[string]CommandResult{
			defaultCommandName + " -r -E -e helper /dev/bus/usb/001/002": {
				Stdout:   `{"schema_version":"1","status":"warning","provider":"termuxusb","provider_kind":"android-usb-fd","state":"experimental","fd_env_present":true,"fd_observed":true,"fd_valid":true,"fd_inspectable":true,"fd_source":"environment","handoff_mode":"env","helper_args":[],"stream_supported":false,"stream_proven":false,"read_state":"experimental","write_state":"experimental","close_state":"experimental","eof_state":"experimental","disconnect_state":"experimental","beginner_summary":"TERMUX_USB_FD observed via environment; stream support remains experimental","professional_details":["fd handoff observed"],"next_step":"add a bounded byte-stream bridge or transport stream adapter"}`,
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
	require.Equal(t, transport.TransportStreamStateExperimental, report.State)
}

func TestProviderValidateUsesExpectedCommandShape(t *testing.T) {
	runner := &scriptedRunner{
		results: map[string]CommandResult{
			defaultCommandName + " -r -E -e helper /dev/bus/usb/001/002": {
				Stdout:   `{"schema_version":"1","status":"warning","provider":"termuxusb","provider_kind":"android-usb-fd","device":{"stable_id":"/dev/bus/usb/001/002","display_name":"/dev/bus/usb/001/002"},"validate_read":true,"validate_write":true,"timeout":2000000000,"helper_args":["--validate-read","--validate-write","--timeout","2s"],"termux_usb_command":"termux-usb -r -E -e helper /dev/bus/usb/001/002","stream_probe":{"schema_version":"1","status":"warning","provider":"termuxusb","provider_kind":"android-usb-fd","device":{"stable_id":"/dev/bus/usb/001/002","display_name":"/dev/bus/usb/001/002"},"state":"experimental","beginner_summary":"TERMUX_USB_FD stream validation is experimental","professional_details":["bounded stream validation is diagnostics-only until the live fd path is proven"],"limitations":["byte-stream support remains experimental"]},"stream_status":"warning","beginner_summary":"TERMUX_USB_FD stream validation is experimental","professional_details":["bounded stream validation is diagnostics-only until the live fd path is proven"],"limitations":["byte-stream support remains experimental"],"next_step":"run the helper through termux-usb -r -E to inspect TERMUX_USB_FD","metadata":{"device_path":"/dev/bus/usb/001/002","helper_command":"helper","handoff_mode":"env","termux_usb_command":"termux-usb -r -E -e helper /dev/bus/usb/001/002"}}`,
				ExitCode: 0,
			},
		},
	}
	provider := NewProviderWithRunner(runner)

	report, err := provider.Validate(context.Background(), transport.StreamValidationRequest{
		Device: transport.DiscoveredDevice{
			StableID: "/dev/bus/usb/001/002",
		},
		HelperCommand: "helper",
		ValidateRead:  true,
		ValidateWrite: true,
		Timeout:       2 * time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, "termux-usb -r -E -e helper /dev/bus/usb/001/002", report.TermuxUSBCommand)
	require.NotEmpty(t, runner.calls)
	require.Equal(t, defaultCommandName+" -r -E -e helper /dev/bus/usb/001/002", runner.calls[0])
	require.Equal(t, transport.TransportStreamStateExperimental, report.StreamProbe.State)
	require.True(t, report.ValidateRead)
	require.True(t, report.ValidateWrite)
	require.Equal(t, 2*time.Second, report.Timeout)
}

func TestProviderClaimReleaseUsesExpectedCommandShape(t *testing.T) {
	runner := &scriptedRunner{
		results: map[string]CommandResult{
			defaultCommandName + " -r -E -e helper /dev/bus/usb/001/002": {
				Stdout:   `{"schema_version":"1","status":"warning","provider":"termuxusb","provider_kind":"android-usb-fd","device":{"stable_id":"/dev/bus/usb/001/002","display_name":"/dev/bus/usb/001/002"},"interface_number":0,"claim_state":"claimed","release_state":"released","termux_usb_command":"termux-usb -r -E -e helper /dev/bus/usb/001/002","beginner_summary":"TERMUX_USB_FD observed via environment; interface claim/release remains experimental","professional_details":["TERMUX_USB_FD was observed and the interface could be claimed and released","no payload transfers were attempted"],"next_step":"use the claim/release evidence to decide whether later transfer diagnostics are safe","metadata":{"device_path":"/dev/bus/usb/001/002","helper_command":"helper","handoff_mode":"env","termux_usb_command":"termux-usb -r -E -e helper /dev/bus/usb/001/002"}}`,
				ExitCode: 0,
			},
		},
	}
	provider := NewProviderWithRunner(runner)

	report, err := provider.ClaimRelease(context.Background(), transport.InterfaceClaimReleaseRequest{
		Device: transport.DiscoveredDevice{
			StableID: "/dev/bus/usb/001/002",
		},
		InterfaceNumber: 0,
		HelperCommand:   "helper",
	})
	require.NoError(t, err)
	require.Equal(t, "termux-usb -r -E -e helper /dev/bus/usb/001/002", report.TermuxUSBCommand)
	require.Equal(t, "claimed", report.ClaimState)
	require.Equal(t, "released", report.ReleaseState)
	require.NotEmpty(t, runner.calls)
	require.Equal(t, defaultCommandName+" -r -E -e helper /dev/bus/usb/001/002", runner.calls[0])
}

func TestHelperStreamValidationReportsMissingFD(t *testing.T) {
	require.NoError(t, os.Unsetenv("TERMUX_USB_FD"))

	report := HelperStreamValidationReportFromInvocation(nil, false, false, 0)
	require.Equal(t, diagnostics.StatusFailed, report.Status)
	require.Contains(t, report.Beginner, "TERMUX_USB_FD is not set")
	require.Equal(t, transport.TransportStreamStateUnavailable, report.StreamProbe.State)
	require.Contains(t, report.NextStep, "termux-usb -r -E -e")
}

func TestHelperStreamValidationRunsWriteBeforeRead(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "termux-usb-stream-validate-*")
	require.NoError(t, err)
	defer file.Close()

	oldFD := os.Getenv("TERMUX_USB_FD")
	t.Cleanup(func() { _ = os.Setenv("TERMUX_USB_FD", oldFD) })
	require.NoError(t, os.Setenv("TERMUX_USB_FD", strconv.FormatUint(uint64(file.Fd()), 10)))

	report := HelperStreamValidationReportFromEnv(true, true, 2*time.Second)
	require.Equal(t, diagnostics.StatusWarning, report.Status)
	require.Empty(t, report.WriteError)
	require.Contains(t, report.ReadError, "EOF")
	require.Equal(t, int64(1), report.WriteBytes)
	require.Equal(t, transport.TransportStreamStateClosed, report.StreamProbe.State)
	require.Equal(t, transport.StreamObservationPassed, report.StreamProbe.WriteState)
	require.Equal(t, transport.StreamObservationFailed, report.StreamProbe.ReadState)
}

func TestHelperClaimReleaseReportsMissingFD(t *testing.T) {
	require.NoError(t, os.Unsetenv("TERMUX_USB_FD"))

	report := HelperClaimReleaseReportFromInvocation(nil, 0)
	require.Equal(t, diagnostics.StatusWarning, report.Status)
	require.Equal(t, "unavailable", report.ClaimState)
	require.Equal(t, "unavailable", report.ReleaseState)
	require.Contains(t, report.Beginner, "TERMUX_USB_FD is not set")
}

func TestClassifyWriteProbeState(t *testing.T) {
	require.Equal(t, transport.StreamObservationUnsupported, classifyWriteProbeState(errors.New("write TERMUX_USB_FD: invalid argument")))
	require.Equal(t, transport.StreamObservationFailed, classifyWriteProbeState(errors.New("unexpected transport failure")))
	require.Equal(t, transport.StreamObservationPassed, classifyWriteProbeState(nil))
}
