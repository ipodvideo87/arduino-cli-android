package termuxusb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/acl/transport"
)

const (
	defaultCommandName = "termux-usb"
	defaultProviderID  = "termuxusb"
)

type Runner interface {
	LookPath(string) (string, error)
	Run(context.Context, string, ...string) CommandResult
}

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

type Provider struct {
	runner  Runner
	command string
}

type Session struct {
	devicePath string
	permission transport.PermissionResult
	report     transport.TransportDiagnosticsReport
	endpoint   transport.EndpointExport
	caps       transport.TransportCapabilities
	stream     transport.TransportStream
}

func NewProvider() *Provider {
	return &Provider{
		runner:  execRunner{},
		command: defaultCommandName,
	}
}

func NewProviderWithRunner(r Runner) *Provider {
	p := NewProvider()
	if r != nil {
		p.runner = r
	}
	return p
}

func (p *Provider) Descriptor() transport.TransportDescriptor {
	available := p.available()
	caps := transport.CapabilitiesFromList(
		transport.CapabilityDiscovery,
		transport.CapabilityPermission,
		transport.CapabilityUSBHandle,
		transport.CapabilityDescriptorDiscovery,
		transport.CapabilityBulkTransfer,
		transport.CapabilityInterruptTransfer,
		transport.CapabilityControlTransfer,
	)
	return transport.TransportDescriptor{
		SchemaVersion: "1",
		Provider:      defaultProviderID,
		StableID:      defaultProviderID,
		Family:        string(transport.TransportFamilyUSBSerial),
		Kind:          transport.KindAndroidUSBFD,
		Name:          "Termux USB transport",
		Available:     available,
		Priority:      1000,
		Capabilities:  caps.List(),
		CapabilitySet: caps,
		Metadata: map[string]string{
			"command":         p.commandName(),
			"acquire_mode":    "permission-and-diagnostics",
			"endpoint_export": "file-descriptor-or-unsupported",
		},
	}
}

func (p *Provider) Discover(ctx context.Context, req transport.DiscoveryRequest) ([]transport.DiscoveredDevice, error) {
	if !p.available() {
		return nil, nil
	}
	devices, _, _, err := p.listDevices(ctx)
	if err != nil {
		return nil, err
	}
	if len(req.Metadata) == 0 {
		return devices, nil
	}
	return filterDiscoveryResults(devices, req.Metadata), nil
}

func (p *Provider) RequestPermission(ctx context.Context, req transport.PermissionRequest) (transport.PermissionResult, error) {
	devicePath := requestDevicePath(req.Device, req.Metadata)
	if strings.TrimSpace(devicePath) == "" {
		return transport.PermissionResult{
			State:       transport.PermissionStateUnavailable,
			Required:    req.Required,
			Method:      methodForRequest(req),
			Reason:      "device path is required",
			UserMessage: "USB permission cannot be requested without a device path",
			Professional: []string{
				"missing device path",
			},
			Metadata: map[string]string{
				"command": p.commandName(),
			},
		}, errors.New("device path is required")
	}

	args := []string{"-r"}
	method := "termux-usb -r"
	if command := strings.TrimSpace(req.Metadata["command"]); command != "" {
		args = append(args, "-e", command)
		method = "termux-usb -r -e"
	}
	args = append(args, devicePath)

	result := p.runner.Run(ctx, p.commandName(), args...)
	interpretation := interpretPermissionResult(result, devicePath)
	state := permissionStateFromInterpretation(interpretation)
	professional := []string{
		fmt.Sprintf("command: %s %s", p.commandName(), strings.Join(args, " ")),
		fmt.Sprintf("stdout: %s", trimTrace(result.Stdout)),
		fmt.Sprintf("stderr: %s", trimTrace(result.Stderr)),
		fmt.Sprintf("exit code: %d", result.ExitCode),
		"interpretation: " + interpretation,
	}
	if result.Err != nil {
		professional = append(professional, "error: "+result.Err.Error())
	}

	perm := transport.PermissionResult{
		State:        state,
		Required:     req.Required,
		Method:       method,
		Reason:       interpretation,
		UserMessage:  userMessageForPermissionState(state, devicePath),
		Professional: professional,
		Metadata: map[string]string{
			"device_path":    devicePath,
			"command":        p.commandName(),
			"args":           strings.Join(args, " "),
			"stdout":         result.Stdout,
			"stderr":         result.Stderr,
			"exit_code":      strconv.Itoa(result.ExitCode),
			"interpretation": interpretation,
		},
	}
	if perm.State == transport.PermissionStateGranted && req.Metadata["command"] != "" {
		perm.Metadata["handed_off_command"] = req.Metadata["command"]
	}
	return perm, result.Err
}

func (p *Provider) Open(ctx context.Context, req transport.OpenRequest) (transport.TransportSession, error) {
	permission := req.Permission
	if permission.State != transport.PermissionStateGranted {
		acquired, err := p.RequestPermission(ctx, transport.PermissionRequest{
			Selection:            req.Selection,
			Required:             true,
			Method:               "termux-usb -r",
			RequiredCapabilities: req.RequiredCapabilities,
			Device:               req.Device,
			Metadata:             req.Metadata,
		})
		permission = acquired
		if err != nil {
			return nil, err
		}
	}
	if permission.State != transport.PermissionStateGranted {
		return nil, fmt.Errorf("permission not granted: %s", permission.BeginnerSummary())
	}
	return newSession(req.Device, permission, p.endpointExport()), nil
}

func (p *Provider) Validate(ctx context.Context, req transport.StreamValidationRequest) (transport.TransportStreamValidationReport, error) {
	devicePath := requestDevicePath(req.Device, req.Metadata)
	if strings.TrimSpace(devicePath) == "" {
		return transport.TransportStreamValidationReport{
			SchemaVersion: "1",
			Status:        diagnostics.StatusFailed,
			Provider:      defaultProviderID,
			ProviderKind:  transport.KindAndroidUSBFD,
			Beginner:      "stream validation requires a device path",
			Limitations:   []string{"missing device path"},
			NextStep:      "run stream-validate with a concrete USB device path",
			Metadata:      map[string]string{},
		}, errors.New("device path is required")
	}

	helperCommand := strings.TrimSpace(req.HelperCommand)
	if helperCommand == "" {
		helperCommand = defaultStreamValidateHelperCommand(req)
	}
	report := transport.TransportStreamValidationReport{
		SchemaVersion: "1",
		Status:        diagnostics.StatusWarning,
		Provider:      defaultProviderID,
		ProviderKind:  transport.KindAndroidUSBFD,
		Device:        req.Device,
		ValidateRead:  req.ValidateRead,
		ValidateWrite: req.ValidateWrite,
		Timeout:       req.Timeout,
		HelperArgs:    []string{},
		StreamProbe: transport.TransportStreamDiagnosticsReport{
			SchemaVersion:   "1",
			Status:          diagnostics.StatusWarning,
			Provider:        defaultProviderID,
			ProviderKind:    transport.KindAndroidUSBFD,
			Device:          req.Device,
			State:           transport.TransportStreamStateExperimental,
			StateReason:     "experimental stream bridge",
			ReadState:       transport.StreamObservationExperimental,
			WriteState:      transport.StreamObservationExperimental,
			CloseState:      transport.StreamObservationExperimental,
			EOFState:        transport.StreamObservationExperimental,
			DisconnectState: transport.StreamObservationExperimental,
			Beginner:        "TERMUX_USB_FD stream validation is experimental",
			Professional: []string{
				"stream validation is diagnostics-only until the bounded byte-stream path is proven",
			},
			Limitations: []string{
				"byte-stream support remains experimental",
			},
		},
		StreamStatus: diagnostics.StatusWarning,
		Beginner:     "TERMUX_USB_FD stream validation is experimental",
		Professional: []string{
			"stream validation is diagnostics-only until the bounded byte-stream path is proven",
		},
		Limitations: []string{
			"byte-stream support remains experimental",
		},
		NextStep: "run the helper through termux-usb -r -E to inspect TERMUX_USB_FD and validate the stream boundary",
		Metadata: map[string]string{},
	}
	report.Device.StableID = devicePath
	if report.Device.DisplayName == "" {
		report.Device.DisplayName = devicePath
	}
	report.Metadata["device_path"] = devicePath
	report.Metadata["helper_command"] = helperCommand
	report.Metadata["handoff_mode"] = "env"

	if !p.available() {
		report.Status = diagnostics.StatusFailed
		report.StreamProbe.State = transport.TransportStreamStateUnavailable
		report.StreamProbe.StateReason = "termux-usb unavailable"
		report.StreamProbe.Beginner = "termux-usb is unavailable on this host"
		report.Beginner = "termux-usb is unavailable on this host"
		report.Limitations = append(report.Limitations, "termux-usb is not present")
		report.NextStep = "install termux-usb and retry stream validation on native Termux"
		return report, nil
	}

	args := []string{"-r", "-E", "-e", helperCommand, devicePath}
	report.TermuxUSBCommand = formatCommand(p.commandName(), args)
	report.Metadata["termux_usb_command"] = report.TermuxUSBCommand

	result := p.runner.Run(ctx, p.commandName(), args...)
	trace := transport.CommandTrace{
		Command:        p.commandName(),
		Args:           append([]string(nil), args...),
		Stdout:         result.Stdout,
		Stderr:         result.Stderr,
		ExitCode:       result.ExitCode,
		Interpretation: "termux-usb stream validation",
	}
	if result.Err != nil {
		trace.Err = result.Err.Error()
	}

	parsed, parseErr := parseStreamValidationReport(result.Stdout)
	if parseErr != nil {
		report.Traces = []transport.CommandTrace{trace}
		report.Warnings = append(report.Warnings, parseErr.Error())
		if result.Err != nil {
			report.Warnings = append(report.Warnings, result.Err.Error())
		}
		if trimmed := strings.TrimSpace(result.Stderr); trimmed != "" {
			report.Professional = append(report.Professional, "helper stderr: "+trimmed)
		}
		if trimmed := strings.TrimSpace(result.Stdout); trimmed != "" {
			report.Professional = append(report.Professional, "helper stdout: "+trimmed)
		}
		report.Status = diagnostics.StatusFailed
		report.StreamProbe.State = transport.TransportStreamStateFailed
		report.StreamProbe.StateReason = "helper JSON output could not be parsed"
		if result.Err != nil || result.ExitCode != 0 {
			report.Beginner = "termux-usb handoff failed before stream validation JSON was produced"
			report.StreamProbe.Beginner = report.Beginner
			report.Limitations = append(report.Limitations, "termux-usb did not hand off execution to the helper")
			report.NextStep = "run stream-validate through termux-usb -r -E to receive TERMUX_USB_FD"
			return report, nil
		}
		report.Beginner = "stream validation helper output could not be parsed"
		report.StreamProbe.Beginner = report.Beginner
		report.Limitations = append(report.Limitations, "helper JSON output was malformed")
		report.NextStep = "fix the helper output parser before attempting bounded stream validation"
		return report, nil
	}

	report = parsed
	report.Provider = defaultProviderID
	report.ProviderKind = transport.KindAndroidUSBFD
	report.Device.StableID = devicePath
	if report.Device.DisplayName == "" {
		report.Device.DisplayName = devicePath
	}
	if report.Metadata == nil {
		report.Metadata = map[string]string{}
	}
	report.Metadata["device_path"] = devicePath
	report.Metadata["helper_command"] = helperCommand
	report.Metadata["handoff_mode"] = "env"
	report.TermuxUSBCommand = formatCommand(p.commandName(), args)
	report.Metadata["termux_usb_command"] = report.TermuxUSBCommand
	report.Traces = append([]transport.CommandTrace{trace}, report.Traces...)
	if result.Err != nil {
		report.Warnings = append(report.Warnings, result.Err.Error())
	}
	if result.ExitCode != 0 && report.Status == "" {
		report.Status = diagnostics.StatusFailed
	}
	if report.StreamProbe.State == "" {
		report.StreamProbe.State = transport.TransportStreamStateExperimental
	}
	if report.StreamProbe.StateReason == "" {
		report.StreamProbe.StateReason = "experimental stream bridge"
	}
	if report.NextStep == "" {
		report.NextStep = "run the helper through termux-usb -r -E to inspect TERMUX_USB_FD and validate the stream boundary"
	}
	return report, nil
}

