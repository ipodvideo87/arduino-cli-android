package transport

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/stretchr/testify/require"
)

type transportProviderFixture struct {
	desc          TransportDescriptor
	discoverFn    func(context.Context, DiscoveryRequest) ([]DiscoveredDevice, error)
	permissionFn  func(context.Context, PermissionRequest) (PermissionResult, error)
	openFn        func(context.Context, OpenRequest) (TransportSession, error)
	diagnosticsFn func(context.Context, DiagnosticsRequest) (TransportDiagnosticsReport, error)
}

func (f transportProviderFixture) Descriptor() TransportDescriptor { return f.desc }

func (f transportProviderFixture) Discover(ctx context.Context, req DiscoveryRequest) ([]DiscoveredDevice, error) {
	if f.discoverFn != nil {
		return f.discoverFn(ctx, req)
	}
	return nil, nil
}

func (f transportProviderFixture) RequestPermission(ctx context.Context, req PermissionRequest) (PermissionResult, error) {
	if f.permissionFn != nil {
		return f.permissionFn(ctx, req)
	}
	return PermissionResult{State: PermissionStateUnavailable}, nil
}

func (f transportProviderFixture) Open(ctx context.Context, req OpenRequest) (TransportSession, error) {
	if f.openFn != nil {
		return f.openFn(ctx, req)
	}
	return nil, errors.New("open not implemented")
}

func (f transportProviderFixture) Diagnostics(ctx context.Context, req DiagnosticsRequest) (TransportDiagnosticsReport, error) {
	if f.diagnosticsFn != nil {
		return f.diagnosticsFn(ctx, req)
	}
	return TransportDiagnosticsReport{}, nil
}

type fixtureSession struct {
	caps     TransportCapabilities
	endpoint EndpointExport
	report   TransportDiagnosticsReport
	closeErr error
	control  ControlLineState
	stream   io.ReadWriteCloser
}

func (s fixtureSession) Close() error { return s.closeErr }

func (s fixtureSession) Capabilities() TransportCapabilities { return s.caps }

func (s fixtureSession) Diagnostics() TransportDiagnosticsReport { return s.report }

func (s fixtureSession) Stream() (io.ReadWriteCloser, error) {
	if s.stream == nil {
		return nil, errors.New("stream unavailable")
	}
	return s.stream, nil
}

func (s fixtureSession) ControlLines() (ControlLineState, error) { return s.control, nil }

func (s fixtureSession) SetControlLines(state ControlLineState) error {
	s.control = state
	return nil
}

func (s fixtureSession) ExportEndpoint() (EndpointExport, error) { return s.endpoint, nil }

var (
	_ TransportSession      = fixtureSession{}
	_ ByteStreamSession     = fixtureSession{}
	_ ControlLineSession    = fixtureSession{}
	_ EndpointExportSession = fixtureSession{}
	_ TransportProvider     = transportProviderFixture{}
	_ TransportDiscoverer   = transportProviderFixture{}
	_ PermissionRequester   = transportProviderFixture{}
	_ SessionOpener         = transportProviderFixture{}
	_ DiagnosticsReporter   = transportProviderFixture{}
)

