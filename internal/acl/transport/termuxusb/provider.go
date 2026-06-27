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

func (p *Provider) Diagnostics(ctx context.Context, req transport.DiagnosticsRequest) (transport.TransportDiagnosticsReport, error) {
	report := transport.TransportDiagnosticsReport{
		SchemaVersion:    "1",
		Status:           diagnostics.StatusWarning,
		Provider:         defaultProviderID,
		ProviderKind:     transport.KindAndroidUSBFD,
		Selected:         p.Descriptor(),
		DiscoveryStatus:  diagnostics.StatusWarning,
		PermissionStatus: diagnostics.StatusSkipped,
		ConnectionStatus: diagnostics.StatusSkipped,
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
		report.Beginner = "Termux USB diagnostics completed for the selected device"
		if report.Status != diagnostics.StatusFailed {
			if export.Kind == transport.EndpointExportFileDescriptor {
				report.Status = diagnostics.StatusPassed
			} else {
				report.Status = diagnostics.StatusWarning
			}
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
	return devices, []transport.CommandTrace{trace}, status, nil
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
	}
	return &Session{
		devicePath: device.StableID,
		permission: permission,
		report:     report,
		endpoint:   endpoint,
		caps:       caps,
	}
}

func (s *Session) Close() error { return nil }

func (s *Session) Capabilities() transport.TransportCapabilities { return s.caps }

func (s *Session) Diagnostics() transport.TransportDiagnosticsReport { return s.report }

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
		details = append(details, line)
	}
	return details
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