func (p *Provider) ClaimRelease(ctx context.Context, req transport.InterfaceClaimReleaseRequest) (transport.InterfaceClaimReleaseReport, error) {
	devicePath := requestDevicePath(req.Device, req.Metadata)
	if strings.TrimSpace(devicePath) == "" {
		return transport.InterfaceClaimReleaseReport{
			SchemaVersion: "1",
			Status:        diagnostics.StatusFailed,
			Provider:      defaultProviderID,
			ProviderKind:  transport.KindAndroidUSBFD,
			Beginner:      "interface claim/release requires a device path",
			Limitations:   []string{"missing device path"},
			NextStep:      "run claim-release with a concrete USB device path",
			Metadata:      map[string]string{},
		}, errors.New("device path is required")
	}
	helperCommand := strings.TrimSpace(req.HelperCommand)
	if helperCommand == "" {
		helperCommand = defaultClaimReleaseHelperCommand(req)
	}
	report := transport.InterfaceClaimReleaseReport{
		SchemaVersion:   "1",
		Status:          diagnostics.StatusWarning,
		Provider:        defaultProviderID,
		ProviderKind:    transport.KindAndroidUSBFD,
		Device:          req.Device,
		InterfaceNumber: req.InterfaceNumber,
		ClaimState:      "not_attempted",
		ReleaseState:    "not_attempted",
		Limitations: []string{
			"claim/release diagnostics are experimental",
			"no payload transfers were attempted",
		},
		Beginner: "interface claim/release validation is experimental",
		Professional: []string{
			"claim/release is diagnostics-only and does not send payload data",
		},
		NextStep: "run the helper through termux-usb -r -E to inspect TERMUX_USB_FD and claim/release behavior",
		Metadata: map[string]string{},
	}
	report.Device.StableID = devicePath
	if report.Device.DisplayName == "" {
		report.Device.DisplayName = devicePath
	}
	report.Metadata["device_path"] = devicePath
	report.Metadata["helper_command"] = helperCommand
	report.Metadata["handoff_mode"] = "env"

	if !p.available() {
		report.Status = diagnostics.StatusFailed
		report.ClaimState = "unavailable"
		report.ReleaseState = "unavailable"
		report.Beginner = "termux-usb is unavailable on this host"
		report.Limitations = append(report.Limitations, "termux-usb is not present")
		report.NextStep = "install termux-usb and retry claim/release validation on native Termux"
		return report, nil
	}

	args := []string{"-r", "-E", "-e", helperCommand, devicePath}
	report.TermuxUSBCommand = formatCommand(p.commandName(), args)
	report.Metadata["termux_usb_command"] = report.TermuxUSBCommand
	result := p.runner.Run(ctx, p.commandName(), args...)
	trace := transport.CommandTrace{
		Command:        p.commandName(),
		Args:           append([]string(nil), args...),
		Stdout:         result.Stdout,
		Stderr:         result.Stderr,
		ExitCode:       result.ExitCode,
		Interpretation: "termux-usb interface claim/release",
	}
	if result.Err != nil {
		trace.Err = result.Err.Error()
	}

	parsed, parseErr := parseUSBClaimReleaseReport(result.Stdout)
	if parseErr != nil {
		report.Traces = []transport.CommandTrace{trace}
		report.Warnings = append(report.Warnings, parseErr.Error())
		if result.Err != nil {
			report.Warnings = append(report.Warnings, result.Err.Error())
		}
		if trimmed := strings.TrimSpace(result.Stderr); trimmed != "" {
			report.Professional = append(report.Professional, "helper stderr: "+trimmed)
		}
		if trimmed := strings.TrimSpace(result.Stdout); trimmed != "" {
			report.Professional = append(report.Professional, "helper stdout: "+trimmed)
		}
		report.Status = diagnostics.StatusFailed
		report.ClaimState = "failed"
		report.ReleaseState = "not_attempted"
		if result.Err != nil || result.ExitCode != 0 {
			report.Beginner = "termux-usb handoff failed before claim/release JSON was produced"
			report.Limitations = append(report.Limitations, "termux-usb did not hand off execution to the helper")
			report.NextStep = "run claim-release through termux-usb -r -E to receive TERMUX_USB_FD"
			return report, nil
		}
		report.Beginner = "claim/release helper output could not be parsed"
		report.Limitations = append(report.Limitations, "helper JSON output was malformed")
		report.NextStep = "fix the helper output parser before attempting interface claim/release validation"
		return report, nil
	}

	report = claimReleaseReportFromHelper(parsed)
	report.Provider = defaultProviderID
	report.ProviderKind = transport.KindAndroidUSBFD
	report.Device.StableID = devicePath
	if report.Device.DisplayName == "" {
		report.Device.DisplayName = devicePath
	}
	if report.Metadata == nil {
		report.Metadata = map[string]string{}
	}
	report.Metadata["device_path"] = devicePath
	report.Metadata["helper_command"] = helperCommand
	report.Metadata["handoff_mode"] = "env"
	report.TermuxUSBCommand = formatCommand(p.commandName(), args)
	report.Metadata["termux_usb_command"] = report.TermuxUSBCommand
	report.Traces = append([]transport.CommandTrace{trace}, report.Traces...)
	if result.Err != nil {
		report.Warnings = append(report.Warnings, result.Err.Error())
	}
	if result.ExitCode != 0 && report.Status == "" {
		report.Status = diagnostics.StatusFailed
	}
	if report.ClaimState == "" {
		report.ClaimState = "not_attempted"
	}
	if report.ReleaseState == "" {
		report.ReleaseState = "not_attempted"
	}
	if report.NextStep == "" {
		report.NextStep = "use the claim/release result to decide whether later transfer diagnostics are safe"
	}
	return report, nil
}

