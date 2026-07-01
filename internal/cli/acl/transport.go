package acl

import (
	"fmt"
	"strings"
	"time"

	acldiagnostics "github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/acl/transport"
	"github.com/arduino/arduino-cli/internal/acl/transport/termuxusb"
	"github.com/spf13/cobra"
)

type transportProvider interface {
	transport.TransportProvider
	transport.TransportDiscoverer
	transport.PermissionRequester
	transport.DiagnosticsReporter
	transport.SessionOpener
	transport.StreamProber
	transport.StreamValidator
	transport.InterfaceClaimReleaser
}

var newTransportProvider = func() transportProvider {
	return termuxusb.NewProvider()
}

type transportListReport struct {
	Status       acldiagnostics.Status                `json:"status"`
	Beginner     string                               `json:"beginner_summary,omitempty"`
	Professional []string                             `json:"professional_details,omitempty"`
	Diagnostics  transport.TransportDiagnosticsReport `json:"diagnostics"`
	Devices      []transport.DiscoveredDevice         `json:"devices,omitempty"`
}

type transportAcquireReport struct {
	Status       acldiagnostics.Status                `json:"status"`
	Beginner     string                               `json:"beginner_summary,omitempty"`
	Professional []string                             `json:"professional_details,omitempty"`
	DevicePath   string                               `json:"device_path,omitempty"`
	Permission   transport.PermissionResult           `json:"permission"`
	Diagnostics  transport.TransportDiagnosticsReport `json:"diagnostics"`
}

func newTransportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transport",
		Short: "Inspect ACL transport providers",
	}
	cmd.AddCommand(newTransportListCommand())
	cmd.AddCommand(newTransportDiagnoseCommand())
	cmd.AddCommand(newTransportAcquireCommand())
	cmd.AddCommand(newTransportProbeFDCommand())
	cmd.AddCommand(newTransportStreamStatusCommand())
	cmd.AddCommand(newTransportStreamValidateCommand())
	cmd.AddCommand(newTransportClaimReleaseCommand())
	cmd.AddCommand(newTransportUSBTopologyHelperCommand())
	cmd.AddCommand(newTransportProbeFDHelperCommand())
	cmd.AddCommand(newTransportStreamValidateHelperCommand())
	cmd.AddCommand(newTransportClaimReleaseHelperCommand())
	return cmd
}

func newTransportListCommand() *cobra.Command {
	opts := transportCommandOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available ACL transport devices",
		RunE: func(cmd *cobra.Command, _ []string) error {
			provider := newTransportProvider()
			report, err := provider.Diagnostics(cmd.Context(), transport.DiagnosticsRequest{})
			if err != nil && strings.TrimSpace(report.Beginner) == "" {
				report.Beginner = err.Error()
			}
			list := transportListReport{
				Status:       report.Status,
				Beginner:     report.BeginnerSummary(),
				Professional: report.ProfessionalDetails(),
				Diagnostics:  report,
				Devices:      append([]transport.DiscoveredDevice(nil), report.Devices...),
			}
			if isJSON(cmd) {
				return writeJSON(cmd, list)
			}
			return writeTransportListReport(cmd, list, opts.details)
		},
	}
	cmd.Flags().BoolVar(&opts.details, "details", false, "Show professional-level details")
	return cmd
}

func newTransportDiagnoseCommand() *cobra.Command {
	opts := transportCommandOptions{}
	var devicePath string
	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "Diagnose Termux USB transport availability",
		RunE: func(cmd *cobra.Command, _ []string) error {
			provider := newTransportProvider()
			report, err := provider.Diagnostics(cmd.Context(), transport.DiagnosticsRequest{
				Metadata: map[string]string{
					"device_path": devicePath,
				},
			})
			if err != nil && strings.TrimSpace(report.Beginner) == "" {
				report.Beginner = err.Error()
			}
			if isJSON(cmd) {
				return writeJSON(cmd, report)
			}
			return writeTransportDiagnosticsReport(cmd, report, opts.details)
		},
	}
	cmd.Flags().BoolVar(&opts.details, "details", false, "Show professional-level details")
	cmd.Flags().StringVar(&devicePath, "device", "", "USB device path to diagnose")
	return cmd
}

