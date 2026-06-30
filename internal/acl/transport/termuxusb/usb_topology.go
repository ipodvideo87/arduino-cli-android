package termuxusb

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/acl/transport"
)

type USBTopologyReport struct {
	SchemaVersion     string                     `json:"schema_version,omitempty"`
	Status            diagnostics.Status         `json:"status,omitempty"`
	Provider          string                     `json:"provider,omitempty"`
	ProviderKind      transport.Kind             `json:"provider_kind,omitempty"`
	Device            transport.DiscoveredDevice `json:"device,omitempty"`
	BridgeState       string                     `json:"bridge_state,omitempty"`
	BridgeStateReason string                     `json:"bridge_state_reason,omitempty"`
	FDEnvPresent      bool                       `json:"fd_env_present,omitempty"`
	FDEnvValue        string                     `json:"fd_env_value,omitempty"`
	FDObserved        bool                       `json:"fd_observed,omitempty"`
	FDValid           bool                       `json:"fd_valid,omitempty"`
	FDInspectable     bool                       `json:"fd_inspectable,omitempty"`
	FDSource          string                     `json:"fd_source,omitempty"`
	HandoffMode       string                     `json:"handoff_mode,omitempty"`
	ClaimReleaseState string                     `json:"claim_release_state,omitempty"`
	Warnings          []string                   `json:"warnings,omitempty"`
	Limitations       []string                   `json:"limitations,omitempty"`
	Beginner          string                     `json:"beginner_summary,omitempty"`
	Professional      []string                   `json:"professional_details,omitempty"`
	NextStep          string                     `json:"next_step,omitempty"`
	Metadata          map[string]string          `json:"metadata,omitempty"`
}

var inspectUSBTopology = inspectUSBTopologyImpl

func HelperUSBTopologyReportFromEnv() USBTopologyReport {
	return HelperUSBTopologyReportFromInvocation(nil)
}

