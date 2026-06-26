package transport

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
)

type TransportCapabilities struct {
	Discovery           bool `json:"discovery,omitempty"`
	Permission          bool `json:"permission,omitempty"`
	SerialIO            bool `json:"serial_io,omitempty"`
	ControlLines        bool `json:"control_lines,omitempty"`
	LineCoding          bool `json:"line_coding,omitempty"`
	ResetSignal         bool `json:"reset_signal,omitempty"`
	BulkTransfer        bool `json:"bulk_transfer,omitempty"`
	InterruptTransfer   bool `json:"interrupt_transfer,omitempty"`
	ControlTransfer     bool `json:"control_transfer,omitempty"`
	PTYExport           bool `json:"pty_export,omitempty"`
	RFC2217Export       bool `json:"rfc2217_export,omitempty"`
	UploadEndpoint      bool `json:"upload_endpoint,omitempty"`
	MonitorEndpoint     bool `json:"monitor_endpoint,omitempty"`
	USBHandle           bool `json:"usb_handle,omitempty"`
	DescriptorDiscovery bool `json:"descriptor_discovery,omitempty"`
	ModemControl        bool `json:"modem_control,omitempty"`
	Future              bool `json:"future,omitempty"`
}

func CapabilitiesFromList(caps ...Capability) TransportCapabilities {
	var out TransportCapabilities
	for _, capability := range caps {
		switch capability {
		case CapabilityDiscovery:
			out.Discovery = true
		case CapabilityPermission:
			out.Permission = true
		case CapabilitySerialIO:
			out.SerialIO = true
		case CapabilityControlLines:
			out.ControlLines = true
		case CapabilityLineCoding:
			out.LineCoding = true
		case CapabilityResetSignal:
			out.ResetSignal = true
		case CapabilityBulkTransfer:
			out.BulkTransfer = true
		case CapabilityInterruptTransfer:
			out.InterruptTransfer = true
		case CapabilityControlTransfer:
			out.ControlTransfer = true
		case CapabilityPTYExport:
			out.PTYExport = true
		case CapabilityRFC2217:
			out.RFC2217Export = true
		case CapabilityUploadEndpoint:
			out.UploadEndpoint = true
		case CapabilityMonitorEndpoint:
			out.MonitorEndpoint = true
		case CapabilityUSBHandle:
			out.USBHandle = true
		case CapabilityDescriptorDiscovery:
			out.DescriptorDiscovery = true
		case CapabilityModemControl:
			out.ModemControl = true
		case CapabilityFuture:
			out.Future = true
		}
	}
	return out
}

func (c TransportCapabilities) List() []Capability {
	var caps []Capability
	if c.Discovery {
		caps = append(caps, CapabilityDiscovery)
	}
	if c.Permission {
		caps = append(caps, CapabilityPermission)
	}
	if c.SerialIO {
		caps = append(caps, CapabilitySerialIO)
	}
	if c.ControlLines {
		caps = append(caps, CapabilityControlLines)
	}
	if c.LineCoding {
		caps = append(caps, CapabilityLineCoding)
	}
	if c.ResetSignal {
		caps = append(caps, CapabilityResetSignal)
	}
	if c.BulkTransfer {
		caps = append(caps, CapabilityBulkTransfer)
	}
	if c.InterruptTransfer {
		caps = append(caps, CapabilityInterruptTransfer)
	}
	if c.ControlTransfer {
		caps = append(caps, CapabilityControlTransfer)
	}
	if c.PTYExport {
		caps = append(caps, CapabilityPTYExport)
	}
	if c.RFC2217Export {
		caps = append(caps, CapabilityRFC2217)
	}
	if c.UploadEndpoint {
		caps = append(caps, CapabilityUploadEndpoint)
	}
	if c.MonitorEndpoint {
		caps = append(caps, CapabilityMonitorEndpoint)
	}
	if c.USBHandle {
		caps = append(caps, CapabilityUSBHandle)
	}
	if c.DescriptorDiscovery {
		caps = append(caps, CapabilityDescriptorDiscovery)
	}
	if c.ModemControl {
		caps = append(caps, CapabilityModemControl)
	}
	if c.Future {
		caps = append(caps, CapabilityFuture)
	}
	sort.SliceStable(caps, func(i, j int) bool {
		return caps[i] < caps[j]
	})
	return caps
}