func newTransportAcquireCommand() *cobra.Command {
	opts := transportCommandOptions{}
	var devicePath string
	var command string
	cmd := &cobra.Command{
		Use:   "acquire",
		Short: "Request permission for a Termux USB device",
		RunE: func(cmd *cobra.Command, _ []string) error {
			provider := newTransportProvider()
			permission, err := provider.RequestPermission(cmd.Context(), transport.PermissionRequest{
				Required: true,
				Method:   "termux-usb -r",
				Device: transport.DiscoveredDevice{
					StableID:        devicePath,
					DisplayName:     devicePath,
					TransportFamily: transport.TransportFamilyUSBSerial,
					Permission:      transport.PermissionStateUnknown,
				},
				Metadata: map[string]string{
					"device_path": devicePath,
					"command":     command,
				},
			})
			diagnosticsReport, diagErr := provider.Diagnostics(cmd.Context(), transport.DiagnosticsRequest{
				Metadata: map[string]string{
					"device_path": devicePath,
				},
			})
			if diagErr != nil && strings.TrimSpace(diagnosticsReport.Beginner) == "" {
				diagnosticsReport.Beginner = diagErr.Error()
			}
			report := transportAcquireReport{
				Status:       permissionStateToStatus(permission.State),
				Beginner:     permission.BeginnerSummary(),
				Professional: append(permission.ProfessionalDetails(), diagnosticsReport.ProfessionalDetails()...),
				DevicePath:   devicePath,
				Permission:   permission,
				Diagnostics:  diagnosticsReport,
			}
			if err != nil && strings.TrimSpace(report.Beginner) == "" {
				report.Beginner = err.Error()
			}
			if isJSON(cmd) {
				return writeJSON(cmd, report)
			}
			return writeTransportAcquireReport(cmd, report, opts.details)
		},
	}
	cmd.Flags().BoolVar(&opts.details, "details", false, "Show professional-level details")
	cmd.Flags().StringVar(&devicePath, "device", "", "USB device path to acquire")
	cmd.Flags().StringVar(&command, "command", "", "Optional command to launch through termux-usb -e")
	_ = cmd.MarkFlagRequired("device")
	return cmd
}

func newTransportProbeFDCommand() *cobra.Command {
	return newTransportStreamProbeCommand("probe-fd", "Probe TERMUX_USB_FD handoff for a Termux USB device")
}

func newTransportStreamStatusCommand() *cobra.Command {
	return newTransportStreamProbeCommand("stream-status", "Report transport stream status for a Termux USB device")
}

func newTransportStreamValidateCommand() *cobra.Command {
	opts := transportStreamValidateOptions{}
	var devicePath string
	cmd := &cobra.Command{
		Use:   "stream-validate",
		Short: "Validate bounded Termux USB stream behavior for a Termux USB device",
		RunE: func(cmd *cobra.Command, _ []string) error {
			provider := newTransportProvider()
			report, err := provider.Validate(cmd.Context(), transport.StreamValidationRequest{
				Device: transport.DiscoveredDevice{
					StableID:        devicePath,
					DisplayName:     devicePath,
					TransportFamily: transport.TransportFamilyUSBSerial,
				},
				ValidateRead:  opts.validateRead,
				ValidateWrite: opts.validateWrite,
				Timeout:       opts.timeout,
				Metadata: map[string]string{
					"device_path": devicePath,
				},
			})
			if err != nil {
				if isJSON(cmd) {
					return writeJSON(cmd, report)
				}
				return writeTransportStreamValidateReport(cmd, report, opts.details)
			}
			if isJSON(cmd) {
				return writeJSON(cmd, report)
			}
			return writeTransportStreamValidateReport(cmd, report, opts.details)
		},
	}
	cmd.Flags().BoolVar(&opts.details, "details", false, "Show professional-level details")
	cmd.Flags().BoolVar(&opts.validateRead, "validate-read", false, "Attempt a single-byte bounded read probe")
	cmd.Flags().BoolVar(&opts.validateWrite, "validate-write", false, "Attempt a single-byte bounded write probe")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", 2*time.Second, "Maximum time to wait for bounded read/write probes")
	cmd.Flags().StringVar(&devicePath, "device", "", "USB device path to validate")
	_ = cmd.MarkFlagRequired("device")
	return cmd
}