func HelperUSBTopologyReportFromInvocation(args []string) USBTopologyReport {
	rawEnv := strings.TrimSpace(os.Getenv("TERMUX_USB_FD"))
	rawArg, argPresent, argMalformed := probeFDFromArgs(args)
	report := USBTopologyReport{
		SchemaVersion:     "1",
		Status:            diagnostics.StatusWarning,
		Provider:          defaultProviderID,
		ProviderKind:      transport.KindAndroidUSBFD,
		BridgeState:       "experimental",
		BridgeStateReason: "descriptor/interface/endpoint diagnostics only",
		ClaimReleaseState: "not_attempted",
		Warnings:          []string{},
		Limitations: []string{
			"no payload transfers were attempted",
			"claim/release diagnostics were not attempted",
		},
		Beginner: "USB topology bridge is experimental",
		Professional: []string{
			"descriptor, interface, and endpoint evidence is collected without sending payload data",
		},
		NextStep: "use the collected topology to decide whether a later transfer milestone is safe",
		Metadata: map[string]string{},
	}
	report.Metadata["helper_args"] = strings.Join(args, " ")
	report.Metadata["bridge_state"] = report.BridgeState
	report.Metadata["claim_release_state"] = report.ClaimReleaseState

	switch {
	case rawEnv != "":
		report.FDEnvPresent = true
		report.FDEnvValue = rawEnv
		report.FDObserved = true
		report.FDSource = "environment"
		report.HandoffMode = "env"
		report.Metadata["fd_source"] = report.FDSource
		report.Metadata["handoff_mode"] = report.HandoffMode
		fd, err := strconv.Atoi(rawEnv)
		if err != nil {
			report.Status = diagnostics.StatusFailed
			report.BridgeState = "failed"
			report.BridgeStateReason = "TERMUX_USB_FD could not be parsed"
			report.Beginner = "TERMUX_USB_FD is invalid"
			report.Warnings = []string{"TERMUX_USB_FD could not be parsed"}
			report.Limitations = append(report.Limitations, "fd value is not a valid integer")
			report.Professional = []string{
				"TERMUX_USB_FD could not be parsed",
				"raw value: " + rawEnv,
			}
			report.NextStep = "fix the helper invocation before attempting USB topology inspection"
			return report
		}
		report.FDValid = true
		device, err := inspectUSBTopology(fd, args)
		if err != nil {
			report.BridgeState = "unsupported"
			report.BridgeStateReason = err.Error()
			report.Warnings = []string{err.Error()}
			report.Limitations = append(report.Limitations, "USB topology inspection is unavailable on this build")
			report.Beginner = "TERMUX_USB_FD observed, but USB topology inspection is unavailable"
			report.Professional = []string{
				"TERMUX_USB_FD was observed",
				"topology inspection could not wrap the fd",
			}
			report.NextStep = "use the native Android build with libusb support to inspect descriptors"
			return report
		}
		report.FDInspectable = true
		report.Device = device
		report.Device.StableID = deviceStableID(device, args)
		report.Device.DisplayName = deviceDisplayName(report.Device)
		if report.Device.Metadata == nil {
			report.Device.Metadata = map[string]string{}
		}
		report.Device.Metadata["bridge_state"] = report.BridgeState
		report.Device.Metadata["claim_release_state"] = report.ClaimReleaseState
		report.Device.Metadata["topology_source"] = "libusb"
		report.BridgeStateReason = "USB descriptors, interfaces, and endpoints were collected"
		report.Beginner = "TERMUX_USB_FD observed via environment; USB topology bridge remains experimental"
		report.Professional = []string{
			"TERMUX_USB_FD was observed and the USB topology could be inspected",
			"descriptors, interfaces, and endpoints were collected without payload transfers",
		}
		report.NextStep = "use the topology evidence to decide whether later claim/release diagnostics are safe"
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
			report.BridgeState = "failed"
			report.BridgeStateReason = "fd argument could not be parsed"
			report.Beginner = "fd argument is invalid"
			report.Warnings = []string{"file descriptor argument could not be parsed"}
			report.Limitations = []string{"fd argument is not a valid integer"}
			report.Professional = []string{
				"fd argument could not be parsed",
				"raw argument: " + rawArg,
			}
			report.NextStep = "fix the helper invocation before attempting USB topology inspection"
			return report
		}
		report.FDValid = true
		device, err := inspectUSBTopology(fd, args)
		if err != nil {
			report.BridgeState = "unsupported"
			report.BridgeStateReason = err.Error()
			report.Warnings = []string{err.Error()}
			report.Limitations = append(report.Limitations, "USB topology inspection is unavailable on this build")
			report.Beginner = "fd argument observed, but USB topology inspection is unavailable"
			report.Professional = []string{
				"fd argument was observed",
				"topology inspection could not wrap the fd",
			}
			report.NextStep = "use the native Android build with libusb support to inspect descriptors"
			return report
		}
		report.FDInspectable = true
		report.Device = device
		report.Device.StableID = deviceStableID(device, args)
		report.Device.DisplayName = deviceDisplayName(report.Device)
		if report.Device.Metadata == nil {
			report.Device.Metadata = map[string]string{}
		}
		report.Device.Metadata["bridge_state"] = report.BridgeState
		report.Device.Metadata["claim_release_state"] = report.ClaimReleaseState
		report.Device.Metadata["topology_source"] = "libusb"
		report.BridgeStateReason = "USB descriptors, interfaces, and endpoints were collected"
		report.Beginner = "file descriptor observed via command-line argument; USB topology bridge remains experimental"
		report.Professional = []string{
			"fd argument was observed and the USB topology could be inspected",
			"descriptors, interfaces, and endpoints were collected without payload transfers",
		}
		report.NextStep = "use the topology evidence to decide whether later claim/release diagnostics are safe"
		return report
	case argPresent && argMalformed:
		report.FDObserved = true
		report.FDSource = "argument"
		report.HandoffMode = "argv"
		report.Metadata["fd_source"] = report.FDSource
		report.Metadata["handoff_mode"] = report.HandoffMode
		report.Status = diagnostics.StatusFailed
		report.BridgeState = "failed"
		report.BridgeStateReason = "fd argument could not be parsed"
		report.Beginner = "fd argument is invalid"
		report.Warnings = []string{"file descriptor argument could not be parsed"}
		report.Limitations = []string{"fd argument is not a valid integer"}
		report.Professional = []string{
			"fd argument could not be parsed",
			"raw argument: " + rawArg,
		}
		report.NextStep = "fix the helper invocation before attempting USB topology inspection"
		return report
	default:
		report.BridgeState = "unavailable"
		report.BridgeStateReason = "TERMUX_USB_FD is not set"
		report.Beginner = "TERMUX_USB_FD is not set"
		report.Limitations = []string{"fd handoff is unavailable outside termux-usb -e"}
		report.Professional = []string{
			"the helper did not observe an fd handoff",
			"run the helper via termux-usb -r -E to inspect TERMUX_USB_FD and the USB topology",
		}
		report.FDSource = "none"
		report.HandoffMode = "env"
		report.Metadata["fd_source"] = report.FDSource
		report.Metadata["handoff_mode"] = report.HandoffMode
		report.NextStep = "run the helper through termux-usb -r -E -e so it receives TERMUX_USB_FD"
		return report
	}
}

