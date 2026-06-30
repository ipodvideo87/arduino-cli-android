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
			session, err := provider.Open(cmd.Context(), transport.OpenRequest{
				Device: transport.DiscoveredDevice{
					StableID:        devicePath,
					DisplayName:     devicePath,
					TransportFamily: transport.TransportFamilyUSBSerial,
				},
				Metadata: map[string]string{
					"device_path": devicePath,
				},
			})
			if err != nil {
				return err
			}
			defer session.Close()

			streamSession, ok := session.(transport.StreamSession)
			if !ok {
				return fmt.Errorf("selected session does not expose a byte stream")
			}
			liveStream, err := streamSession.Stream()
			if err != nil {
				return err
			}

			baseDiagnostics := session.Diagnostics()
			boundedStream := transport.NewBoundedTransportStream(liveStream, baseDiagnostics.StreamProbe, transport.TransportStreamOptions{
				State:        transport.StreamStateFromDiagnostics(baseDiagnostics.StreamProbe),
				Experimental: baseDiagnostics.StreamProbe.State == transport.TransportStreamStateExperimental,
				Bounds: transport.TransportStreamBounds{
					MaxReadBytes:  1,
					MaxWriteBytes: 1,
				},
				Timeouts: transport.TransportStreamTimeouts{
					Read:  opts.timeout,
					Write: opts.timeout,
					Idle:  opts.timeout,
				},
			})

			report := transportStreamValidateReport{
				Status:        acldiagnostics.StatusWarning,
				Beginner:      "Termux USB stream wrapped for bounded validation",
				DevicePath:    devicePath,
				Diagnostics:   baseDiagnostics,
				ValidateRead:  opts.validateRead,
				ValidateWrite: opts.validateWrite,
				Timeout:       opts.timeout,
			}
			if baseDiagnostics.StreamProbe.Beginner != "" {
				report.Professional = append(report.Professional, baseDiagnostics.StreamProbe.ProfessionalDetails()...)
			}
			report.Professional = append(report.Professional,
				"bounded validation is diagnostic-only",
				"read and write are limited to a single byte when requested",
			)

			if timeoutController, ok := boundedStream.(transport.TransportStreamTimeoutController); ok && opts.timeout > 0 {
				_ = timeoutController.SetTimeouts(transport.TransportStreamTimeouts{
					Read:  opts.timeout,
					Write: opts.timeout,
					Idle:  opts.timeout,
				})
			}

			if opts.validateRead {
				buf := make([]byte, 1)
				report.ReadBytes, report.ReadError = readSingleByte(boundedStream, buf)
				if report.ReadError != "" {
					report.Professional = append(report.Professional, "read probe error: "+report.ReadError)
				}
			}
			if opts.validateWrite {
				report.WriteBytes, report.WriteError = writeSingleByte(boundedStream)
				if report.WriteError != "" {
					report.Professional = append(report.Professional, "write probe error: "+report.WriteError)
				}
			}

			_ = boundedStream.Close()
			finalReport := boundedStream.Diagnostics()
			baseDiagnostics.StreamProbe = finalReport
			baseDiagnostics.StreamStatus = finalReport.Status
			report.Diagnostics = baseDiagnostics

			if report.ReadError != "" || report.WriteError != "" {
				report.Professional = append(report.Professional, "bounded validation completed with warnings")
			} else if !opts.validateRead && !opts.validateWrite {
				report.Professional = append(report.Professional, "no byte probes were requested")
			} else {
				report.Professional = append(report.Professional, "bounded byte probes completed")
			}
			report.StreamStatus = finalReport.Status
			report.StreamState = finalReport.State
			report.CloseReason = finalReport.CloseReason
			report.DisconnectReason = finalReport.DisconnectReason
			report.BytesRead = finalReport.BytesRead
			report.BytesWritten = finalReport.BytesWritten
			report.LastActivity = finalReport.LastActivity
			report.StreamProbe = finalReport
			report.Status = acldiagnostics.StatusWarning
			if report.ReadError == "" && report.WriteError == "" && !opts.validateRead && !opts.validateWrite {
				report.Beginner = "Termux USB stream wrapped; bounded validation probes were not requested"
			} else if report.ReadError == "" && report.WriteError == "" {
				report.Beginner = "Termux USB bounded stream validation completed"
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

type transportCommandOptions struct {
	details bool
}

type transportStreamValidateOptions struct {
	details       bool
	validateRead  bool
	validateWrite bool
	timeout       time.Duration
}

type transportStreamValidateReport struct {
	Status           acldiagnostics.Status                      `json:"status"`
	Beginner         string                                     `json:"beginner_summary,omitempty"`
	Professional     []string                                   `json:"professional_details,omitempty"`
	DevicePath       string                                     `json:"device_path,omitempty"`
	ValidateRead     bool                                       `json:"validate_read,omitempty"`
	ValidateWrite    bool                                       `json:"validate_write,omitempty"`
	Timeout          time.Duration                              `json:"timeout,omitempty"`
	ReadBytes        int64                                      `json:"read_bytes,omitempty"`
	WriteBytes       int64                                      `json:"write_bytes,omitempty"`
	ReadError        string                                     `json:"read_error,omitempty"`
	WriteError       string                                     `json:"write_error,omitempty"`
	Diagnostics      transport.TransportDiagnosticsReport       `json:"diagnostics"`
	StreamProbe      transport.TransportStreamDiagnosticsReport `json:"stream_probe,omitempty"`
	StreamStatus     acldiagnostics.Status                      `json:"stream_status,omitempty"`
	StreamState      transport.TransportStreamState             `json:"stream_state,omitempty"`
	CloseReason      string                                     `json:"close_reason,omitempty"`
	DisconnectReason string                                     `json:"disconnect_reason,omitempty"`
	BytesRead        int64                                      `json:"bytes_read,omitempty"`
	BytesWritten     int64                                      `json:"bytes_written,omitempty"`
	LastActivity     time.Time                                  `json:"last_activity,omitempty"`
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

func writeTransportStreamValidateReport(cmd *cobra.Command, report transportStreamValidateReport, details bool) error {
	fmt.Fprintln(cmd.OutOrStdout(), "ACL Transport Stream Validate")
	fmt.Fprintln(cmd.OutOrStdout(), report.Beginner)
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", report.Status)
	if report.DevicePath != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Device: %s\n", report.DevicePath)
	}
	if report.Timeout > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Timeout: %s\n", report.Timeout)
	}
	if report.ValidateRead {
		fmt.Fprintf(cmd.OutOrStdout(), "Validate read: requested\n")
	}
	if report.ValidateWrite {
		fmt.Fprintf(cmd.OutOrStdout(), "Validate write: requested\n")
	}
	if report.StreamState != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Stream state: %s\n", report.StreamState)
	}
	if report.StreamStatus != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Stream status: %s\n", report.StreamStatus)
	}
	if report.BytesRead != 0 || report.BytesWritten != 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Bytes: read=%d written=%d\n", report.BytesRead, report.BytesWritten)
	}
	if report.ReadError != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Read error: %s\n", report.ReadError)
	}
	if report.WriteError != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Write error: %s\n", report.WriteError)
	}
	if report.CloseReason != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Close reason: %s\n", report.CloseReason)
	}
	if report.DisconnectReason != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Disconnect reason: %s\n", report.DisconnectReason)
	}
	if details {
		for _, detail := range report.Professional {
			fmt.Fprintln(cmd.OutOrStdout(), detail)
		}
		for _, detail := range report.Diagnostics.ProfessionalDetails() {
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
		for _, detail := range report.StreamProbe.ProfessionalDetails() {
			fmt.Fprintln(cmd.OutOrStdout(), detail)
		}
	}
	return nil
}

func readSingleByte(stream transport.TransportStream, buf []byte) (int64, string) {
	n, err := stream.Read(buf[:1])
	if err == nil {
		return int64(n), ""
	}
	return int64(n), err.Error()
}

func writeSingleByte(stream transport.TransportStream) (int64, string) {
	n, err := stream.Write([]byte{0})
	if err == nil {
		return int64(n), ""
	}
	return int64(n), err.Error()
}