func (p *Provider) Probe(ctx context.Context, req transport.StreamProbeRequest) (transport.TransportStreamDiagnosticsReport, error) {
	devicePath := requestDevicePath(req.Device, req.Metadata)
	if strings.TrimSpace(devicePath) == "" {
		return transport.TransportStreamDiagnosticsReport{
			SchemaVersion: "1",
			Status:        diagnostics.StatusFailed,
			Provider:      defaultProviderID,
			ProviderKind:  transport.KindAndroidUSBFD,
			Beginner:      "fd probe requires a device path",
			Limitations:   []string{"missing device path"},
			NextStep:      "run probe-fd with a concrete USB device path",
			Metadata:      map[string]string{},
		}, errors.New("device path is required")
	}

	report := transport.TransportStreamDiagnosticsReport{
		SchemaVersion: "1",
		Status:        diagnostics.StatusWarning,
		Provider:      defaultProviderID,
		ProviderKind:  transport.KindAndroidUSBFD,
		Device:        req.Device,
		State:         transport.TransportStreamStateExperimental,
		Beginner:      "stream probe has not been proven yet",
		Professional: []string{
			"stream probing is diagnostics-only until a bounded byte-stream bridge is implemented",
		},
		Warnings: []string{},
		Limitations: []string{
			"byte-stream support remains experimental",
		},
		Metadata: map[string]string{},
	}
	report.Device.StableID = devicePath
	if report.Device.DisplayName == "" {
		report.Device.DisplayName = devicePath
	}

	helperCommand := strings.TrimSpace(req.HelperCommand)
	if helperCommand == "" {
		helperCommand = defaultProbeHelperCommand()
	}
	handoffMode := strings.TrimSpace(req.HandoffMode)
	if handoffMode == "" {
		handoffMode = "env"
	}
	report.HandoffMode = handoffMode

	if !p.available() {
		report.Beginner = "termux-usb is unavailable on this host"
		report.State = transport.TransportStreamStateUnavailable
		report.Limitations = append(report.Limitations, "termux-usb is not present")
		report.NextStep = "install termux-usb and retry the fd probe on native Termux"
		return report, nil
	}

	args := []string{"-r", "-E", "-e", helperCommand, devicePath}
	handoffMode = "env"
	report.TermuxUSBCommand = formatCommand(p.commandName(), args)
	if report.Metadata == nil {
		report.Metadata = map[string]string{}
	}
	report.Metadata["handoff_mode"] = handoffMode
	report.Metadata["termux_usb_command"] = report.TermuxUSBCommand
	result := p.runner.Run(ctx, p.commandName(), args...)
	trace := transport.CommandTrace{
		Command:        p.commandName(),
		Args:           append([]string(nil), args...),
		Stdout:         result.Stdout,
		Stderr:         result.Stderr,
		ExitCode:       result.ExitCode,
		Interpretation: "termux-usb fd handoff probe",
	}
	if result.Err != nil {
		trace.Err = result.Err.Error()
	}

	parsed, parseErr := parseStreamProbeReport(result.Stdout)
	if parseErr != nil {
		report.Traces = []transport.CommandTrace{trace}
		report.Warnings = append(report.Warnings, parseErr.Error())
		if result.Err != nil {
			report.Warnings = append(report.Warnings, result.Err.Error())
		}
		if trimmed := strings.TrimSpace(result.Stderr); trimmed != "" {
			report.Professional = append(report.Professional, "helper stderr: "+trimmed)
		}
		if trimmed := strings.TrimSpace(result.Stdout); trimmed != "" {
			report.Professional = append(report.Professional, "helper stdout: "+trimmed)
		}
		if result.Err != nil || result.ExitCode != 0 {
			report.Status = diagnostics.StatusFailed
			report.Beginner = "termux-usb handoff failed before helper JSON was produced"
			report.State = transport.TransportStreamStateFailed
			report.Limitations = append(report.Limitations, "termux-usb did not hand off execution to the helper")
			report.NextStep = "fix the termux-usb handoff path before probing the helper JSON"
			return report, nil
		}
		report.Status = diagnostics.StatusFailed
		report.Beginner = "fd probe helper output could not be parsed"
		report.State = transport.TransportStreamStateFailed
		report.Limitations = append(report.Limitations, "helper JSON output was malformed")
		report.NextStep = "fix the helper output parser before attempting a byte-stream bridge"
		return report, nil
	}

	report = parsed
	report.Provider = defaultProviderID
	report.ProviderKind = transport.KindAndroidUSBFD
	report.Device.StableID = devicePath
	if report.Device.DisplayName == "" {
		report.Device.DisplayName = devicePath
	}
	if report.Metadata == nil {
		report.Metadata = map[string]string{}
	}
	report.Metadata["device_path"] = devicePath
	report.Metadata["helper_command"] = helperCommand
	report.Metadata["handoff_mode"] = handoffMode
	report.HandoffMode = handoffMode
	report.TermuxUSBCommand = formatCommand(p.commandName(), args)
	report.Metadata["termux_usb_command"] = report.TermuxUSBCommand
	report.Traces = append([]transport.CommandTrace{trace}, report.Traces...)
	if result.Err != nil {
		report.Warnings = append(report.Warnings, result.Err.Error())
	}
	if result.ExitCode != 0 && report.Status == "" {
		report.Status = diagnostics.StatusFailed
	}
	if report.FDObserved && report.FDInspectable {
		report.State = transport.TransportStreamStateExperimental
		report.ReadState = transport.StreamObservationExperimental
		report.WriteState = transport.StreamObservationExperimental
		report.CloseState = transport.StreamObservationExperimental
		report.EOFState = transport.StreamObservationExperimental
		report.DisconnectState = transport.StreamObservationExperimental
	} else if report.State == "" {
		report.State = transport.TransportStreamStateUnavailable
	}
	if report.NextStep == "" {
		report.NextStep = "native byte-stream support remains experimental"
	}
	return report, nil
}

func parseStreamValidationReport(raw string) (transport.TransportStreamValidationReport, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return transport.TransportStreamValidationReport{}, errors.New("empty stream validation output")
	}
	var report transport.TransportStreamValidationReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		return transport.TransportStreamValidationReport{}, err
	}
	return report, nil
}

func (p *Provider) Diagnostics(ctx context.Context, req transport.DiagnosticsRequest) (transport.TransportDiagnosticsReport, error) {
	device := transport.DiscoveredDevice{}
	if req.Device != nil {
		device = *req.Device
	}
	streamProbe := buildSessionStreamProbeReport(device, endpointExportProbeReport(), permissionStatus(device.Permission))
	report := transport.TransportDiagnosticsReport{
		SchemaVersion:    "1",
		Status:           diagnostics.StatusWarning,
		Provider:         defaultProviderID,
		ProviderKind:     transport.KindAndroidUSBFD,
		Selected:         p.Descriptor(),
		DiscoveryStatus:  diagnostics.StatusWarning,
		PermissionStatus: diagnostics.StatusSkipped,
		ConnectionStatus: diagnostics.StatusSkipped,
		StreamProbe:      streamProbe,
		StreamStatus:     streamProbe.Status,
		NextStep:         streamProbe.NextStep,
		Beginner:         "Termux USB diagnostics completed with limitations",
		Professional: []string{
			"discovery is implemented through termux-usb -l",
			"permission acquisition is implemented through termux-usb -r",
			"endpoint export is diagnostic-only until a byte stream is implemented",
		},
		Fields:   map[string]string{},
		Metadata: map[string]string{},
	}
	usbAvailable := p.available()
	report.Fields["command"] = p.commandName()
	report.Fields["termux_usb_available"] = strconv.FormatBool(usbAvailable)
	report.Fields["termux_api_available"] = strconv.FormatBool(p.termuxAPIAvailable())
	report.Fields["termux_usb_fd_available"] = strconv.FormatBool(strings.TrimSpace(os.Getenv("TERMUX_USB_FD")) != "")
	report.Fields["selected_kind"] = string(report.Selected.Kind)

	devices, traces, discoveryStatus, discoveryErr := p.collectDiscovery(ctx, req.Metadata)
	report.Devices = devices
	report.Traces = append(report.Traces, traces...)
	report.DiscoveryStatus = discoveryStatus
	if discoveryErr != nil {
		report.Warnings = append(report.Warnings, discoveryErr.Error())
		report.Professional = append(report.Professional, discoveryErr.Error())
		report.Status = diagnostics.StatusFailed
	}

	selectedPath := requestPathFromMetadata(req.Metadata)
	if selectedPath == "" && req.Device != nil && req.Device.StableID != "" {
		selectedPath = req.Device.StableID
	}
	selectedDevice, selectedWarning := selectDevice(devices, selectedPath)
	if selectedDevice != nil {
		report.Device = *selectedDevice
		report.Interfaces = append([]transport.InterfaceSummary(nil), selectedDevice.Interfaces...)
		report.Endpoints = flattenInterfaces(selectedDevice.Interfaces)
		report.PermissionStatus = permissionStatus(selectedDevice.Permission)
		report.Warnings = append(report.Warnings, selectedDevice.Warnings...)
		report.Limitations = append(report.Limitations, selectedDevice.Limitations...)
	} else if selectedPath != "" {
		report.Device = transport.DiscoveredDevice{
			Provider:        defaultProviderID,
			StableID:        selectedPath,
			DisplayName:     filepath.Base(selectedPath),
			TransportFamily: transport.TransportFamilyUSBSerial,
			Permission:      transport.PermissionStateUnknown,
		}
		report.Warnings = append(report.Warnings, selectedWarning)
		report.Limitations = append(report.Limitations, "selected device path was not found during discovery")
		if report.Status != diagnostics.StatusFailed {
			report.Status = diagnostics.StatusWarning
		}
	}

	export := p.endpointExport()
	report.SelectedEndpoint = export
	if export.Kind == transport.EndpointExportUnsupported {
		report.Limitations = append(report.Limitations, export.Reason)
	}
	report.StreamProbe = buildSessionStreamProbeReport(report.Device, export, report.PermissionStatus)
	report.StreamStatus = report.StreamProbe.Status
	report.NextStep = report.StreamProbe.NextStep
	if !usbAvailable {
		report.Beginner = "termux-usb is unavailable on this host"
		if report.Status != diagnostics.StatusFailed {
			report.Status = diagnostics.StatusWarning
		}
	} else if len(report.Devices) == 0 && discoveryErr == nil {
		report.Beginner = "no USB devices were discovered by termux-usb"
		if report.Status != diagnostics.StatusFailed {
			report.Status = diagnostics.StatusWarning
		}
	} else if selectedDevice != nil && selectedPath != "" {
		if len(report.Interfaces) > 0 || len(report.Endpoints) > 0 {
			report.Beginner = "Termux USB diagnostics completed for the selected device with USB topology evidence"
			report.NextStep = "use the USB topology evidence to decide whether claim/release diagnostics are safe"
		} else {
			report.Beginner = "Termux USB diagnostics completed for the selected device"
		}
		if report.Status != diagnostics.StatusFailed {
			report.Status = diagnostics.StatusWarning
		}
	} else if p.available() {
		report.Beginner = "Termux USB diagnostics completed"
		if report.Status != diagnostics.StatusFailed {
			report.Status = diagnostics.StatusWarning
		}
	}
	report.Professional = append(report.Professional, deviceSummaryDetails(report.Devices)...)
	if selectedWarning != "" {
		report.Professional = append(report.Professional, selectedWarning)
	}
	if export.Kind == transport.EndpointExportFileDescriptor {
		report.Professional = append(report.Professional, fmt.Sprintf("TERMUX_USB_FD=%d", export.FileDescriptor))
	} else {
		report.Professional = append(report.Professional, export.Reason)
	}
	if len(report.Traces) == 0 {
		report.Traces = append(report.Traces, syntheticTrace("lookpath "+p.commandName(), nil, "provider availability check"))
	}
	return report, nil
}