func newTransportClaimReleaseCommand() *cobra.Command {
	opts := transportCommandOptions{}
	var devicePath string
	var interfaceNumber int
	cmd := &cobra.Command{
		Use:   "claim-release",
		Short: "Validate diagnostic interface claim/release behavior for a Termux USB device",
		RunE: func(cmd *cobra.Command, _ []string) error {
			provider := newTransportProvider()
			report, err := provider.ClaimRelease(cmd.Context(), transport.InterfaceClaimReleaseRequest{
				Device: transport.DiscoveredDevice{
					StableID:        devicePath,
					DisplayName:     devicePath,
					TransportFamily: transport.TransportFamilyUSBSerial,
				},
				InterfaceNumber: interfaceNumber,
				Metadata: map[string]string{
					"device_path":      devicePath,
					"interface_number": fmt.Sprintf("%d", interfaceNumber),
				},
			})
			if err != nil && strings.TrimSpace(report.Beginner) == "" {
				report.Beginner = err.Error()
			}
			if isJSON(cmd) {
				return writeJSON(cmd, report)
			}
			return writeTransportClaimReleaseReport(cmd, report, opts.details)
		},
	}
	cmd.Flags().BoolVar(&opts.details, "details", false, "Show professional-level details")
	cmd.Flags().StringVar(&devicePath, "device", "", "USB device path to validate")
	cmd.Flags().IntVar(&interfaceNumber, "interface", 0, "USB interface number to claim and release")
	_ = cmd.MarkFlagRequired("device")
	return cmd
}

func newTransportStreamProbeCommand(use, short string) *cobra.Command {
	opts := transportCommandOptions{}
	var devicePath string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			provider := newTransportProvider()
			report, err := provider.Probe(cmd.Context(), transport.StreamProbeRequest{
				Device: transport.DiscoveredDevice{
					StableID:        devicePath,
					DisplayName:     devicePath,
					TransportFamily: transport.TransportFamilyUSBSerial,
				},
				HandoffMode: "env",
				Metadata: map[string]string{
					"device_path":  devicePath,
					"handoff_mode": "env",
				},
			})
			if err != nil && strings.TrimSpace(report.Beginner) == "" {
				report.Beginner = err.Error()
			}
			if isJSON(cmd) {
				return writeJSON(cmd, report)
			}
			return writeTransportStreamProbeReport(cmd, report, opts.details)
		},
	}
	cmd.Flags().BoolVar(&opts.details, "details", false, "Show professional-level details")
	cmd.Flags().StringVar(&devicePath, "device", "", "USB device path to probe")
	_ = cmd.MarkFlagRequired("device")
	return cmd
}

func newTransportProbeFDHelperCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "probe-fd-helper",
		Hidden: true,
		Short:  "Internal helper for TERMUX_USB_FD diagnostics",
		Args:   cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeJSON(cmd, termuxusb.HelperStreamProbeReportFromInvocation(args))
		},
	}
	return cmd
}

func newTransportStreamValidateHelperCommand() *cobra.Command {
	opts := transportStreamValidateOptions{}
	cmd := &cobra.Command{
		Use:    "stream-validate-helper",
		Hidden: true,
		Short:  "Internal helper for bounded TERMUX_USB_FD stream validation",
		Args:   cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeJSON(cmd, termuxusb.HelperStreamValidationReportFromInvocation(args, opts.validateRead, opts.validateWrite, opts.timeout))
		},
	}
	cmd.Flags().BoolVar(&opts.validateRead, "validate-read", false, "Attempt a single-byte bounded read probe")
	cmd.Flags().BoolVar(&opts.validateWrite, "validate-write", false, "Attempt a single-byte bounded write probe")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", 2*time.Second, "Maximum time to wait for bounded read/write probes")
	return cmd
}

func newTransportClaimReleaseHelperCommand() *cobra.Command {
	opts := transportClaimReleaseOptions{}
	cmd := &cobra.Command{
		Use:    "claim-release-helper",
		Hidden: true,
		Short:  "Internal helper for diagnostic TERMUX_USB_FD interface claim/release validation",
		Args:   cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeJSON(cmd, termuxusb.HelperClaimReleaseReportFromInvocation(args, opts.interfaceNumber))
		},
	}
	cmd.Flags().IntVar(&opts.interfaceNumber, "interface", 0, "USB interface number to claim and release")
	return cmd
}

func newTransportUSBTopologyHelperCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "usb-topology-helper",
		Hidden: true,
		Short:  "Internal helper for TERMUX_USB_FD USB topology diagnostics",
		Args:   cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeJSON(cmd, termuxusb.HelperUSBTopologyReportFromInvocation(args))
		},
	}
	return cmd
}

