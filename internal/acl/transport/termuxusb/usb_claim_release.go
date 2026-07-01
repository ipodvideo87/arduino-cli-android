package termuxusb

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/acl/transport"
)

type USBClaimReleaseReport struct {
	SchemaVersion    string                     `json:"schema_version,omitempty"`
	Status           diagnostics.Status         `json:"status,omitempty"`
	Provider         string                     `json:"provider,omitempty"`
	ProviderKind     transport.Kind             `json:"provider_kind,omitempty"`
	Device           transport.DiscoveredDevice `json:"device,omitempty"`
	InterfaceNumber  int                        `json:"interface_number,omitempty"`
	TermuxUSBCommand string                     `json:"termux_usb_command,omitempty"`
	ClaimState       string                     `json:"claim_state,omitempty"`
	ReleaseState     string                     `json:"release_state,omitempty"`
	ClaimError       string                     `json:"claim_error,omitempty"`
	ReleaseError     string                     `json:"release_error,omitempty"`
	FDEnvPresent     bool                       `json:"fd_env_present,omitempty"`
	FDEnvValue       string                     `json:"fd_env_value,omitempty"`
	FDObserved       bool                       `json:"fd_observed,omitempty"`
	FDValid          bool                       `json:"fd_valid,omitempty"`
	FDInspectable    bool                       `json:"fd_inspectable,omitempty"`
	FDSource         string                     `json:"fd_source,omitempty"`
	HandoffMode      string                     `json:"handoff_mode,omitempty"`
	Warnings         []string                   `json:"warnings,omitempty"`
	Limitations      []string                   `json:"limitations,omitempty"`
	Traces           []transport.CommandTrace   `json:"traces,omitempty"`
	Beginner         string                     `json:"beginner_summary,omitempty"`
	Professional     []string                   `json:"professional_details,omitempty"`
	NextStep         string                     `json:"next_step,omitempty"`
	Metadata         map[string]string          `json:"metadata,omitempty"`
}

func HelperClaimReleaseReportFromEnv(interfaceNumber int) USBClaimReleaseReport {
	return HelperClaimReleaseReportFromInvocation(nil, interfaceNumber)
}