func (p *Provider) available() bool {
	_, err := p.runner.LookPath(p.commandName())
	return err == nil
}

func (p *Provider) commandName() string {
	if strings.TrimSpace(p.command) == "" {
		return defaultCommandName
	}
	return p.command
}

func (p *Provider) listDevices(ctx context.Context) ([]transport.DiscoveredDevice, []transport.CommandTrace, diagnostics.Status, error) {
	devices, traces, status, err := p.collectDiscovery(ctx, nil)
	return devices, traces, status, err
}

func (p *Provider) collectDiscovery(ctx context.Context, metadata map[string]string) ([]transport.DiscoveredDevice, []transport.CommandTrace, diagnostics.Status, error) {
	command := p.commandName()
	if !p.available() {
		trace := transport.CommandTrace{
			Command:        command,
			Args:           []string{"-l"},
			Interpretation: "termux-usb is unavailable",
		}
		return nil, []transport.CommandTrace{trace}, diagnostics.StatusWarning, nil
	}
	result := p.runner.Run(ctx, command, "-l")
	trace := transport.CommandTrace{
		Command:        command,
		Args:           []string{"-l"},
		Stdout:         result.Stdout,
		Stderr:         result.Stderr,
		ExitCode:       result.ExitCode,
		Interpretation: "termux-usb discovery",
	}
	if result.Err != nil {
		trace.Err = result.Err.Error()
	}

	devices, parseErr := parseDeviceList(result.Stdout)
	if parseErr != nil {
		trace.Interpretation = "termux-usb discovery output could not be parsed"
	}
	for i := range devices {
		devices[i].Provider = defaultProviderID
		devices[i].TransportFamily = transport.TransportFamilyUSBSerial
		if devices[i].Permission == "" {
			devices[i].Permission = transport.PermissionStateUnknown
		}
		if isZeroCapabilities(devices[i].Capabilities) {
			devices[i].Capabilities = transport.CapabilitiesFromList(
				transport.CapabilityDiscovery,
				transport.CapabilityPermission,
				transport.CapabilityUSBHandle,
				transport.CapabilityDescriptorDiscovery,
				transport.CapabilityBulkTransfer,
				transport.CapabilityInterruptTransfer,
				transport.CapabilityControlTransfer,
			)
		}
		if devices[i].Metadata == nil {
			devices[i].Metadata = map[string]string{}
		}
		devices[i].Metadata["command"] = command
		devices[i].Metadata["source"] = "termux-usb -l"
	}
	status := diagnostics.StatusPassed
	if result.ExitCode != 0 || result.Err != nil || parseErr != nil {
		status = diagnostics.StatusWarning
	}
	if result.Err != nil {
		return devices, []transport.CommandTrace{trace}, diagnostics.StatusFailed, result.Err
	}
	if parseErr != nil && len(devices) == 0 {
		return devices, []transport.CommandTrace{trace}, diagnostics.StatusWarning, parseErr
	}
	if len(devices) == 0 {
		trace.Interpretation = "termux-usb returned no devices"
		status = diagnostics.StatusWarning
	}
	if selected := requestPathFromMetadata(metadata); selected != "" {
		if _, matched := findDevice(devices, selected); !matched {
			status = diagnostics.StatusWarning
		}
	}

	topologyHelper := strings.TrimSpace(metadata["topology_helper_command"])
	if topologyHelper == "" {
		topologyHelper = defaultUSBTopologyHelperCommand()
	}
	topologyTraces := make([]transport.CommandTrace, 0, len(devices))
	for i := range devices {
		topology, topoTrace, topologyErr := p.collectUSBTopology(ctx, devices[i].StableID, topologyHelper)
		topologyTraces = append(topologyTraces, topoTrace)
		devices[i] = mergeUSBTopologyDevice(devices[i], topology.Device)
		devices[i].Warnings = appendUniqueStrings(devices[i].Warnings, topology.Warnings...)
		devices[i].Limitations = appendUniqueStrings(devices[i].Limitations, topology.Limitations...)
		if devices[i].Metadata == nil {
			devices[i].Metadata = map[string]string{}
		}
		if topology.BridgeState != "" {
			devices[i].Metadata["bridge_state"] = topology.BridgeState
		}
		if topology.ClaimReleaseState != "" {
			devices[i].Metadata["claim_release_state"] = topology.ClaimReleaseState
		}
		if topologyErr != nil {
			devices[i].Warnings = appendUniqueStrings(devices[i].Warnings, topologyErr.Error())
			status = diagnostics.StatusWarning
		}
	}
	if len(devices) > 0 {
		status = diagnostics.StatusWarning
	}
	return devices, append([]transport.CommandTrace{trace}, topologyTraces...), status, nil
}

func (p *Provider) termuxAPIAvailable() bool {
	for _, candidate := range []string{"termux-info", "termux-api-start", "termux-usb"} {
		if _, err := p.runner.LookPath(candidate); err == nil {
			return true
		}
	}
	return false
}

func (p *Provider) endpointExport() transport.EndpointExport {
	raw := strings.TrimSpace(os.Getenv("TERMUX_USB_FD"))
	if raw == "" {
		return transport.EndpointExport{
			Kind:        transport.EndpointExportUnsupported,
			Supported:   false,
			Reason:      "TERMUX_USB_FD is not set; file-descriptor handoff is unavailable outside termux-usb -e",
			UserMessage: "file descriptor export is unavailable",
			Professional: []string{
				"run the provider via termux-usb -r -e to receive a USB file descriptor",
			},
		}
	}
	fd, err := strconv.Atoi(raw)
	if err != nil {
		return transport.EndpointExport{
			Kind:        transport.EndpointExportUnsupported,
			Supported:   false,
			Reason:      "TERMUX_USB_FD is not a valid file descriptor",
			UserMessage: "file descriptor export is unavailable",
			Professional: []string{
				"TERMUX_USB_FD could not be parsed",
				"raw value: " + raw,
			},
		}
	}
	return transport.EndpointExport{
		Kind:           transport.EndpointExportFileDescriptor,
		FileDescriptor: fd,
		Supported:      true,
		UserMessage:    "file descriptor export is available",
		Professional: []string{
			"TERMUX_USB_FD is set for the current process",
		},
	}
}

func (p *Provider) collectUSBTopology(ctx context.Context, devicePath, helperCommand string) (USBTopologyReport, transport.CommandTrace, error) {
	args := []string{"-r", "-E", "-e", helperCommand, devicePath}
	result := p.runner.Run(ctx, p.commandName(), args...)
	trace := transport.CommandTrace{
		Command:        p.commandName(),
		Args:           append([]string(nil), args...),
		Stdout:         result.Stdout,
		Stderr:         result.Stderr,
		ExitCode:       result.ExitCode,
		Interpretation: "termux-usb USB topology bridge",
	}
	if result.Err != nil {
		trace.Err = result.Err.Error()
	}
	report, parseErr := parseUSBTopologyReport(result.Stdout)
	if parseErr != nil {
		return USBTopologyReport{}, trace, parseErr
	}
	if report.Metadata == nil {
		report.Metadata = map[string]string{}
	}
	report.Metadata["device_path"] = devicePath
	report.Metadata["helper_command"] = helperCommand
	report.Metadata["termux_usb_command"] = formatCommand(p.commandName(), args)
	if result.Err != nil {
		report.Warnings = append(report.Warnings, result.Err.Error())
	}
	return report, trace, nil
}

func defaultProbeHelperCommand() string {
	return fmt.Sprintf("%s acl transport probe-fd-helper --json", os.Args[0])
}

func defaultStreamValidateHelperCommand(req transport.StreamValidationRequest) string {
	args := []string{"acl", "transport", "stream-validate-helper", "--json"}
	if req.ValidateRead {
		args = append(args, "--validate-read")
	}
	if req.ValidateWrite {
		args = append(args, "--validate-write")
	}
	if req.Timeout > 0 {
		args = append(args, "--timeout", req.Timeout.String())
	}
	return fmt.Sprintf("%s %s", os.Args[0], strings.Join(args, " "))
}

func defaultClaimReleaseHelperCommand(req transport.InterfaceClaimReleaseRequest) string {
	args := []string{"acl", "transport", "claim-release-helper", "--json", "--interface", strconv.Itoa(req.InterfaceNumber)}
	return fmt.Sprintf("%s %s", os.Args[0], strings.Join(args, " "))
}

