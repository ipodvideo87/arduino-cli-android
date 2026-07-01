package transport

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

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

type CommandTrace struct {
	Command        string   `json:"command,omitempty"`
	Args           []string `json:"args,omitempty"`
	Stdout         string   `json:"stdout,omitempty"`
	Stderr         string   `json:"stderr,omitempty"`
	ExitCode       int      `json:"exit_code,omitempty"`
	Err            string   `json:"error,omitempty"`
	Interpretation string   `json:"interpretation,omitempty"`
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

// StreamSession is the preferred name for a session that can expose a byte stream.
type StreamSession interface {
	TransportSession
	Stream() (TransportStream, error)
}

// ByteStreamSession is retained for compatibility. Prefer StreamSession.
type ByteStreamSession = StreamSession

type ControlLineSession interface {
	TransportSession
	ControlLines() (ControlLineState, error)
	SetControlLines(ControlLineState) error
}

type EndpointExportSession interface {
	TransportSession
	ExportEndpoint() (EndpointExport, error)
}

type StreamObservationState string

const (
	StreamObservationUnknown      StreamObservationState = "unknown"
	StreamObservationUnavailable  StreamObservationState = "unavailable"
	StreamObservationUnsupported  StreamObservationState = "unsupported"
	StreamObservationObserved     StreamObservationState = "observed"
	StreamObservationExperimental StreamObservationState = "experimental"
	StreamObservationPassed       StreamObservationState = "passed"
	StreamObservationFailed       StreamObservationState = "failed"
)

type TransportStreamState string

const (
	TransportStreamStateUnavailable  TransportStreamState = "unavailable"
	TransportStreamStateExperimental TransportStreamState = "experimental"
	TransportStreamStateReady        TransportStreamState = "ready"
	TransportStreamStateActive       TransportStreamState = "active"
	TransportStreamStateClosing      TransportStreamState = "closing"
	TransportStreamStateClosed       TransportStreamState = "closed"
	TransportStreamStateDisconnected TransportStreamState = "disconnected"
	TransportStreamStateFailed       TransportStreamState = "failed"
)

type TransportStreamTimeouts struct {
	Read  time.Duration `json:"read,omitempty"`
	Write time.Duration `json:"write,omitempty"`
	Idle  time.Duration `json:"idle,omitempty"`
}

type TransportStreamCapabilities struct {
	Read         bool `json:"read,omitempty"`
	Write        bool `json:"write,omitempty"`
	Close        bool `json:"close,omitempty"`
	Timeouts     bool `json:"timeouts,omitempty"`
	Cancellation bool `json:"cancellation,omitempty"`
	LastActivity bool `json:"last_activity,omitempty"`
	CloseReason  bool `json:"close_reason,omitempty"`
	EOF          bool `json:"eof,omitempty"`
	Disconnect   bool `json:"disconnect,omitempty"`
	Experimental bool `json:"experimental,omitempty"`
}

type TransportStreamReader interface{ Read([]byte) (int, error) }

type TransportStreamWriter interface{ Write([]byte) (int, error) }

type TransportStreamCloser interface{ Close() error }

type TransportStreamStateReporter interface{ State() TransportStreamState }

type TransportStreamTimeoutController interface {
	Timeouts() TransportStreamTimeouts
	SetTimeouts(TransportStreamTimeouts) error
}

type TransportStreamCancellationController interface{ Cancel(reason string) error }

type TransportStreamActivityReporter interface{ LastActivity() time.Time }

type TransportStreamEOFReporter interface{ EOF() bool }

type TransportStreamDisconnectReporter interface {
	Disconnected() bool
	DisconnectReason() string
}

type TransportStreamCloseReasonReporter interface{ CloseReason() string }

type TransportStream interface {
	TransportStreamReader
	TransportStreamWriter
	TransportStreamCloser
	Capabilities() TransportStreamCapabilities
	Diagnostics() TransportStreamDiagnosticsReport
}

type StreamProbeRequest struct {
	Selection            SelectionRequest  `json:"selection,omitempty"`
	Device               DiscoveredDevice  `json:"device,omitempty"`
	Permission           PermissionResult  `json:"permission,omitempty"`
	HelperCommand        string            `json:"helper_command,omitempty"`
	HandoffMode          string            `json:"handoff_mode,omitempty"`
	ExpectedCapabilities []Capability      `json:"expected_capabilities,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

type StreamValidationRequest struct {
	Selection     SelectionRequest  `json:"selection,omitempty"`
	Device        DiscoveredDevice  `json:"device,omitempty"`
	Permission    PermissionResult  `json:"permission,omitempty"`
	HelperCommand string            `json:"helper_command,omitempty"`
	HandoffMode   string            `json:"handoff_mode,omitempty"`
	ValidateRead  bool              `json:"validate_read,omitempty"`
	ValidateWrite bool              `json:"validate_write,omitempty"`
	Timeout       time.Duration     `json:"timeout,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type TransportStreamDiagnosticsReport struct {
	SchemaVersion    string                  `json:"schema_version,omitempty"`
	Status           diagnostics.Status      `json:"status,omitempty"`
	Provider         string                  `json:"provider,omitempty"`
	ProviderKind     Kind                    `json:"provider_kind,omitempty"`
	Device           DiscoveredDevice        `json:"device,omitempty"`
	State            TransportStreamState    `json:"state,omitempty"`
	StateReason      string                  `json:"state_reason,omitempty"`
	FDEnvPresent     bool                    `json:"fd_env_present,omitempty"`
	FDEnvValue       string                  `json:"fd_env_value,omitempty"`
	FDObserved       bool                    `json:"fd_observed,omitempty"`
	FDValid          bool                    `json:"fd_valid,omitempty"`
	FDInspectable    bool                    `json:"fd_inspectable,omitempty"`
	FDSource         string                  `json:"fd_source,omitempty"`
	HandoffMode      string                  `json:"handoff_mode,omitempty"`
	HelperArgs       []string                `json:"helper_args,omitempty"`
	TermuxUSBCommand string                  `json:"termux_usb_command,omitempty"`
	StreamSupported  bool                    `json:"stream_supported,omitempty"`
	StreamProven     bool                    `json:"stream_proven,omitempty"`
	Timeouts         TransportStreamTimeouts `json:"timeouts,omitempty"`
	BytesRead        int64                   `json:"bytes_read,omitempty"`
	BytesWritten     int64                   `json:"bytes_written,omitempty"`
	LastActivity     time.Time               `json:"last_activity,omitempty"`
	CloseReason      string                  `json:"close_reason,omitempty"`
	DisconnectReason string                  `json:"disconnect_reason,omitempty"`
	ReadState        StreamObservationState  `json:"read_state,omitempty"`
	WriteState       StreamObservationState  `json:"write_state,omitempty"`
	CloseState       StreamObservationState  `json:"close_state,omitempty"`
	EOFState         StreamObservationState  `json:"eof_state,omitempty"`
	DisconnectState  StreamObservationState  `json:"disconnect_state,omitempty"`
	Warnings         []string                `json:"warnings,omitempty"`
	Limitations      []string                `json:"limitations,omitempty"`
	Traces           []CommandTrace          `json:"traces,omitempty"`
	Beginner         string                  `json:"beginner_summary,omitempty"`
	Professional     []string                `json:"professional_details,omitempty"`
	NextStep         string                  `json:"next_step,omitempty"`
	Metadata         map[string]string       `json:"metadata,omitempty"`
}

type TransportStreamValidationReport struct {
	SchemaVersion    string                           `json:"schema_version,omitempty"`
	Status           diagnostics.Status               `json:"status,omitempty"`
	Provider         string                           `json:"provider,omitempty"`
	ProviderKind     Kind                             `json:"provider_kind,omitempty"`
	Device           DiscoveredDevice                 `json:"device,omitempty"`
	ValidateRead     bool                             `json:"validate_read,omitempty"`
	ValidateWrite    bool                             `json:"validate_write,omitempty"`
	Timeout          time.Duration                    `json:"timeout,omitempty"`
	HelperArgs       []string                         `json:"helper_args,omitempty"`
	TermuxUSBCommand string                           `json:"termux_usb_command,omitempty"`
	StreamProbe      TransportStreamDiagnosticsReport `json:"stream_probe,omitempty"`
	StreamStatus     diagnostics.Status               `json:"stream_status,omitempty"`
	ReadBytes        int64                            `json:"read_bytes,omitempty"`
	WriteBytes       int64                            `json:"write_bytes,omitempty"`
	ReadError        string                           `json:"read_error,omitempty"`
	WriteError       string                           `json:"write_error,omitempty"`
	Warnings         []string                         `json:"warnings,omitempty"`
	Limitations      []string                         `json:"limitations,omitempty"`
	Traces           []CommandTrace                   `json:"traces,omitempty"`
	Beginner         string                           `json:"beginner_summary,omitempty"`
	Professional     []string                         `json:"professional_details,omitempty"`
	NextStep         string                           `json:"next_step,omitempty"`
	Metadata         map[string]string                `json:"metadata,omitempty"`
}

func (r TransportStreamValidationReport) BeginnerSummary() string {
	if strings.TrimSpace(r.Beginner) != "" {
		return r.Beginner
	}
	switch r.Status {
	case diagnostics.StatusPassed:
		return "stream validation completed"
	case diagnostics.StatusWarning:
		return "stream validation completed with warnings"
	case diagnostics.StatusFailed:
		return "stream validation failed"
	case diagnostics.StatusSkipped:
		return "stream validation skipped"
	default:
		return "stream validation pending"
	}
}

func (r TransportStreamValidationReport) ProfessionalDetails() []string {
	details := append([]string(nil), r.Professional...)
	if r.Provider != "" {
		details = append(details, "provider: "+r.Provider)
	}
	if r.ProviderKind != "" {
		details = append(details, "provider kind: "+string(r.ProviderKind))
	}
	if r.ValidateRead {
		details = append(details, "validate read: true")
	}
	if r.ValidateWrite {
		details = append(details, "validate write: true")
	}
	if r.Timeout != 0 {
		details = append(details, "timeout: "+r.Timeout.String())
	}
	if r.TermuxUSBCommand != "" {
		details = append(details, "termux-usb command: "+r.TermuxUSBCommand)
	}
	if len(r.HelperArgs) > 0 {
		details = append(details, "helper args: "+strings.Join(r.HelperArgs, " "))
	}
	if r.StreamProbe.HasContent() {
		details = append(details, "stream probe: "+r.StreamProbe.BeginnerSummary())
	}
	if r.StreamProbe.State != "" {
		details = append(details, "stream state: "+string(r.StreamProbe.State))
	}
	if r.StreamStatus != "" {
		details = append(details, "stream status: "+string(r.StreamStatus))
	}
	if r.ReadBytes != 0 || r.WriteBytes != 0 {
		details = append(details, fmt.Sprintf("bytes: read=%d written=%d", r.ReadBytes, r.WriteBytes))
	}
	if r.ReadError != "" {
		details = append(details, "read error: "+r.ReadError)
	}
	if r.WriteError != "" {
		details = append(details, "write error: "+r.WriteError)
	}
	if len(r.Warnings) > 0 {
		details = append(details, "warnings: "+strings.Join(r.Warnings, "; "))
	}
	if len(r.Limitations) > 0 {
		details = append(details, "limitations: "+strings.Join(r.Limitations, "; "))
	}
	if r.NextStep != "" {
		details = append(details, "next step: "+r.NextStep)
	}
	return details
}

type StreamValidator interface {
	Validate(context.Context, StreamValidationRequest) (TransportStreamValidationReport, error)
}

type InterfaceClaimReleaseRequest struct {
	Selection       SelectionRequest  `json:"selection,omitempty"`
	Device          DiscoveredDevice  `json:"device,omitempty"`
	InterfaceNumber int               `json:"interface_number,omitempty"`
	HelperCommand   string            `json:"helper_command,omitempty"`
	HandoffMode     string            `json:"handoff_mode,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

type InterfaceClaimReleaseReport struct {
	SchemaVersion    string             `json:"schema_version,omitempty"`
	Status           diagnostics.Status `json:"status,omitempty"`
	Provider         string             `json:"provider,omitempty"`
	ProviderKind     Kind               `json:"provider_kind,omitempty"`
	Device           DiscoveredDevice   `json:"device,omitempty"`
	InterfaceNumber  int                `json:"interface_number,omitempty"`
	TermuxUSBCommand string             `json:"termux_usb_command,omitempty"`
	ClaimState       string             `json:"claim_state,omitempty"`
	ReleaseState     string             `json:"release_state,omitempty"`
	ClaimError       string             `json:"claim_error,omitempty"`
	ReleaseError     string             `json:"release_error,omitempty"`
	Warnings         []string           `json:"warnings,omitempty"`
	Limitations      []string           `json:"limitations,omitempty"`
	Traces           []CommandTrace     `json:"traces,omitempty"`
	Beginner         string             `json:"beginner_summary,omitempty"`
	Professional     []string           `json:"professional_details,omitempty"`
	NextStep         string             `json:"next_step,omitempty"`
	Metadata         map[string]string  `json:"metadata,omitempty"`
}

func (r InterfaceClaimReleaseReport) BeginnerSummary() string {
	if strings.TrimSpace(r.Beginner) != "" {
		return r.Beginner
	}
	switch r.Status {
	case diagnostics.StatusPassed:
		return "interface claim/release completed"
	case diagnostics.StatusWarning:
		return "interface claim/release completed with warnings"
	case diagnostics.StatusFailed:
		return "interface claim/release failed"
	case diagnostics.StatusSkipped:
		return "interface claim/release skipped"
	default:
		return "interface claim/release pending"
	}
}

func (r InterfaceClaimReleaseReport) ProfessionalDetails() []string {
	details := append([]string(nil), r.Professional...)
	if r.Provider != "" {
		details = append(details, "provider: "+r.Provider)
	}
	if r.ProviderKind != "" {
		details = append(details, "provider kind: "+string(r.ProviderKind))
	}
	details = append(details, fmt.Sprintf("interface number: %d", r.InterfaceNumber))
	if r.TermuxUSBCommand != "" {
		details = append(details, "termux-usb command: "+r.TermuxUSBCommand)
	}
	if r.ClaimState != "" {
		details = append(details, "claim state: "+r.ClaimState)
	}
	if r.ReleaseState != "" {
		details = append(details, "release state: "+r.ReleaseState)
	}
	if r.ClaimError != "" {
		details = append(details, "claim error: "+r.ClaimError)
	}
	if r.ReleaseError != "" {
		details = append(details, "release error: "+r.ReleaseError)
	}
	if len(r.Warnings) > 0 {
		details = append(details, "warnings: "+strings.Join(r.Warnings, "; "))
	}
	if len(r.Limitations) > 0 {
		details = append(details, "limitations: "+strings.Join(r.Limitations, "; "))
	}
	if r.NextStep != "" {
		details = append(details, "next step: "+r.NextStep)
	}
	return details
}

type InterfaceClaimReleaser interface {
	ClaimRelease(context.Context, InterfaceClaimReleaseRequest) (InterfaceClaimReleaseReport, error)
}

func (r TransportStreamDiagnosticsReport) HasContent() bool {
	return strings.TrimSpace(r.SchemaVersion) != "" ||
		r.Status != "" ||
		strings.TrimSpace(r.Provider) != "" ||
		r.ProviderKind != "" ||
		strings.TrimSpace(r.Device.Provider) != "" ||
		strings.TrimSpace(r.Device.DisplayName) != "" ||
		strings.TrimSpace(r.Device.StableID) != "" ||
		r.State != "" ||
		strings.TrimSpace(r.StateReason) != "" ||
		r.FDEnvPresent ||
		strings.TrimSpace(r.FDEnvValue) != "" ||
		r.FDObserved ||
		r.FDValid ||
		r.FDInspectable ||
		strings.TrimSpace(r.FDSource) != "" ||
		strings.TrimSpace(r.HandoffMode) != "" ||
		len(r.HelperArgs) > 0 ||
		strings.TrimSpace(r.TermuxUSBCommand) != "" ||
		r.StreamSupported ||
		r.StreamProven ||
		r.Timeouts != (TransportStreamTimeouts{}) ||
		r.BytesRead != 0 ||
		r.BytesWritten != 0 ||
		!r.LastActivity.IsZero() ||
		strings.TrimSpace(r.CloseReason) != "" ||
		strings.TrimSpace(r.DisconnectReason) != "" ||
		r.ReadState != "" ||
		r.WriteState != "" ||
		r.CloseState != "" ||
		r.EOFState != "" ||
		r.DisconnectState != "" ||
		len(r.Warnings) > 0 ||
		len(r.Limitations) > 0 ||
		len(r.Traces) > 0 ||
		strings.TrimSpace(r.Beginner) != "" ||
		len(r.Professional) > 0 ||
		strings.TrimSpace(r.NextStep) != "" ||
		len(r.Metadata) > 0
}

func (r TransportStreamDiagnosticsReport) BeginnerSummary() string {
	if strings.TrimSpace(r.Beginner) != "" {
		return r.Beginner
	}
	switch r.Status {
	case diagnostics.StatusPassed:
		return "stream probe completed"
	case diagnostics.StatusWarning:
		return "stream probe completed with warnings"
	case diagnostics.StatusFailed:
		return "stream probe failed"
	case diagnostics.StatusSkipped:
		return "stream probe skipped"
	default:
		return "stream probe pending"
	}
}

func (r TransportStreamDiagnosticsReport) ProfessionalDetails() []string {
	details := append([]string(nil), r.Professional...)
	if r.Provider != "" {
		details = append(details, "provider: "+r.Provider)
	}
	if r.ProviderKind != "" {
		details = append(details, "provider kind: "+string(r.ProviderKind))
	}
	if r.State != "" {
		details = append(details, "state: "+string(r.State))
	}
	if r.StateReason != "" {
		details = append(details, "state reason: "+r.StateReason)
	}
	if r.StreamSupported {
		details = append(details, "stream supported: true")
	}
	if r.StreamProven {
		details = append(details, "stream proven: true")
	}
	if r.FDEnvPresent {
		details = append(details, "TERMUX_USB_FD present")
	}
	if r.FDObserved {
		details = append(details, "TERMUX_USB_FD observed")
	}
	if r.FDValid {
		details = append(details, "TERMUX_USB_FD valid")
	}
	if r.FDInspectable {
		details = append(details, "TERMUX_USB_FD inspectable")
	}
	if r.FDSource != "" {
		details = append(details, "fd source: "+r.FDSource)
	}
	if r.HandoffMode != "" {
		details = append(details, "handoff mode: "+r.HandoffMode)
	}
	if len(r.HelperArgs) > 0 {
		details = append(details, "helper args: "+strings.Join(r.HelperArgs, " "))
	}
	if r.TermuxUSBCommand != "" {
		details = append(details, "termux-usb command: "+r.TermuxUSBCommand)
	}
	if r.ReadState != "" {
		details = append(details, "read state: "+string(r.ReadState))
	}
	if r.WriteState != "" {
		details = append(details, "write state: "+string(r.WriteState))
	}
	if r.CloseState != "" {
		details = append(details, "close state: "+string(r.CloseState))
	}
	if r.EOFState != "" {
		details = append(details, "eof state: "+string(r.EOFState))
	}
	if r.DisconnectState != "" {
		details = append(details, "disconnect state: "+string(r.DisconnectState))
	}
	if r.Timeouts.Read != 0 || r.Timeouts.Write != 0 || r.Timeouts.Idle != 0 {
		details = append(details, fmt.Sprintf("timeouts: read=%s write=%s idle=%s", r.Timeouts.Read, r.Timeouts.Write, r.Timeouts.Idle))
	}
	if r.BytesRead != 0 || r.BytesWritten != 0 {
		details = append(details, fmt.Sprintf("bytes: read=%d written=%d", r.BytesRead, r.BytesWritten))
	}
	if !r.LastActivity.IsZero() {
		details = append(details, "last activity: "+r.LastActivity.Format(time.RFC3339Nano))
	}
	if r.CloseReason != "" {
		details = append(details, "close reason: "+r.CloseReason)
	}
	if r.DisconnectReason != "" {
		details = append(details, "disconnect reason: "+r.DisconnectReason)
	}
	if len(r.Warnings) > 0 {
		details = append(details, "warnings: "+strings.Join(r.Warnings, "; "))
	}
	if len(r.Limitations) > 0 {
		details = append(details, "limitations: "+strings.Join(r.Limitations, "; "))
	}
	if r.NextStep != "" {
		details = append(details, "next step: "+r.NextStep)
	}
	return details
}

type StreamProber interface {
	Probe(context.Context, StreamProbeRequest) (TransportStreamDiagnosticsReport, error)
}

type TransportDiagnosticsReport struct {
	SchemaVersion    string                           `json:"schema_version,omitempty"`
	Status           diagnostics.Status               `json:"status,omitempty"`
	Provider         string                           `json:"provider,omitempty"`
	ProviderKind     Kind                             `json:"provider_kind,omitempty"`
	Selected         TransportDescriptor              `json:"selected,omitempty"`
	Alternatives     []TransportDescriptor            `json:"alternatives,omitempty"`
	Device           DiscoveredDevice                 `json:"device,omitempty"`
	Devices          []DiscoveredDevice               `json:"devices,omitempty"`
	DiscoveryStatus  diagnostics.Status               `json:"discovery_status,omitempty"`
	PermissionStatus diagnostics.Status               `json:"permission_status,omitempty"`
	ConnectionStatus diagnostics.Status               `json:"connection_status,omitempty"`
	SelectedEndpoint EndpointExport                   `json:"selected_endpoint,omitempty"`
	StreamProbe      TransportStreamDiagnosticsReport `json:"stream_probe,omitempty"`
	StreamStatus     diagnostics.Status               `json:"stream_status,omitempty"`
	NextStep         string                           `json:"next_step,omitempty"`
	Interfaces       []InterfaceSummary               `json:"interfaces,omitempty"`
	Endpoints        []EndpointSummary                `json:"endpoints,omitempty"`
	Warnings         []string                         `json:"warnings,omitempty"`
	Limitations      []string                         `json:"limitations,omitempty"`
	Traces           []CommandTrace                   `json:"traces,omitempty"`
	Beginner         string                           `json:"beginner_summary,omitempty"`
	Professional     []string                         `json:"professional_details,omitempty"`
	Fields           map[string]string                `json:"fields,omitempty"`
	Metadata         map[string]string                `json:"metadata,omitempty"`
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
	if r.StreamStatus != "" {
		details = append(details, "stream status: "+string(r.StreamStatus))
	}
	if r.StreamProbe.Status != "" || strings.TrimSpace(r.StreamProbe.Beginner) != "" {
		details = append(details, "stream probe: "+r.StreamProbe.BeginnerSummary())
	}
	if r.StreamProbe.State != "" {
		details = append(details, "stream state: "+string(r.StreamProbe.State))
	}
	if r.NextStep != "" {
		details = append(details, "next step: "+r.NextStep)
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