func (c TransportCapabilities) Has(capability Capability) bool {
	switch capability {
	case CapabilityDiscovery:
		return c.Discovery
	case CapabilityPermission:
		return c.Permission
	case CapabilitySerialIO:
		return c.SerialIO
	case CapabilityControlLines:
		return c.ControlLines
	case CapabilityLineCoding:
		return c.LineCoding
	case CapabilityResetSignal:
		return c.ResetSignal
	case CapabilityBulkTransfer:
		return c.BulkTransfer
	case CapabilityInterruptTransfer:
		return c.InterruptTransfer
	case CapabilityControlTransfer:
		return c.ControlTransfer
	case CapabilityPTYExport:
		return c.PTYExport
	case CapabilityRFC2217:
		return c.RFC2217Export
	case CapabilityUploadEndpoint:
		return c.UploadEndpoint
	case CapabilityMonitorEndpoint:
		return c.MonitorEndpoint
	case CapabilityUSBHandle:
		return c.USBHandle
	case CapabilityDescriptorDiscovery:
		return c.DescriptorDiscovery
	case CapabilityModemControl:
		return c.ModemControl
	case CapabilityFuture:
		return c.Future
	default:
		return false
	}
}

type PermissionState string

const (
	PermissionStateUnknown     PermissionState = "unknown"
	PermissionStateRequired    PermissionState = "required"
	PermissionStateGranted     PermissionState = "granted"
	PermissionStateDenied      PermissionState = "denied"
	PermissionStateUnavailable PermissionState = "unavailable"
)