func formatCommand(name string, args []string) string {
	return strings.TrimSpace(strings.Join(append([]string{name}, args...), " "))
}

func parseStreamProbeReport(raw string) (transport.TransportStreamDiagnosticsReport, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return transport.TransportStreamDiagnosticsReport{}, errors.New("empty stream probe output")
	}
	var report transport.TransportStreamDiagnosticsReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		return transport.TransportStreamDiagnosticsReport{}, err
	}
	return report, nil
}

func HelperStreamProbeReportFromEnv() transport.TransportStreamDiagnosticsReport {
	return HelperStreamProbeReportFromInvocation(nil)
}

func HelperStreamProbeReportFromInvocation(args []string) transport.TransportStreamDiagnosticsReport {
	rawEnv := strings.TrimSpace(os.Getenv("TERMUX_USB_FD"))
	rawArg, argPresent, argMalformed := probeFDFromArgs(args)
	report := transport.TransportStreamDiagnosticsReport{
		SchemaVersion:   "1",
		Status:          diagnostics.StatusWarning,
		Provider:        defaultProviderID,
		ProviderKind:    transport.KindAndroidUSBFD,
		FDEnvPresent:    rawEnv != "",
		FDEnvValue:      rawEnv,
		HelperArgs:      append([]string(nil), args...),
		StreamSupported: false,
		StreamProven:    false,
		ReadState:       transport.StreamObservationUnsupported,
		WriteState:      transport.StreamObservationUnsupported,
		CloseState:      transport.StreamObservationUnsupported,
		EOFState:        transport.StreamObservationUnsupported,
		DisconnectState: transport.StreamObservationUnsupported,
		Metadata:        map[string]string{},
		NextStep:        "run a bounded read/write bridge before claiming byte-stream support",
	}
	report.Metadata["helper_args"] = strings.Join(args, " ")
	switch {
	case rawEnv != "":
		report.FDObserved = true
		report.FDSource = "environment"
		report.HandoffMode = "env"
		report.Metadata["fd_source"] = report.FDSource
		report.Metadata["handoff_mode"] = report.HandoffMode
		fd, err := strconv.Atoi(rawEnv)
		if err != nil {
			report.Status = diagnostics.StatusFailed
			report.State = transport.TransportStreamStateFailed
			report.Beginner = "TERMUX_USB_FD is invalid"
			report.Warnings = []string{"TERMUX_USB_FD could not be parsed"}
			report.Limitations = []string{"fd value is not a valid integer"}
			report.Professional = []string{
				"TERMUX_USB_FD could not be parsed",
				"raw value: " + rawEnv,
			}
			report.NextStep = "fix the helper invocation before probing the stream"
			return report
		}
		report.FDValid = true
		file := os.NewFile(uintptr(fd), "TERMUX_USB_FD")
		if file == nil {
			report.Status = diagnostics.StatusFailed
			report.State = transport.TransportStreamStateFailed
			report.Beginner = "TERMUX_USB_FD could not be wrapped as a file"
			report.Warnings = []string{"os.NewFile returned nil"}
			report.Limitations = []string{"fd wrapping failed"}
			report.NextStep = "debug the Termux fd handoff path"
			return report
		}
		defer file.Close()
		if _, err := file.Stat(); err != nil {
			report.Status = diagnostics.StatusFailed
			report.State = transport.TransportStreamStateFailed
			report.Beginner = "TERMUX_USB_FD is not inspectable"
			report.Warnings = []string{err.Error()}
			report.Limitations = []string{"fd metadata could not be inspected"}
			report.Professional = []string{
				"TERMUX_USB_FD was present but stat failed",
				"fd: " + rawEnv,
			}
			report.NextStep = "verify the fd handoff path before attempting byte-stream work"
			return report
		}
		report.FDInspectable = true
		report.State = transport.TransportStreamStateExperimental
		report.ReadState = transport.StreamObservationExperimental
		report.WriteState = transport.StreamObservationExperimental
		report.CloseState = transport.StreamObservationExperimental
		report.EOFState = transport.StreamObservationExperimental
		report.DisconnectState = transport.StreamObservationExperimental
		report.Beginner = "TERMUX_USB_FD observed via environment; stream support remains experimental"
		report.Professional = []string{
			"TERMUX_USB_FD was observed and inspected successfully",
			"byte-stream read/write behavior is still unproven",
		}
		report.Limitations = []string{
			"no bounded read/write bridge has been implemented",
			"the helper does not attempt destructive device I/O",
		}
		report.NextStep = "add a bounded byte-stream bridge or transport stream adapter"
		return report
	case argPresent && !argMalformed:
		report.FDObserved = true
		report.FDSource = "argument"
		report.HandoffMode = "argv"
		report.Metadata["fd_source"] = report.FDSource
		report.Metadata["handoff_mode"] = report.HandoffMode
		fd, err := strconv.Atoi(rawArg)
		if err != nil {
			report.Status = diagnostics.StatusFailed
			report.State = transport.TransportStreamStateFailed
			report.Beginner = "fd argument is invalid"
			report.Warnings = []string{"file descriptor argument could not be parsed"}
			report.Limitations = []string{"fd argument is not a valid integer"}
			report.Professional = []string{
				"fd argument could not be parsed",
				"raw argument: " + rawArg,
			}
			report.NextStep = "fix the helper invocation before probing the stream"
			return report
		}
		report.FDValid = true
		file := os.NewFile(uintptr(fd), "TERMUX_USB_FD")
		if file == nil {
			report.Status = diagnostics.StatusFailed
			report.State = transport.TransportStreamStateFailed
			report.Beginner = "fd argument could not be wrapped as a file"
			report.Warnings = []string{"os.NewFile returned nil"}
			report.Limitations = []string{"fd wrapping failed"}
			report.NextStep = "debug the Termux fd handoff path"
			return report
		}
		defer file.Close()
		if _, err := file.Stat(); err != nil {
			report.Status = diagnostics.StatusFailed
			report.State = transport.TransportStreamStateFailed
			report.Beginner = "fd argument is not inspectable"
			report.Warnings = []string{err.Error()}
			report.Limitations = []string{"fd metadata could not be inspected"}
			report.Professional = []string{
				"fd argument was present but stat failed",
				"fd: " + rawArg,
			}
			report.NextStep = "verify the fd handoff path before attempting byte-stream work"
			return report
		}
		report.FDInspectable = true
		report.State = transport.TransportStreamStateExperimental
		report.ReadState = transport.StreamObservationExperimental
		report.WriteState = transport.StreamObservationExperimental
		report.CloseState = transport.StreamObservationExperimental
		report.EOFState = transport.StreamObservationExperimental
		report.DisconnectState = transport.StreamObservationExperimental
		report.Beginner = "file descriptor observed via command-line argument; stream support remains experimental"
		report.Professional = []string{
			"fd argument was observed and inspected successfully",
			"byte-stream read/write behavior is still unproven",
		}
		report.Limitations = []string{
			"no bounded read/write bridge has been implemented",
			"the helper does not attempt destructive device I/O",
		}
		report.NextStep = "add a bounded byte-stream bridge or transport stream adapter"
		return report
	case argPresent && argMalformed:
		report.FDObserved = true
		report.FDSource = "argument"
		report.HandoffMode = "argv"
		report.Metadata["fd_source"] = report.FDSource
		report.Metadata["handoff_mode"] = report.HandoffMode
		report.Status = diagnostics.StatusFailed
		report.Beginner = "fd argument is invalid"
		report.Warnings = []string{"file descriptor argument could not be parsed"}
		report.Limitations = []string{"fd argument is not a valid integer"}
		report.Professional = []string{
			"fd argument could not be parsed",
			"raw argument: " + rawArg,
		}
		report.NextStep = "fix the helper invocation before probing the stream"
		return report
	default:
		report.Beginner = "TERMUX_USB_FD is not set"
		report.Limitations = []string{"fd handoff is unavailable outside termux-usb -e"}
		report.Professional = []string{
			"the helper did not observe an fd handoff",
			"run the helper via termux-usb -r -E to inspect TERMUX_USB_FD, or pass an fd argument when using -e",
		}
		report.FDSource = "none"
		report.HandoffMode = "env"
		report.State = transport.TransportStreamStateUnavailable
		report.Metadata["fd_source"] = report.FDSource
		report.Metadata["handoff_mode"] = report.HandoffMode
		return report
	}
}

func HelperStreamValidationReportFromEnv(validateRead, validateWrite bool, timeout time.Duration) transport.TransportStreamValidationReport {
	return HelperStreamValidationReportFromInvocation(nil, validateRead, validateWrite, timeout)
}