func successfulSerialProvider() transportProviderFixture {
	return transportProviderFixture{
		desc: TransportDescriptor{
			Kind:         KindNativeSerial,
			Name:         "serial-fixture",
			Provider:     "serial-fixture",
			StableID:     "fixture-serial",
			Family:       string(TransportFamilyUSBSerial),
			Available:    true,
			Priority:     100,
			Capabilities: []Capability{CapabilitySerialIO, CapabilityControlLines, CapabilityLineCoding, CapabilityPTYExport},
		},
		discoverFn: func(_ context.Context, _ DiscoveryRequest) ([]DiscoveredDevice, error) {
			return []DiscoveredDevice{
				{
					Provider:        "serial-fixture",
					StableID:        "fixture-serial",
					DisplayName:     "Fixture Serial",
					TransportFamily: TransportFamilyUSBSerial,
					Permission:      PermissionStateGranted,
					Capabilities:    CapabilitiesFromList(CapabilitySerialIO, CapabilityControlLines, CapabilityLineCoding),
					Interfaces: []InterfaceSummary{{
						Number:      0,
						Class:       "ff",
						Subclass:    "ff",
						Protocol:    "01",
						Description: "serial interface",
						Endpoints: []EndpointSummary{{
							Address:   1,
							Direction: "out",
							Type:      "bulk",
							Usage:     "tx",
						}, {
							Address:   2,
							Direction: "in",
							Type:      "bulk",
							Usage:     "rx",
						}},
					}},
				},
			}, nil
		},
		permissionFn: func(_ context.Context, req PermissionRequest) (PermissionResult, error) {
			return PermissionResult{
				State:       PermissionStateGranted,
				Required:    req.Required,
				Method:      "fixture-dialog",
				UserMessage: "permission granted",
				Professional: []string{
					"permission acquired through fixture dialog",
				},
			}, nil
		},
		openFn: func(_ context.Context, req OpenRequest) (TransportSession, error) {
			return fixtureSession{
				caps: CapabilitiesFromList(CapabilitySerialIO, CapabilityControlLines, CapabilityLineCoding, CapabilityPTYExport),
				report: TransportDiagnosticsReport{
					SchemaVersion: "1",
					Status:        diagnostics.StatusPassed,
					Beginner:      "serial session open",
				},
				endpoint: EndpointExport{
					Kind:         EndpointExportPTY,
					Path:         "/tmp/fixture-serial.pty",
					Supported:    true,
					UserMessage:  "PTY export ready",
					Professional: []string{"fixture PTY endpoint"},
				},
			}, nil
		},
		diagnosticsFn: func(_ context.Context, req DiagnosticsRequest) (TransportDiagnosticsReport, error) {
			return TransportDiagnosticsReport{
				SchemaVersion:    "1",
				Status:           diagnostics.StatusPassed,
				Provider:         "serial-fixture",
				ProviderKind:     KindNativeSerial,
				Selected:         req.Selected.Descriptor,
				DiscoveryStatus:  diagnostics.StatusPassed,
				PermissionStatus: diagnostics.StatusPassed,
				ConnectionStatus: diagnostics.StatusPassed,
				Beginner:         "fixture serial diagnostics",
				Professional:     []string{"fixture provider report", "selection confirmed"},
				Warnings:         []string{"fixture warning"},
			}, nil
		},
	}
}

func permissionRequiredProvider() transportProviderFixture {
	return transportProviderFixture{
		desc: TransportDescriptor{
			Kind:         KindPTY,
			Name:         "permission-fixture",
			Provider:     "permission-fixture",
			StableID:     "fixture-permission",
			Family:       string(TransportFamilyNetwork),
			Available:    true,
			Priority:     90,
			Capabilities: []Capability{CapabilitySerialIO, CapabilityPTYExport, CapabilityPermission},
		},
		permissionFn: func(_ context.Context, _ PermissionRequest) (PermissionResult, error) {
			return PermissionResult{
				State:         PermissionStateRequired,
				Required:      true,
				Method:        "termux-usb",
				Reason:        "user grant needed",
				RetryGuidance: "retry after granting USB permission",
				UserMessage:   "USB permission is required",
				Professional:  []string{"permission dialog must be shown before opening the transport"},
			}, nil
		},
	}
}

func unsupportedProvider() transportProviderFixture {
	return transportProviderFixture{
		desc: TransportDescriptor{
			Kind:         KindFuture,
			Name:         "unsupported-fixture",
			Provider:     "unsupported-fixture",
			StableID:     "fixture-unsupported",
			Available:    false,
			Priority:     1,
			Capabilities: []Capability{CapabilityFuture},
		},
	}
}

func failingProvider() transportProviderFixture {
	return transportProviderFixture{
		desc: TransportDescriptor{
			Kind:         KindRFC2217,
			Name:         "failing-fixture",
			Provider:     "failing-fixture",
			StableID:     "fixture-failing",
			Available:    true,
			Priority:     80,
			Capabilities: []Capability{CapabilitySerialIO, CapabilityRFC2217},
		},
		openFn: func(_ context.Context, _ OpenRequest) (TransportSession, error) {
			return nil, errors.New("open failed")
		},
		diagnosticsFn: func(_ context.Context, _ DiagnosticsRequest) (TransportDiagnosticsReport, error) {
			return TransportDiagnosticsReport{}, errors.New("diagnostics failed")
		},
	}
}