func parseUSBTopologyReport(raw string) (USBTopologyReport, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return USBTopologyReport{}, errors.New("empty USB topology output")
	}
	var report USBTopologyReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		return USBTopologyReport{}, err
	}
	return report, nil
}

func defaultUSBTopologyHelperCommand() string {
	return fmt.Sprintf("%s acl transport usb-topology-helper --json", os.Args[0])
}

func mergeUSBTopologyDevice(base, overlay transport.DiscoveredDevice) transport.DiscoveredDevice {
	if strings.TrimSpace(overlay.Provider) != "" {
		base.Provider = overlay.Provider
	}
	if strings.TrimSpace(overlay.StableID) != "" {
		base.StableID = overlay.StableID
	}
	if strings.TrimSpace(overlay.DisplayName) != "" {
		base.DisplayName = overlay.DisplayName
	}
	if strings.TrimSpace(string(overlay.TransportFamily)) != "" {
		base.TransportFamily = overlay.TransportFamily
	}
	if overlay.VID != 0 {
		base.VID = overlay.VID
	}
	if overlay.PID != 0 {
		base.PID = overlay.PID
	}
	if strings.TrimSpace(overlay.Manufacturer) != "" {
		base.Manufacturer = overlay.Manufacturer
	}
	if strings.TrimSpace(overlay.Product) != "" {
		base.Product = overlay.Product
	}
	if strings.TrimSpace(overlay.SerialNumber) != "" {
		base.SerialNumber = overlay.SerialNumber
	}
	if len(overlay.Interfaces) > 0 {
		base.Interfaces = append([]transport.InterfaceSummary(nil), overlay.Interfaces...)
	}
	base.Capabilities = mergeTransportCapabilities(base.Capabilities, overlay.Capabilities)
	base.Warnings = appendUniqueStrings(base.Warnings, overlay.Warnings...)
	base.Limitations = appendUniqueStrings(base.Limitations, overlay.Limitations...)
	if base.Metadata == nil {
		base.Metadata = map[string]string{}
	}
	for k, v := range overlay.Metadata {
		base.Metadata[k] = v
	}
	if overlay.Permission != "" {
		base.Permission = overlay.Permission
	}
	return base
}

func mergeTransportCapabilities(base, overlay transport.TransportCapabilities) transport.TransportCapabilities {
	return transport.TransportCapabilities{
		Discovery:           base.Discovery || overlay.Discovery,
		Permission:          base.Permission || overlay.Permission,
		SerialIO:            base.SerialIO || overlay.SerialIO,
		ControlLines:        base.ControlLines || overlay.ControlLines,
		LineCoding:          base.LineCoding || overlay.LineCoding,
		ResetSignal:         base.ResetSignal || overlay.ResetSignal,
		BulkTransfer:        base.BulkTransfer || overlay.BulkTransfer,
		InterruptTransfer:   base.InterruptTransfer || overlay.InterruptTransfer,
		ControlTransfer:     base.ControlTransfer || overlay.ControlTransfer,
		PTYExport:           base.PTYExport || overlay.PTYExport,
		RFC2217Export:       base.RFC2217Export || overlay.RFC2217Export,
		UploadEndpoint:      base.UploadEndpoint || overlay.UploadEndpoint,
		MonitorEndpoint:     base.MonitorEndpoint || overlay.MonitorEndpoint,
		USBHandle:           base.USBHandle || overlay.USBHandle,
		DescriptorDiscovery: base.DescriptorDiscovery || overlay.DescriptorDiscovery,
		ModemControl:        base.ModemControl || overlay.ModemControl,
		Future:              base.Future || overlay.Future,
	}
}

func appendUniqueStrings(base []string, values ...string) []string {
	seen := make(map[string]struct{}, len(base))
	for _, item := range base {
		seen[item] = struct{}{}
	}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		base = append(base, trimmed)
	}
	return base
}

func deviceStableID(device transport.DiscoveredDevice, args []string) string {
	if strings.TrimSpace(device.StableID) != "" {
		return device.StableID
	}
	if len(args) > 0 {
		if raw, ok := lastNonFlagArg(args); ok {
			return raw
		}
	}
	return ""
}

func deviceDisplayName(device transport.DiscoveredDevice) string {
	if strings.TrimSpace(device.DisplayName) != "" {
		return device.DisplayName
	}
	return device.StableID
}

func lastNonFlagArg(args []string) (string, bool) {
	for i := len(args) - 1; i >= 0; i-- {
		arg := strings.TrimSpace(args[i])
		if arg == "" || strings.HasPrefix(arg, "-") {
			continue
		}
		return arg, true
	}
	return "", false
}