func HelperStreamValidationReportFromInvocation(args []string, validateRead, validateWrite bool, timeout time.Duration) transport.TransportStreamValidationReport {
	rawEnv := strings.TrimSpace(os.Getenv("TERMUX_USB_FD"))
	rawArg, argPresent, argMalformed := probeFDFromArgs(args)
	report := transport.TransportStreamValidationReport{
		SchemaVersion: "1",
		Status:        diagnostics.StatusWarning,
		Provider:      defaultProviderID,
		ProviderKind:  transport.KindAndroidUSBFD,
		Device:        transport.DiscoveredDevice{},
		ValidateRead:  validateRead,
		ValidateWrite: validateWrite,
		Timeout:       timeout,
		HelperArgs:    append([]string(nil), args...),
		StreamProbe: transport.TransportStreamDiagnosticsReport{
			SchemaVersion:   "1",
			Status:          diagnostics.StatusWarning,
			Provider:        defaultProviderID,
			ProviderKind:    transport.KindAndroidUSBFD,
			State:           transport.TransportStreamStateExperimental,
			StateReason:     "experimental stream bridge",
			ReadState:       transport.StreamObservationExperimental,
			WriteState:      transport.StreamObservationExperimental,
			CloseState:      transport.StreamObservationExperimental,
			EOFState:        transport.StreamObservationExperimental,
			DisconnectState: transport.StreamObservationExperimental,
			Beginner:        "TERMUX_USB_FD stream validation is experimental",
			Professional: []string{
				"bounded stream validation is diagnostics-only until the live fd path is proven",
			},
			Limitations: []string{
				"byte-stream support remains experimental",
			},
			Metadata: map[string]string{},
		},
		StreamStatus: diagnostics.StatusWarning,
		Beginner:     "TERMUX_USB_FD stream validation is experimental",
		Professional: []string{
			"bounded stream validation is diagnostics-only until the live fd path is proven",
		},
		Limitations: []string{
			"byte-stream support remains experimental",
		},
		NextStep: "run through termux-usb -r -E so the helper can inspect TERMUX_USB_FD",
		Metadata: map[string]string{},
	}
	report.Metadata["helper_args"] = strings.Join(args, " ")
	report.StreamProbe.Metadata["helper_args"] = strings.Join(args, " ")

	switch {
	case rawEnv != "":
		report.StreamProbe.FDEnvPresent = true
		report.StreamProbe.FDEnvValue = rawEnv
		report.StreamProbe.FDObserved = true
		report.StreamProbe.FDSource = "environment"
		report.StreamProbe.HandoffMode = "env"
		report.Metadata["fd_source"] = "environment"
		report.Metadata["handoff_mode"] = "env"
		fd, err := strconv.Atoi(rawEnv)
		if err != nil {
			report.Status = diagnostics.StatusFailed
			report.StreamProbe.Status = diagnostics.StatusFailed
			report.StreamProbe.State = transport.TransportStreamStateFailed
			report.Beginner = "TERMUX_USB_FD is invalid"
			report.StreamProbe.Beginner = report.Beginner
			report.Warnings = []string{"TERMUX_USB_FD could not be parsed"}
			report.Limitations = []string{"fd value is not a valid integer"}
			report.Professional = []string{
				"TERMUX_USB_FD could not be parsed",
				"raw value: " + rawEnv,
			}
			report.NextStep = "fix the helper invocation before probing the stream"
			return report
		}
		report.StreamProbe.FDValid = true
		file := os.NewFile(uintptr(fd), "TERMUX_USB_FD")
		if file == nil {
			report.Status = diagnostics.StatusFailed
			report.StreamProbe.Status = diagnostics.StatusFailed
			report.StreamProbe.State = transport.TransportStreamStateFailed
			report.Beginner = "TERMUX_USB_FD could not be wrapped as a file"
			report.StreamProbe.Beginner = report.Beginner
			report.Warnings = []string{"os.NewFile returned nil"}
			report.Limitations = []string{"fd wrapping failed"}
			report.NextStep = "debug the Termux fd handoff path"
			return report
		}
		defer file.Close()
		if _, err := file.Stat(); err != nil {
			report.Status = diagnostics.StatusFailed
			report.StreamProbe.Status = diagnostics.StatusFailed
			report.StreamProbe.State = transport.TransportStreamStateFailed
			report.Beginner = "TERMUX_USB_FD is not inspectable"
			report.StreamProbe.Beginner = report.Beginner
			report.Warnings = []string{err.Error()}
			report.Limitations = []string{"fd metadata could not be inspected"}
			report.Professional = []string{
				"TERMUX_USB_FD was present but stat failed",
				"fd: " + rawEnv,
			}
			report.NextStep = "verify the fd handoff path before attempting bounded stream validation"
			return report
		}
		report.StreamProbe.FDInspectable = true
		report.StreamProbe.State = transport.TransportStreamStateExperimental
		report.StreamProbe.ReadState = transport.StreamObservationExperimental
		report.StreamProbe.WriteState = transport.StreamObservationExperimental
		report.StreamProbe.CloseState = transport.StreamObservationExperimental
		report.StreamProbe.EOFState = transport.StreamObservationExperimental
		report.StreamProbe.DisconnectState = transport.StreamObservationExperimental
		report.StreamProbe.Beginner = "TERMUX_USB_FD observed via environment; stream validation remains experimental"
		report.StreamProbe.Professional = []string{
			"TERMUX_USB_FD was observed and inspected successfully",
			"bounded read/write behavior is still being validated",
		}
		report.StreamProbe.Limitations = []string{
			"no bounded read/write bridge has been proven on-device yet",
			"the helper does not attempt destructive device I/O",
		}
		report.StreamProbe.NextStep = "add --validate-read and/or --validate-write to exercise the live stream"

		streamReport := report.StreamProbe
		streamReport.Status = diagnostics.StatusWarning
		stream := transport.NewExperimentalTransportStream(file, streamReport)
		if timeout > 0 {
			if timeoutController, ok := stream.(transport.TransportStreamTimeoutController); ok {
				_ = timeoutController.SetTimeouts(transport.TransportStreamTimeouts{Read: timeout, Write: timeout, Idle: timeout})
			}
		}
		if validateWrite {
			n, err := stream.Write([]byte{0})
			report.WriteBytes = int64(n)
			if err != nil {
				report.WriteError = err.Error()
			}
			if n > 0 && report.StreamProbe.WriteState == transport.StreamObservationExperimental {
				report.StreamProbe.WriteState = transport.StreamObservationPassed
			}
			if err != nil {
				report.StreamProbe.WriteState = classifyWriteProbeState(err)
				report.Warnings = append(report.Warnings, "write probe error: "+err.Error())
				if report.StreamProbe.WriteState == transport.StreamObservationUnsupported {
					report.Professional = append(report.Professional, "raw write probe returned invalid argument; TERMUX_USB_FD is not behaving like a generic byte-stream fd")
					report.Limitations = append(report.Limitations, "raw byte-stream write is unsupported on this native Termux fd")
					report.NextStep = "investigate a USB-transfer-oriented bridge instead of raw fd read/write"
				}
			}
		}
		if validateRead {
			if report.WriteError != "" {
				report.StreamProbe.ReadState = transport.StreamObservationUnavailable
				report.Warnings = append(report.Warnings, "read probe skipped after write probe failure")
			} else {
				buf := make([]byte, 1)
				n, err := stream.Read(buf)
				report.ReadBytes = int64(n)
				if err != nil {
					report.ReadError = err.Error()
				}
				if n > 0 && report.StreamProbe.ReadState == transport.StreamObservationExperimental {
					report.StreamProbe.ReadState = transport.StreamObservationPassed
				}
				if err != nil {
					report.StreamProbe.ReadState = transport.StreamObservationFailed
					report.Warnings = append(report.Warnings, "read probe error: "+err.Error())
				}
			}
		}
		_ = stream.Close()
		finalStreamReport := stream.Diagnostics()
		report.StreamProbe.State = finalStreamReport.State
		report.StreamProbe.StateReason = finalStreamReport.StateReason
		report.StreamProbe.CloseState = finalStreamReport.CloseState
		report.StreamProbe.EOFState = finalStreamReport.EOFState
		report.StreamProbe.DisconnectState = finalStreamReport.DisconnectState
		report.StreamProbe.BytesRead = finalStreamReport.BytesRead
		report.StreamProbe.BytesWritten = finalStreamReport.BytesWritten
		report.StreamProbe.LastActivity = finalStreamReport.LastActivity
		report.StreamProbe.CloseReason = finalStreamReport.CloseReason
		report.StreamProbe.DisconnectReason = finalStreamReport.DisconnectReason
		report.StreamProbe.Metadata = finalStreamReport.Metadata
		report.StreamProbe.Traces = finalStreamReport.Traces
		report.StreamStatus = report.StreamProbe.Status
		report.Status = diagnostics.StatusWarning
		report.Beginner = "TERMUX_USB_FD observed via environment; stream validation remains experimental"
		if !validateRead && !validateWrite {
			report.Professional = append(report.Professional, "no byte probes were requested")
			report.NextStep = "add --validate-read and/or --validate-write to exercise the bounded stream"
		} else {
			report.Professional = append(report.Professional, "bounded byte probes completed")
			if report.ReadError != "" || report.WriteError != "" {
				report.NextStep = "inspect the probe error and compare it with native Termux evidence"
			} else {
				report.NextStep = "if native Termux matches these results, keep the stream state experimental until readiness is proven"
			}
		}
		report.TermuxUSBCommand = ""
		report.StreamProbe.TermuxUSBCommand = ""
		return report
	case argPresent && !argMalformed:
		report.StreamProbe.FDObserved = true
		report.StreamProbe.FDSource = "argument"
		report.StreamProbe.HandoffMode = "argv"
		report.Metadata["fd_source"] = "argument"
		report.Metadata["handoff_mode"] = "argv"
		fd, err := strconv.Atoi(rawArg)
		if err != nil {
			report.Status = diagnostics.StatusFailed
			report.StreamProbe.Status = diagnostics.StatusFailed
			report.StreamProbe.State = transport.TransportStreamStateFailed
			report.Beginner = "fd argument is invalid"
			report.StreamProbe.Beginner = report.Beginner
			report.Warnings = []string{"file descriptor argument could not be parsed"}
			report.Limitations = []string{"fd argument is not a valid integer"}
			report.Professional = []string{
				"fd argument could not be parsed",
				"raw argument: " + rawArg,
			}
			report.NextStep = "fix the helper invocation before probing the stream"
			return report
		}
		report.StreamProbe.FDValid = true
		file := os.NewFile(uintptr(fd), "TERMUX_USB_FD")
		if file == nil {
			report.Status = diagnostics.StatusFailed
			report.StreamProbe.Status = diagnostics.StatusFailed
			report.StreamProbe.State = transport.TransportStreamStateFailed
			report.Beginner = "fd argument could not be wrapped as a file"
			report.StreamProbe.Beginner = report.Beginner
			report.Warnings = []string{"os.NewFile returned nil"}
			report.Limitations = []string{"fd wrapping failed"}
			report.NextStep = "debug the Termux fd handoff path"
			return report
		}
		defer file.Close()
		if _, err := file.Stat(); err != nil {
			report.Status = diagnostics.StatusFailed
			report.StreamProbe.Status = diagnostics.StatusFailed
			report.StreamProbe.State = transport.TransportStreamStateFailed
			report.Beginner = "fd argument is not inspectable"
			report.StreamProbe.Beginner = report.Beginner
			report.Warnings = []string{err.Error()}
			report.Limitations = []string{"fd metadata could not be inspected"}
			report.Professional = []string{
				"fd argument was present but stat failed",
				"fd: " + rawArg,
			}
			report.NextStep = "verify the fd handoff path before attempting bounded stream validation"
			return report
		}
		report.StreamProbe.FDInspectable = true
		report.StreamProbe.State = transport.TransportStreamStateExperimental
		report.StreamProbe.ReadState = transport.StreamObservationExperimental
		report.StreamProbe.WriteState = transport.StreamObservationExperimental
		report.StreamProbe.CloseState = transport.StreamObservationExperimental
		report.StreamProbe.EOFState = transport.StreamObservationExperimental
		report.StreamProbe.DisconnectState = transport.StreamObservationExperimental
		report.StreamProbe.Beginner = "file descriptor observed via command-line argument; stream validation remains experimental"
		report.StreamProbe.Professional = []string{
			"fd argument was observed and inspected successfully",
			"bounded read/write behavior is still being validated",
		}
		report.StreamProbe.Limitations = []string{
			"no bounded read/write bridge has been proven on-device yet",
			"the helper does not attempt destructive device I/O",
		}
		report.StreamProbe.NextStep = "add --validate-read and/or --validate-write to exercise the live stream"
		return report
	case argPresent && argMalformed:
		report.StreamProbe.FDObserved = true
		report.StreamProbe.FDSource = "argument"
		report.StreamProbe.HandoffMode = "argv"
		report.Metadata["fd_source"] = "argument"
		report.Metadata["handoff_mode"] = "argv"
		report.Status = diagnostics.StatusFailed
		report.StreamProbe.Status = diagnostics.StatusFailed
		report.StreamProbe.State = transport.TransportStreamStateFailed
		report.Beginner = "fd argument is invalid"
		report.StreamProbe.Beginner = report.Beginner
		report.Warnings = []string{"file descriptor argument could not be parsed"}
		report.Limitations = []string{"fd argument is not a valid integer"}
		report.Professional = []string{
			"fd argument could not be parsed",
			"raw argument: " + rawArg,
		}
		report.NextStep = "fix the helper invocation before probing the stream"
		return report
	default:
		report.Status = diagnostics.StatusFailed
		report.StreamProbe.Status = diagnostics.StatusFailed
		report.StreamProbe.State = transport.TransportStreamStateUnavailable
		report.StreamProbe.StateReason = "TERMUX_USB_FD is not set"
		report.Beginner = "TERMUX_USB_FD is not set; stream validation must run through termux-usb -r -E -e"
		report.StreamProbe.Beginner = report.Beginner
		report.Limitations = []string{"fd handoff is unavailable outside termux-usb -e"}
		report.Professional = []string{
			"the helper did not observe an fd handoff",
			"run the helper via termux-usb -r -E to inspect TERMUX_USB_FD and validate the stream boundary",
		}
		report.NextStep = "run stream-validate through termux-usb -r -E -e so the helper receives TERMUX_USB_FD"
		report.Metadata["fd_source"] = "none"
		report.Metadata["handoff_mode"] = "env"
		return report
	}
}