func HelperClaimReleaseReportFromInvocation(args []string, interfaceNumber int) USBClaimReleaseReport {
	rawEnv := strings.TrimSpace(os.Getenv("TERMUX_USB_FD"))
	rawArg, argPresent, argMalformed := probeFDFromArgs(args)
	report := USBClaimReleaseReport{
		SchemaVersion:   "1",
		Status:          diagnostics.StatusWarning,
		Provider:        defaultProviderID,
		ProviderKind:    transport.KindAndroidUSBFD,
		InterfaceNumber: interfaceNumber,
		ClaimState:      "not_attempted",
		ReleaseState:    "not_attempted",
		Warnings:        []string{},
		Limitations: []string{
			"no payload transfers were attempted",
			"claim/release diagnostics are experimental",
		},
		Beginner: "interface claim/release validation is experimental",
		Professional: []string{
			"claim/release is diagnostics-only and does not send payload data",
		},
		NextStep: "compare the claim/release outcome with native Termux device behavior",
		Metadata: map[string]string{},
	}
	report.Metadata["helper_args"] = strings.Join(args, " ")
	report.Metadata["interface_number"] = strconv.Itoa(interfaceNumber)
	switch {
	case rawEnv != "":
		report.FDEnvPresent = true
		report.FDEnvValue = rawEnv
		report.FDObserved = true
		report.FDSource = "environment"
		report.HandoffMode = "env"
		report.Metadata["fd_source"] = "environment"
		report.Metadata["handoff_mode"] = "env"
		fd, err := strconv.Atoi(rawEnv)
		if err != nil {
			report.Status = diagnostics.StatusFailed
			report.Beginner = "TERMUX_USB_FD is invalid"
			report.ClaimState = "failed"
			report.ReleaseState = "not_attempted"
			report.Warnings = []string{"TERMUX_USB_FD could not be parsed"}
			report.Limitations = []string{"fd value is not a valid integer"}
			report.Professional = []string{
				"TERMUX_USB_FD could not be parsed",
				"raw value: " + rawEnv,
			}
			report.NextStep = "fix the helper invocation before attempting interface claim/release"
			return report
		}
		report.FDValid = true
		device, err := inspectUSBClaimRelease(fd, interfaceNumber)
		if err != nil {
			report.Status = diagnostics.StatusFailed
			report.Beginner = "TERMUX_USB_FD observed, but interface claim/release is unavailable"
			report.ClaimState = "failed"
			report.ReleaseState = "not_attempted"
			report.Warnings = []string{err.Error()}
			report.Limitations = append(report.Limitations, "interface claim/release could not be inspected")
			report.Professional = []string{
				"TERMUX_USB_FD was observed",
				"interface claim/release could not wrap the fd",
			}
			report.NextStep = "use the native Android build with libusb support to inspect claim/release behavior"
			return report
		}
		report.FDInspectable = true
		report.Device = device.Device
		report.ClaimState = device.ClaimState
		report.ReleaseState = device.ReleaseState
		report.ClaimError = device.ClaimError
		report.ReleaseError = device.ReleaseError
		report.Beginner = device.Beginner
		report.Professional = device.Professional
		report.Warnings = appendUniqueStrings(report.Warnings, device.Warnings...)
		report.Limitations = appendUniqueStrings(report.Limitations, device.Limitations...)
		report.NextStep = device.NextStep
		if report.Device.StableID == "" {
			report.Device.StableID = deviceStableID(report.Device, args)
		}
		if report.Device.DisplayName == "" {
			report.Device.DisplayName = deviceDisplayName(report.Device)
		}
		return report
	case argPresent && !argMalformed:
		report.FDObserved = true
		report.FDSource = "argument"
		report.HandoffMode = "argv"
		report.Metadata["fd_source"] = "argument"
		report.Metadata["handoff_mode"] = "argv"
		fd, err := strconv.Atoi(rawArg)
		if err != nil {
			report.Status = diagnostics.StatusFailed
			report.Beginner = "fd argument is invalid"
			report.ClaimState = "failed"
			report.ReleaseState = "not_attempted"
			report.Warnings = []string{"file descriptor argument could not be parsed"}
			report.Limitations = []string{"fd argument is not a valid integer"}
			report.Professional = []string{
				"fd argument could not be parsed",
				"raw argument: " + rawArg,
			}
			report.NextStep = "fix the helper invocation before attempting interface claim/release"
			return report
		}
		report.FDValid = true
		device, err := inspectUSBClaimRelease(fd, interfaceNumber)
		if err != nil {
			report.Status = diagnostics.StatusFailed
			report.Beginner = "fd argument observed, but interface claim/release is unavailable"
			report.ClaimState = "failed"
			report.ReleaseState = "not_attempted"
			report.Warnings = []string{err.Error()}
			report.Limitations = append(report.Limitations, "interface claim/release could not be inspected")
			report.Professional = []string{
				"fd argument was observed",
				"interface claim/release could not wrap the fd",
			}
			report.NextStep = "use the native Android build with libusb support to inspect claim/release behavior"
			return report
		}
		report.FDInspectable = true
		report.Device = device.Device
		report.ClaimState = device.ClaimState
		report.ReleaseState = device.ReleaseState
		report.ClaimError = device.ClaimError
		report.ReleaseError = device.ReleaseError
		report.Beginner = device.Beginner
		report.Professional = device.Professional
		report.Warnings = appendUniqueStrings(report.Warnings, device.Warnings...)
		report.Limitations = appendUniqueStrings(report.Limitations, device.Limitations...)
		report.NextStep = device.NextStep
		if report.Device.StableID == "" {
			report.Device.StableID = deviceStableID(report.Device, args)
		}
		if report.Device.DisplayName == "" {
			report.Device.DisplayName = deviceDisplayName(report.Device)
		}
		return report
	case argPresent && argMalformed:
		report.FDObserved = true
		report.FDSource = "argument"
		report.HandoffMode = "argv"
		report.Metadata["fd_source"] = "argument"
		report.Metadata["handoff_mode"] = "argv"
		report.Status = diagnostics.StatusFailed
		report.Beginner = "fd argument is invalid"
		report.ClaimState = "failed"
		report.ReleaseState = "not_attempted"
		report.Warnings = []string{"file descriptor argument could not be parsed"}
		report.Limitations = []string{"fd argument is not a valid integer"}
		report.Professional = []string{
			"fd argument could not be parsed",
			"raw argument: " + rawArg,
		}
		report.NextStep = "fix the helper invocation before attempting interface claim/release"
		return report
	default:
		report.Beginner = "TERMUX_USB_FD is not set"
		report.ClaimState = "unavailable"
		report.ReleaseState = "unavailable"
		report.Limitations = []string{"fd handoff is unavailable outside termux-usb -e"}
		report.Professional = []string{
			"the helper did not observe an fd handoff",
			"run the helper via termux-usb -r -E to inspect TERMUX_USB_FD and claim/release behavior",
		}
		report.FDSource = "none"
		report.HandoffMode = "env"
		report.Metadata["fd_source"] = "none"
		report.Metadata["handoff_mode"] = "env"
		report.NextStep = "run the helper through termux-usb -r -E -e so it receives TERMUX_USB_FD"
		return report
	}
}

func parseUSBClaimReleaseReport(raw string) (USBClaimReleaseReport, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return USBClaimReleaseReport{}, errors.New("empty USB claim/release output")
	}
	var report USBClaimReleaseReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		return USBClaimReleaseReport{}, err
	}
	return report, nil
}

func claimReleaseReportFromHelper(helper USBClaimReleaseReport) transport.InterfaceClaimReleaseReport {
	metadata := make(map[string]string, len(helper.Metadata))
	for k, v := range helper.Metadata {
		metadata[k] = v
	}
	return transport.InterfaceClaimReleaseReport{
		SchemaVersion:   helper.SchemaVersion,
		Status:          helper.Status,
		Provider:        helper.Provider,
		ProviderKind:    helper.ProviderKind,
		Device:          helper.Device,
		InterfaceNumber: helper.InterfaceNumber,
		ClaimState:      helper.ClaimState,
		ReleaseState:    helper.ReleaseState,
		ClaimError:      helper.ClaimError,
		ReleaseError:    helper.ReleaseError,
		Warnings:        append([]string(nil), helper.Warnings...),
		Limitations:     append([]string(nil), helper.Limitations...),
		Traces:          append([]transport.CommandTrace(nil), helper.Traces...),
		Beginner:        helper.Beginner,
		Professional:    append([]string(nil), helper.Professional...),
		NextStep:        helper.NextStep,
		Metadata:        metadata,
	}
}