type transportCommandOptions struct {
	details bool
}

type transportStreamValidateOptions struct {
	details       bool
	validateRead  bool
	validateWrite bool
	timeout       time.Duration
}

type transportClaimReleaseOptions struct {
	interfaceNumber int
}

func permissionStateToStatus(state transport.PermissionState) acldiagnostics.Status {
	switch state {
	case transport.PermissionStateGranted:
		return acldiagnostics.StatusPassed
	case transport.PermissionStateDenied:
		return acldiagnostics.StatusFailed
	case transport.PermissionStateRequired:
		return acldiagnostics.StatusWarning
	case transport.PermissionStateUnavailable:
		return acldiagnostics.StatusWarning
	default:
		return acldiagnostics.StatusSkipped
	}
}

func writeTransportListReport(cmd *cobra.Command, report transportListReport, details bool) error {
	fmt.Fprintln(cmd.OutOrStdout(), "ACL Transport List")
	fmt.Fprintln(cmd.OutOrStdout(), report.Beginner)
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", report.Status)
	fmt.Fprintf(cmd.OutOrStdout(), "Devices: %d\n", len(report.Devices))
	if details {
		for _, detail := range report.Professional {
			fmt.Fprintln(cmd.OutOrStdout(), detail)
		}
		for _, trace := range report.Diagnostics.Traces {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", trace.Command, strings.Join(trace.Args, " "))
			if trace.Stdout != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "stdout: %s\n", trace.Stdout)
			}
			if trace.Stderr != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "stderr: %s\n", trace.Stderr)
			}
			if trace.Interpretation != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "interpretation: %s\n", trace.Interpretation)
			}
		}
	}
	return nil
}

func writeTransportDiagnosticsReport(cmd *cobra.Command, report transport.TransportDiagnosticsReport, details bool) error {
	fmt.Fprintln(cmd.OutOrStdout(), "ACL Transport Diagnose")
	fmt.Fprintln(cmd.OutOrStdout(), report.BeginnerSummary())
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", report.Status)
	if details {
		for _, detail := range report.ProfessionalDetails() {
			fmt.Fprintln(cmd.OutOrStdout(), detail)
		}
		writeTransportDeviceTopology(cmd, report)
		for _, trace := range report.Traces {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", trace.Command, strings.Join(trace.Args, " "))
			if trace.Stdout != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "stdout: %s\n", trace.Stdout)
			}
			if trace.Stderr != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "stderr: %s\n", trace.Stderr)
			}
			if trace.Interpretation != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "interpretation: %s\n", trace.Interpretation)
			}
		}
	}
	return nil
}

func writeTransportDeviceTopology(cmd *cobra.Command, report transport.TransportDiagnosticsReport) {
	if report.Device.VID != 0 || report.Device.PID != 0 || strings.TrimSpace(report.Device.Manufacturer) != "" || strings.TrimSpace(report.Device.Product) != "" || strings.TrimSpace(report.Device.SerialNumber) != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "device descriptor: vid=0x%04x pid=0x%04x\n", report.Device.VID, report.Device.PID)
		if strings.TrimSpace(report.Device.Manufacturer) != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "manufacturer: %s\n", report.Device.Manufacturer)
		}
		if strings.TrimSpace(report.Device.Product) != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "product: %s\n", report.Device.Product)
		}
		if strings.TrimSpace(report.Device.SerialNumber) != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "serial: %s\n", report.Device.SerialNumber)
		}
	}
	for _, iface := range report.Interfaces {
		fmt.Fprintf(cmd.OutOrStdout(), "interface %d alt %d: class=%s subclass=%s protocol=%s\n", iface.Number, iface.Alternate, iface.Class, iface.Subclass, iface.Protocol)
		if strings.TrimSpace(iface.Description) != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  description: %s\n", iface.Description)
		}
		for _, note := range iface.Notes {
			fmt.Fprintf(cmd.OutOrStdout(), "  note: %s\n", note)
		}
		for _, endpoint := range iface.Endpoints {
			fmt.Fprintf(cmd.OutOrStdout(), "  endpoint 0x%02x: direction=%s type=%s mps=%d interval=%d\n", endpoint.Address, endpoint.Direction, endpoint.Type, endpoint.MaxPacketSize, endpoint.Interval)
			if strings.TrimSpace(endpoint.Usage) != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "    usage: %s\n", endpoint.Usage)
			}
			for _, note := range endpoint.Notes {
				fmt.Fprintf(cmd.OutOrStdout(), "    note: %s\n", note)
			}
		}
	}
}