func probeFDFromArgs(args []string) (string, bool, bool) {
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if trimmed == "" || strings.HasPrefix(trimmed, "-") {
			continue
		}
		if _, err := strconv.Atoi(trimmed); err == nil {
			return trimmed, true, false
		}
		return trimmed, true, true
	}
	return "", false, false
}

func classifyWriteProbeState(err error) transport.StreamObservationState {
	if err == nil {
		return transport.StreamObservationPassed
	}
	lowered := strings.ToLower(err.Error())
	if strings.Contains(lowered, "invalid argument") || strings.Contains(lowered, "bad file descriptor") {
		return transport.StreamObservationUnsupported
	}
	return transport.StreamObservationFailed
}

func buildSessionStreamProbeReport(device transport.DiscoveredDevice, endpoint transport.EndpointExport, permission diagnostics.Status) transport.TransportStreamDiagnosticsReport {
	report := HelperStreamProbeReportFromEnv()
	report.Device = device
	if report.Device.StableID == "" {
		report.Device.StableID = device.StableID
	}
	if report.Device.DisplayName == "" {
		report.Device.DisplayName = device.DisplayName
	}
	report.StreamSupported = false
	report.StreamProven = false
	if report.State == "" {
		report.State = transport.TransportStreamStateUnavailable
	}
	report.Metadata["permission_state"] = string(permission)
	report.Metadata["fd_source"] = report.FDSource
	if endpoint.Kind == transport.EndpointExportFileDescriptor {
		report.Metadata["endpoint"] = "file-descriptor"
		if report.FDObserved && report.FDInspectable {
			report.State = transport.TransportStreamStateExperimental
			report.ReadState = transport.StreamObservationExperimental
			report.WriteState = transport.StreamObservationExperimental
			report.CloseState = transport.StreamObservationExperimental
			report.EOFState = transport.StreamObservationExperimental
			report.DisconnectState = transport.StreamObservationExperimental
		}
		if report.NextStep == "" {
			report.NextStep = "probe-fd can inspect the fd handoff, but stream support remains experimental"
		}
	} else {
		report.Metadata["endpoint"] = "unsupported"
		if report.NextStep == "" {
			report.NextStep = "run probe-fd once TERMUX_USB_FD handoff is available"
		}
	}
	return report
}

func endpointExportProbeReport() transport.EndpointExport {
	return transport.EndpointExport{
		Kind:        transport.EndpointExportUnsupported,
		Supported:   false,
		Reason:      "stream probe has not been run",
		UserMessage: "stream probe has not been run",
		Professional: []string{
			"run acl transport probe-fd to inspect TERMUX_USB_FD handoff",
		},
	}
}

func newSession(device transport.DiscoveredDevice, permission transport.PermissionResult, endpoint transport.EndpointExport) *Session {
	caps := transport.CapabilitiesFromList(
		transport.CapabilityDiscovery,
		transport.CapabilityPermission,
		transport.CapabilityUSBHandle,
		transport.CapabilityDescriptorDiscovery,
		transport.CapabilityBulkTransfer,
		transport.CapabilityInterruptTransfer,
		transport.CapabilityControlTransfer,
	)
	report := transport.TransportDiagnosticsReport{
		SchemaVersion:    "1",
		Status:           diagnostics.StatusWarning,
		Provider:         defaultProviderID,
		ProviderKind:     transport.KindAndroidUSBFD,
		Device:           device,
		DiscoveryStatus:  diagnostics.StatusPassed,
		PermissionStatus: permissionStatus(permission.State),
		ConnectionStatus: diagnostics.StatusWarning,
		SelectedEndpoint: endpoint,
		StreamProbe:      buildSessionStreamProbeReport(device, endpoint, permissionStatus(permission.State)),
		StreamStatus:     diagnostics.StatusWarning,
		NextStep:         "run acl transport probe-fd to inspect TERMUX_USB_FD handoff",
		Beginner:         "Termux USB session is diagnostic-only",
		Professional: []string{
			"the provider does not yet expose a byte stream",
			"endpoint export is limited to file-descriptor handoff when TERMUX_USB_FD is available",
		},
		Fields:   map[string]string{},
		Metadata: map[string]string{},
	}
	if endpoint.Kind == transport.EndpointExportFileDescriptor {
		report.Status = diagnostics.StatusPassed
		report.ConnectionStatus = diagnostics.StatusPassed
		report.Beginner = "Termux USB session is ready for file-descriptor handoff"
		report.NextStep = "probe-fd can now inspect fd handoff evidence"
	}
	report.StreamProbe = buildSessionStreamProbeReport(device, endpoint, permissionStatus(permission.State))
	report.StreamStatus = report.StreamProbe.Status
	return &Session{
		devicePath: device.StableID,
		permission: permission,
		report:     report,
		endpoint:   endpoint,
		caps:       caps,
	}
}

func (s *Session) Close() error {
	if s.stream != nil {
		_ = s.stream.Close()
		s.stream = nil
	}
	return nil
}

func (s *Session) Capabilities() transport.TransportCapabilities { return s.caps }

func (s *Session) Diagnostics() transport.TransportDiagnosticsReport {
	report := s.report
	if s.stream != nil {
		report.StreamProbe = s.stream.Diagnostics()
		report.StreamStatus = report.StreamProbe.Status
		report.NextStep = report.StreamProbe.NextStep
	}
	return report
}