type PermissionRequest struct {
	Selection            SelectionRequest  `json:"selection,omitempty"`
	Required             bool              `json:"required,omitempty"`
	Method               string            `json:"method,omitempty"`
	Reason               string            `json:"reason,omitempty"`
	RetryGuidance        string            `json:"retry_guidance,omitempty"`
	UserMessage          string            `json:"user_message,omitempty"`
	ProfessionalDetails  []string          `json:"professional_details,omitempty"`
	RequiredCapabilities []Capability      `json:"required_capabilities,omitempty"`
	Device               DiscoveredDevice  `json:"device,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

type PermissionResult struct {
	State         PermissionState   `json:"state,omitempty"`
	Required      bool              `json:"required,omitempty"`
	Method        string            `json:"method,omitempty"`
	Reason        string            `json:"reason,omitempty"`
	RetryGuidance string            `json:"retry_guidance,omitempty"`
	UserMessage   string            `json:"user_message,omitempty"`
	Professional  []string          `json:"professional_details,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

func (r PermissionResult) BeginnerSummary() string {
	if strings.TrimSpace(r.UserMessage) != "" {
		return r.UserMessage
	}
	switch r.State {
	case PermissionStateGranted:
		return "permission granted"
	case PermissionStateRequired:
		return "permission required"
	case PermissionStateDenied:
		return "permission denied"
	case PermissionStateUnavailable:
		return "permission unavailable"
	default:
		return "permission state unknown"
	}
}

func (r PermissionResult) ProfessionalDetails() []string {
	details := append([]string(nil), r.Professional...)
	if r.Method != "" {
		details = append(details, "method: "+r.Method)
	}
	if r.Reason != "" {
		details = append(details, "reason: "+r.Reason)
	}
	if r.RetryGuidance != "" {
		details = append(details, "retry guidance: "+r.RetryGuidance)
	}
	return details
}

type EndpointExportKind string

const (
	EndpointExportNativePath      EndpointExportKind = "native-path"
	EndpointExportPTY             EndpointExportKind = "pty"
	EndpointExportRFC2217         EndpointExportKind = "rfc2217"
	EndpointExportFileDescriptor  EndpointExportKind = "file-descriptor"
	EndpointExportInProcessStream EndpointExportKind = "in-process-stream"
	EndpointExportUnsupported     EndpointExportKind = "unsupported"
)

type EndpointExport struct {
	Kind           EndpointExportKind `json:"kind,omitempty"`
	Path           string             `json:"path,omitempty"`
	URL            string             `json:"url,omitempty"`
	FileDescriptor int                `json:"file_descriptor,omitempty"`
	StreamName     string             `json:"stream_name,omitempty"`
	Supported      bool               `json:"supported,omitempty"`
	Reason         string             `json:"reason,omitempty"`
	UserMessage    string             `json:"user_message,omitempty"`
	Professional   []string           `json:"professional_details,omitempty"`
	Metadata       map[string]string  `json:"metadata,omitempty"`
	Stream         any                `json:"-"`
}

func (e EndpointExport) BeginnerSummary() string {
	if strings.TrimSpace(e.UserMessage) != "" {
		return e.UserMessage
	}
	switch e.Kind {
	case EndpointExportNativePath:
		return "native path export ready"
	case EndpointExportPTY:
		return "PTY export ready"
	case EndpointExportRFC2217:
		return "RFC2217 export ready"
	case EndpointExportFileDescriptor:
		return "file descriptor export ready"
	case EndpointExportInProcessStream:
		return "in-process stream export ready"
	default:
		return "endpoint export unavailable"
	}
}

func (e EndpointExport) ProfessionalDetails() []string {
	details := append([]string(nil), e.Professional...)
	if e.Path != "" {
		details = append(details, "path: "+e.Path)
	}
	if e.URL != "" {
		details = append(details, "url: "+e.URL)
	}
	if e.FileDescriptor > 0 {
		details = append(details, fmt.Sprintf("file descriptor: %d", e.FileDescriptor))
	}
	if e.StreamName != "" {
		details = append(details, "stream: "+e.StreamName)
	}
	if e.Reason != "" {
		details = append(details, "reason: "+e.Reason)
	}
	return details
}

type TransportFamily string

const (
	TransportFamilyUnknown   TransportFamily = "unknown"
	TransportFamilyUSBSerial TransportFamily = "usb-serial"
	TransportFamilyCDCACM    TransportFamily = "cdc-acm"
	TransportFamilyDFU       TransportFamily = "dfu"
	TransportFamilyHID       TransportFamily = "hid"
	TransportFamilyCMSISDAP  TransportFamily = "cmsis-dap"
	TransportFamilyJTAG      TransportFamily = "jtag"
	TransportFamilySWD       TransportFamily = "swd"
	TransportFamilyNetwork   TransportFamily = "network"
	TransportFamilyBluetooth TransportFamily = "bluetooth"
	TransportFamilyFuture    TransportFamily = "future"
)

type ControlLineState struct {
	DTR bool `json:"dtr,omitempty"`
	RTS bool `json:"rts,omitempty"`
	CTS bool `json:"cts,omitempty"`
	DSR bool `json:"dsr,omitempty"`
	DCD bool `json:"dcd,omitempty"`
	RI  bool `json:"ri,omitempty"`
}

type EndpointSummary struct {
	Address       uint8    `json:"address,omitempty"`
	Direction     string   `json:"direction,omitempty"`
	Type          string   `json:"type,omitempty"`
	MaxPacketSize int      `json:"max_packet_size,omitempty"`
	Interval      int      `json:"interval,omitempty"`
	Usage         string   `json:"usage,omitempty"`
	Notes         []string `json:"notes,omitempty"`
}

type InterfaceSummary struct {
	Number      int               `json:"number,omitempty"`
	Alternate   int               `json:"alternate,omitempty"`
	Class       string            `json:"class,omitempty"`
	Subclass    string            `json:"subclass,omitempty"`
	Protocol    string            `json:"protocol,omitempty"`
	Description string            `json:"description,omitempty"`
	Endpoints   []EndpointSummary `json:"endpoints,omitempty"`
	Notes       []string          `json:"notes,omitempty"`
}

type DiscoveredDevice struct {
	Provider        string                `json:"provider,omitempty"`
	StableID        string                `json:"stable_id,omitempty"`
	DisplayName     string                `json:"display_name,omitempty"`
	TransportFamily TransportFamily       `json:"transport_family,omitempty"`
	VID             uint16                `json:"vid,omitempty"`
	PID             uint16                `json:"pid,omitempty"`
	Manufacturer    string                `json:"manufacturer,omitempty"`
	Product         string                `json:"product,omitempty"`
	SerialNumber    string                `json:"serial_number,omitempty"`
	Interfaces      []InterfaceSummary    `json:"interfaces,omitempty"`
	Permission      PermissionState       `json:"permission_state,omitempty"`
	Capabilities    TransportCapabilities `json:"capabilities,omitempty"`
	Warnings        []string              `json:"warnings,omitempty"`
	Limitations     []string              `json:"limitations,omitempty"`
	Metadata        map[string]string     `json:"metadata,omitempty"`
}

type DiscoveryRequest struct {
	Selection            SelectionRequest  `json:"selection,omitempty"`
	RequiredCapabilities []Capability      `json:"required_capabilities,omitempty"`
	PreferredKinds       []Kind            `json:"preferred_kinds,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

type OpenRequest struct {
	Selection            SelectionRequest  `json:"selection,omitempty"`
	Device               DiscoveredDevice  `json:"device,omitempty"`
	Permission           PermissionResult  `json:"permission,omitempty"`
	RequiredCapabilities []Capability      `json:"required_capabilities,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

type DiagnosticsRequest struct {
	Selection  SelectionRequest   `json:"selection,omitempty"`
	Selected   TransportSelection `json:"selected,omitempty"`
	Device     *DiscoveredDevice  `json:"device,omitempty"`
	Permission *PermissionResult  `json:"permission,omitempty"`
	Metadata   map[string]string  `json:"metadata,omitempty"`
}

type TransportSession interface {
	Close() error
	Capabilities() TransportCapabilities
	Diagnostics() TransportDiagnosticsReport
}

type ByteStreamSession interface {
	TransportSession
	Stream() (io.ReadWriteCloser, error)
}

type ControlLineSession interface {
	TransportSession
	ControlLines() (ControlLineState, error)
	SetControlLines(ControlLineState) error
}

type EndpointExportSession interface {
	TransportSession
	ExportEndpoint() (EndpointExport, error)
}

type TransportDiagnosticsReport struct {
	SchemaVersion    string                `json:"schema_version,omitempty"`
	Status           diagnostics.Status    `json:"status,omitempty"`
	Provider         string                `json:"provider,omitempty"`
	ProviderKind     Kind                  `json:"provider_kind,omitempty"`
	Selected         TransportDescriptor   `json:"selected,omitempty"`
	Alternatives     []TransportDescriptor `json:"alternatives,omitempty"`
	Device           DiscoveredDevice      `json:"device,omitempty"`
	DiscoveryStatus  diagnostics.Status    `json:"discovery_status,omitempty"`
	PermissionStatus diagnostics.Status    `json:"permission_status,omitempty"`
	ConnectionStatus diagnostics.Status    `json:"connection_status,omitempty"`
	SelectedEndpoint EndpointExport        `json:"selected_endpoint,omitempty"`
	Interfaces       []InterfaceSummary    `json:"interfaces,omitempty"`
	Endpoints        []EndpointSummary     `json:"endpoints,omitempty"`
	Warnings         []string              `json:"warnings,omitempty"`
	Limitations      []string              `json:"limitations,omitempty"`
	Beginner         string                `json:"beginner_summary,omitempty"`
	Professional     []string              `json:"professional_details,omitempty"`
	Fields           map[string]string     `json:"fields,omitempty"`
	Metadata         map[string]string     `json:"metadata,omitempty"`
}

func (r TransportDiagnosticsReport) BeginnerSummary() string {
	if strings.TrimSpace(r.Beginner) != "" {
		return r.Beginner
	}
	switch r.Status {
	case diagnostics.StatusPassed:
		return "transport diagnostics completed"
	case diagnostics.StatusWarning:
		return "transport diagnostics completed with warnings"
	case diagnostics.StatusFailed:
		return "transport diagnostics failed"
	case diagnostics.StatusSkipped:
		return "transport diagnostics skipped"
	default:
		return "transport diagnostics pending"
	}
}

func (r TransportDiagnosticsReport) ProfessionalDetails() []string {
	details := append([]string(nil), r.Professional...)
	if r.Provider != "" {
		details = append(details, "provider: "+r.Provider)
	}
	if r.ProviderKind != "" {
		details = append(details, "provider kind: "+string(r.ProviderKind))
	}
	if r.Selected.Kind != "" {
		details = append(details, "selected kind: "+string(r.Selected.Kind))
	}
	if r.Device.DisplayName != "" {
		details = append(details, "device: "+r.Device.DisplayName)
	}
	if len(r.Interfaces) > 0 {
		details = append(details, fmt.Sprintf("interfaces: %d", len(r.Interfaces)))
	}
	if len(r.Endpoints) > 0 {
		details = append(details, fmt.Sprintf("endpoints: %d", len(r.Endpoints)))
	}
	if len(r.Warnings) > 0 {
		details = append(details, "warnings: "+strings.Join(r.Warnings, "; "))
	}
	if len(r.Limitations) > 0 {
		details = append(details, "limitations: "+strings.Join(r.Limitations, "; "))
	}
	return details
}

type TransportDiscoverer interface {
	Discover(context.Context, DiscoveryRequest) ([]DiscoveredDevice, error)
}

type PermissionRequester interface {
	RequestPermission(context.Context, PermissionRequest) (PermissionResult, error)
}

type SessionOpener interface {
	Open(context.Context, OpenRequest) (TransportSession, error)
}

type DiagnosticsReporter interface {
	Diagnostics(context.Context, DiagnosticsRequest) (TransportDiagnosticsReport, error)
}