func TestTransportManagerRegistersAndOrdersProviders(t *testing.T) {
	mgr := NewTransportManager(successfulSerialProvider(), permissionRequiredProvider(), failingProvider())

	available := mgr.Available()
	require.Len(t, available, 3)
	require.Equal(t, "serial-fixture", available[0].Name)
	require.Equal(t, "permission-fixture", available[1].Name)
	require.Equal(t, "failing-fixture", available[2].Name)
}

func TestTransportManagerSelectsPreferredProviderByCapabilityAndPreference(t *testing.T) {
	mgr := NewTransportManager(successfulSerialProvider(), permissionRequiredProvider(), failingProvider())

	selected, err := mgr.Select(SelectionRequest{
		RequiredCapabilities: []Capability{CapabilitySerialIO},
		PreferredKinds:       []Kind{KindPTY},
	})
	require.NoError(t, err)
	require.Equal(t, KindPTY, selected.Descriptor.Kind)
	require.Equal(t, "permission-fixture", selected.Descriptor.Name)
	require.Len(t, selected.Alternatives, 2)
	require.Contains(t, selected.Reason, "serial-io")
}

func TestTransportManagerReturnsNoProviderError(t *testing.T) {
	mgr := NewTransportManager()

	_, err := mgr.Select(SelectionRequest{RequiredCapabilities: []Capability{CapabilitySerialIO}})
	require.Error(t, err)

	report, reportErr := mgr.Diagnostics(context.Background(), SelectionRequest{RequiredCapabilities: []Capability{CapabilitySerialIO}})
	require.Error(t, reportErr)
	require.Equal(t, diagnostics.StatusFailed, report.Status)
	require.Equal(t, "no transport satisfies the requested capabilities", report.BeginnerSummary())
	require.NotEmpty(t, report.ProfessionalDetails())
}

func TestTransportManagerPermissionRequiredFlow(t *testing.T) {
	mgr := NewTransportManager(permissionRequiredProvider())

	result, err := mgr.RequestPermission(context.Background(), SelectionRequest{
		RequiredCapabilities: []Capability{CapabilitySerialIO},
		PreferredKinds:       []Kind{KindPTY},
	}, PermissionRequest{
		Required: true,
		Method:   "termux-usb",
	})
	require.NoError(t, err)
	require.Equal(t, PermissionStateRequired, result.State)
	require.True(t, result.Required)
	require.Contains(t, result.BeginnerSummary(), "required")
	require.Contains(t, strings.Join(result.ProfessionalDetails(), " "), "permission")
}

func TestTransportManagerOpenPropagatesProviderFailure(t *testing.T) {
	mgr := NewTransportManager(failingProvider())

	_, err := mgr.Open(context.Background(), SelectionRequest{
		RequiredCapabilities: []Capability{CapabilitySerialIO},
	}, OpenRequest{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "open failed")
}

func TestTransportManagerDiagnosticsSeparatesBeginnerAndProfessionalDetails(t *testing.T) {
	mgr := NewTransportManager(successfulSerialProvider())

	report, err := mgr.Diagnostics(context.Background(), SelectionRequest{
		RequiredCapabilities: []Capability{CapabilitySerialIO},
	})
	require.NoError(t, err)
	require.Equal(t, diagnostics.StatusPassed, report.Status)
	require.Equal(t, "fixture serial diagnostics", report.BeginnerSummary())
	require.Contains(t, strings.Join(report.ProfessionalDetails(), " "), "selected kind")
	require.Contains(t, strings.Join(report.ProfessionalDetails(), " "), "fixture provider report")
	require.Equal(t, "serial-fixture", report.Provider)
	require.NotEmpty(t, report.Fields)
	require.NotEmpty(t, report.Device.DisplayName)
}

func TestTransportManagerDiagnosticsUsesProviderDiagnostics(t *testing.T) {
	mgr := NewTransportManager(successfulSerialProvider())

	report, err := mgr.Diagnostics(context.Background(), SelectionRequest{
		RequiredCapabilities: []Capability{CapabilitySerialIO},
	})
	require.NoError(t, err)
	require.Contains(t, report.Warnings, "fixture warning")
	require.Equal(t, diagnostics.StatusPassed, report.DiscoveryStatus)
	require.Equal(t, diagnostics.StatusPassed, report.PermissionStatus)
	require.Equal(t, diagnostics.StatusPassed, report.ConnectionStatus)
}
