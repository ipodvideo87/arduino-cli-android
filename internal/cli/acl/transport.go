package acl

import (
	"fmt"
	"strings"

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
	cmd.AddCommand(newTransportProbeFDHelperCommand())
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
	opts := transportCommandOptions{}
	var devicePath string
	cmd := &cobra.Command{
		Use:   "probe-fd",
		Short: "Probe TERMUX_USB_FD handoff for a Termux USB device",
		RunE: func(cmd *cobra.Command, _ []string) error {
			provider := newTransportProvider()
			report, err := provider.Probe(cmd.Context(), transport.StreamProbeRequest{
				Device: transport.DiscoveredDevice{
					StableID:        devicePath,
					DisplayName:     devicePath,
					TransportFamily: transport.TransportFamilyUSBSerial,
				},
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			return writeJSON(cmd, termuxusb.HelperStreamProbeReportFromEnv())
		},
	}
	return cmd
}

type transportCommandOptions struct {
	details bool
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
	fmt.Fprintln(cmd.OutOrStdout(), "ACL Transport Probe FD")
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