func writeTransportAcquireReport(cmd *cobra.Command, report transportAcquireReport, details bool) error {
	fmt.Fprintln(cmd.OutOrStdout(), "ACL Transport Acquire")
	fmt.Fprintln(cmd.OutOrStdout(), report.Beginner)
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", report.Status)
	if details {
		for _, detail := range report.Professional {
			fmt.Fprintln(cmd.OutOrStdout(), detail)
		}
		for _, trace := range report.Diagnostics.Traces {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", trace.Command, strings.Join(trace.Args, " "))
			if trace.Stdout != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "stdout: %s\n", trace.Stdout)
			}
			if trace.Stderr != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "stderr: %s\n", trace.Stderr)
			}
			if trace.Interpretation != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "interpretation: %s\n", trace.Interpretation)
			}
		}
	}
	return nil
}

func writeTransportStreamProbeReport(cmd *cobra.Command, report transport.TransportStreamDiagnosticsReport, details bool) error {
	title := "ACL Transport Probe FD"
	if cmd != nil && cmd.Name() == "stream-status" {
		title = "ACL Transport Stream Status"
	}
	fmt.Fprintln(cmd.OutOrStdout(), title)
	fmt.Fprintln(cmd.OutOrStdout(), report.BeginnerSummary())
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", report.Status)
	if details {
		for _, detail := range report.ProfessionalDetails() {
			fmt.Fprintln(cmd.OutOrStdout(), detail)
		}
		for _, trace := range report.Traces {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", trace.Command, strings.Join(trace.Args, " "))
			if trace.Stdout != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "stdout: %s\n", trace.Stdout)
			}
			if trace.Stderr != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "stderr: %s\n", trace.Stderr)
			}
			if trace.Interpretation != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "interpretation: %s\n", trace.Interpretation)
			}
		}
	}
	return nil
}

func writeTransportStreamValidateReport(cmd *cobra.Command, report transport.TransportStreamValidationReport, details bool) error {
	fmt.Fprintln(cmd.OutOrStdout(), "ACL Transport Stream Validate")
	fmt.Fprintln(cmd.OutOrStdout(), report.BeginnerSummary())
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", report.Status)
	fmt.Fprintf(cmd.OutOrStdout(), "Device: %s\n", report.Device.StableID)
	if report.Timeout > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Timeout: %s\n", report.Timeout)
	}
	if report.ValidateRead {
		fmt.Fprintf(cmd.OutOrStdout(), "Validate read: requested\n")
	}
	if report.ValidateWrite {
		fmt.Fprintf(cmd.OutOrStdout(), "Validate write: requested\n")
	}
	if details {
		for _, detail := range report.ProfessionalDetails() {
			fmt.Fprintln(cmd.OutOrStdout(), detail)
		}
		for _, trace := range report.Traces {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", trace.Command, strings.Join(trace.Args, " "))
			if trace.Stdout != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "stdout: %s\n", trace.Stdout)
			}
			if trace.Stderr != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "stderr: %s\n", trace.Stderr)
			}
			if trace.Interpretation != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "interpretation: %s\n", trace.Interpretation)
			}
		}
	}
	return nil
}

func writeTransportClaimReleaseReport(cmd *cobra.Command, report transport.InterfaceClaimReleaseReport, details bool) error {
	fmt.Fprintln(cmd.OutOrStdout(), "ACL Transport Claim Release")
	fmt.Fprintln(cmd.OutOrStdout(), report.BeginnerSummary())
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", report.Status)
	fmt.Fprintf(cmd.OutOrStdout(), "Device: %s\n", report.Device.StableID)
	fmt.Fprintf(cmd.OutOrStdout(), "Interface: %d\n", report.InterfaceNumber)
	if details {
		for _, detail := range report.ProfessionalDetails() {
			fmt.Fprintln(cmd.OutOrStdout(), detail)
		}
		for _, trace := range report.Traces {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", trace.Command, strings.Join(trace.Args, " "))
			if trace.Stdout != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "stdout: %s\n", trace.Stdout)
			}
			if trace.Stderr != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "stderr: %s\n", trace.Stderr)
			}
			if trace.Interpretation != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "interpretation: %s\n", trace.Interpretation)
			}
		}
	}
	return nil
}