func (s *Session) Stream() (transport.TransportStream, error) {
	if s.stream != nil {
		return s.stream, nil
	}
	if s.endpoint.Kind != transport.EndpointExportFileDescriptor || s.endpoint.FileDescriptor <= 0 {
		return nil, errors.New("byte-stream export is unavailable")
	}
	file := os.NewFile(uintptr(s.endpoint.FileDescriptor), "TERMUX_USB_FD")
	if file == nil {
		return nil, errors.New("TERMUX_USB_FD could not be wrapped as a stream")
	}
	report := s.report.StreamProbe
	report.State = transport.TransportStreamStateExperimental
	report.StreamSupported = false
	report.StreamProven = false
	report.ReadState = transport.StreamObservationExperimental
	report.WriteState = transport.StreamObservationExperimental
	report.CloseState = transport.StreamObservationExperimental
	report.EOFState = transport.StreamObservationExperimental
	report.DisconnectState = transport.StreamObservationExperimental
	report.Beginner = "TERMUX_USB_FD stream is experimental"
	report.Professional = append(report.Professional,
		"the provider can wrap the handed-off fd, but read/write behavior is still experimental",
	)
	s.report.StreamProbe = report
	s.report.StreamStatus = diagnostics.StatusWarning
	s.report.NextStep = "validate bounded read/write behavior before claiming stream support"
	s.stream = transport.NewExperimentalTransportStream(file, report)
	return s.stream, nil
}

func (s *Session) ExportEndpoint() (transport.EndpointExport, error) { return s.endpoint, nil }

func (p *Provider) permissionFromDevice(device transport.DiscoveredDevice) transport.PermissionResult {
	if device.Permission != "" {
		return transport.PermissionResult{State: device.Permission}
	}
	return transport.PermissionResult{State: transport.PermissionStateUnknown}
}

func (p *Provider) commandTrace(args []string, result CommandResult, interpretation string) transport.CommandTrace {
	trace := transport.CommandTrace{
		Command:        p.commandName(),
		Args:           append([]string(nil), args...),
		Stdout:         result.Stdout,
		Stderr:         result.Stderr,
		ExitCode:       result.ExitCode,
		Interpretation: interpretation,
	}
	if result.Err != nil {
		trace.Err = result.Err.Error()
	}
	return trace
}

func methodForRequest(req transport.PermissionRequest) string {
	if strings.TrimSpace(req.Metadata["command"]) != "" {
		return "termux-usb -r -e"
	}
	return "termux-usb -r"
}

func permissionStateFromInterpretation(interpretation string) transport.PermissionState {
	switch {
	case strings.Contains(interpretation, "granted"):
		return transport.PermissionStateGranted
	case strings.Contains(interpretation, "denied"):
		return transport.PermissionStateDenied
	case strings.Contains(interpretation, "unavailable"):
		return transport.PermissionStateUnavailable
	default:
		return transport.PermissionStateUnknown
	}
}

func interpretPermissionResult(result CommandResult, devicePath string) string {
	joined := strings.ToLower(result.Stdout + "\n" + result.Stderr)
	switch {
	case strings.Contains(joined, "no such device"):
		return fmt.Sprintf("device %s is stale or unavailable", devicePath)
	case strings.Contains(joined, "permission denied"):
		return "permission denied by Android"
	case strings.Contains(joined, "access granted"):
		return "permission granted by termux-usb"
	case result.ExitCode == 0:
		return "permission granted by termux-usb"
	case result.Err != nil:
		return result.Err.Error()
	default:
		return fmt.Sprintf("termux-usb exited with code %d", result.ExitCode)
	}
}

func userMessageForPermissionState(state transport.PermissionState, devicePath string) string {
	switch state {
	case transport.PermissionStateGranted:
		return fmt.Sprintf("USB permission granted for %s", devicePath)
	case transport.PermissionStateDenied:
		return fmt.Sprintf("USB permission was denied for %s", devicePath)
	case transport.PermissionStateUnavailable:
		return fmt.Sprintf("USB permission is unavailable for %s", devicePath)
	default:
		return fmt.Sprintf("USB permission state is unknown for %s", devicePath)
	}
}

func trimTrace(value string) string {
	return strings.TrimSpace(value)
}

func requestDevicePath(device transport.DiscoveredDevice, metadata map[string]string) string {
	if strings.TrimSpace(device.StableID) != "" {
		return device.StableID
	}
	for _, key := range []string{"device_path", "path", "usb_path", "stable_id"} {
		if value := strings.TrimSpace(metadata[key]); value != "" {
			return value
		}
	}
	return ""
}

func requestPathFromMetadata(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}
	for _, key := range []string{"device_path", "path", "usb_path", "stable_id"} {
		if value := strings.TrimSpace(metadata[key]); value != "" {
			return value
		}
	}
	return ""
}

func filterDiscoveryResults(devices []transport.DiscoveredDevice, metadata map[string]string) []transport.DiscoveredDevice {
	selected := requestPathFromMetadata(metadata)
	if selected == "" {
		return devices
	}
	if device, ok := findDevice(devices, selected); ok {
		return []transport.DiscoveredDevice{device}
	}
	return devices
}

func findDevice(devices []transport.DiscoveredDevice, path string) (transport.DiscoveredDevice, bool) {
	for _, device := range devices {
		if device.StableID == path || device.DisplayName == path || strings.EqualFold(filepath.Base(device.StableID), filepath.Base(path)) {
			return device, true
		}
	}
	return transport.DiscoveredDevice{}, false
}

func selectDevice(devices []transport.DiscoveredDevice, path string) (*transport.DiscoveredDevice, string) {
	if path == "" {
		if len(devices) == 0 {
			return nil, ""
		}
		return &devices[0], ""
	}
	if device, ok := findDevice(devices, path); ok {
		return &device, ""
	}
	return nil, fmt.Sprintf("device %s was not found in the current discovery results", path)
}

func flattenInterfaces(interfaces []transport.InterfaceSummary) []transport.EndpointSummary {
	var endpoints []transport.EndpointSummary
	for _, iface := range interfaces {
		endpoints = append(endpoints, iface.Endpoints...)
	}
	return endpoints
}

func deviceSummaryDetails(devices []transport.DiscoveredDevice) []string {
	if len(devices) == 0 {
		return []string{"no USB devices discovered"}
	}
	details := []string{fmt.Sprintf("%d USB device(s) discovered", len(devices))}
	for _, device := range devices {
		line := device.StableID
		if strings.TrimSpace(line) == "" {
			line = device.DisplayName
		}
		if line == "" {
			line = "usb device"
		}
		if device.VID != 0 || device.PID != 0 {
			line = fmt.Sprintf("%s (vid=0x%04x pid=0x%04x)", line, device.VID, device.PID)
		}
		if len(device.Interfaces) > 0 {
			line = fmt.Sprintf("%s interfaces=%d endpoints=%d", line, len(device.Interfaces), countEndpoints(device.Interfaces))
		}
		details = append(details, line)
	}
	return details
}

func countEndpoints(interfaces []transport.InterfaceSummary) int {
	total := 0
	for _, iface := range interfaces {
		total += len(iface.Endpoints)
	}
	return total
}

func syntheticTrace(command string, args []string, interpretation string) transport.CommandTrace {
	return transport.CommandTrace{
		Command:        command,
		Args:           append([]string(nil), args...),
		Interpretation: interpretation,
	}
}

type execRunner struct{}

func (execRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (execRunner) Run(ctx context.Context, name string, args ...string) CommandResult {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := CommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Err:    err,
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		return result
	}
	result.ExitCode = 0
	return result
}

func parseDeviceList(raw string) ([]transport.DiscoveredDevice, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var (
		paths      []string
		parsedJSON bool
	)
	if strings.HasPrefix(raw, "[") {
		var parsed []string
		if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
			paths = parsed
			parsedJSON = true
		}
	}
	if !parsedJSON {
		for _, line := range strings.Split(raw, "\n") {
			line = strings.TrimSpace(strings.Trim(line, ","))
			line = strings.Trim(line, "\"")
			line = strings.Trim(line, "'")
			if line == "" || line == "[" || line == "]" {
				continue
			}
			paths = append(paths, line)
		}
	}

	unique := make(map[string]struct{}, len(paths))
	devices := make([]transport.DiscoveredDevice, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := unique[path]; ok {
			continue
		}
		unique[path] = struct{}{}
		devices = append(devices, transport.DiscoveredDevice{
			StableID:        path,
			DisplayName:     path,
			TransportFamily: transport.TransportFamilyUSBSerial,
			Permission:      transport.PermissionStateUnknown,
			Capabilities: transport.CapabilitiesFromList(
				transport.CapabilityDiscovery,
				transport.CapabilityPermission,
				transport.CapabilityUSBHandle,
				transport.CapabilityDescriptorDiscovery,
				transport.CapabilityBulkTransfer,
				transport.CapabilityInterruptTransfer,
				transport.CapabilityControlTransfer,
			),
			Metadata: map[string]string{
				"raw": path,
			},
		})
	}
	sort.SliceStable(devices, func(i, j int) bool { return devices[i].StableID < devices[j].StableID })
	return devices, nil
}

func permissionStatus(state transport.PermissionState) diagnostics.Status {
	switch state {
	case transport.PermissionStateGranted:
		return diagnostics.StatusPassed
	case transport.PermissionStateDenied:
		return diagnostics.StatusFailed
	case transport.PermissionStateRequired:
		return diagnostics.StatusWarning
	case transport.PermissionStateUnavailable:
		return diagnostics.StatusWarning
	default:
		return diagnostics.StatusSkipped
	}
}

func isZeroCapabilities(c transport.TransportCapabilities) bool {
	return len(c.List()) == 0
}
