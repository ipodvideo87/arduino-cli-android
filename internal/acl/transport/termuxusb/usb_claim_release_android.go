//go:build android && cgo

package termuxusb

/*
#cgo pkg-config: libusb-1.0
#include <libusb.h>
#include <stdint.h>
*/
import "C"

import (
	"fmt"
	"strings"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/acl/transport"
)

func inspectUSBClaimRelease(fd int, interfaceNumber int) (USBClaimReleaseReport, error) {
	var ctx *C.libusb_context
	if rc := C.libusb_init(&ctx); rc != 0 {
		return USBClaimReleaseReport{}, fmt.Errorf("libusb_init failed: %s", libusbErrorString(rc))
	}
	defer C.libusb_exit(ctx)

	var handle *C.libusb_device_handle
	if rc := C.libusb_wrap_sys_device(ctx, C.intptr_t(fd), &handle); rc != 0 {
		return USBClaimReleaseReport{}, fmt.Errorf("libusb_wrap_sys_device failed: %s", libusbErrorString(rc))
	}
	if handle == nil {
		return USBClaimReleaseReport{}, fmt.Errorf("libusb_wrap_sys_device returned nil handle")
	}
	defer C.libusb_close(handle)

	device := C.libusb_get_device(handle)
	if device == nil {
		return USBClaimReleaseReport{}, fmt.Errorf("libusb_get_device returned nil")
	}
	var desc C.struct_libusb_device_descriptor
	if rc := C.libusb_get_device_descriptor(device, &desc); rc != 0 {
		return USBClaimReleaseReport{}, fmt.Errorf("libusb_get_device_descriptor failed: %s", libusbErrorString(rc))
	}
	report := USBClaimReleaseReport{
		SchemaVersion:   "1",
		Status:          diagnostics.StatusWarning,
		Provider:        defaultProviderID,
		ProviderKind:    transport.KindAndroidUSBFD,
		InterfaceNumber: interfaceNumber,
		ClaimState:      "not_attempted",
		ReleaseState:    "not_attempted",
		Limitations: []string{
			"no payload transfers were attempted",
			"claim/release diagnostics are experimental",
		},
		Beginner: "interface claim/release validation is experimental",
		Professional: []string{
			"descriptor and interface evidence was collected before claim/release",
		},
		NextStep: "use the claim/release result to decide whether later transfer diagnostics are safe",
		Metadata: map[string]string{
			"bridge_state":        "experimental",
			"claim_release_state": "not_attempted",
		},
	}
	report.Device = transport.DiscoveredDevice{
		TransportFamily: transport.TransportFamilyUSBSerial,
		Permission:      transport.PermissionStateUnknown,
		VID:             uint16(desc.idVendor),
		PID:             uint16(desc.idProduct),
		Manufacturer:    usbStringDescriptor(handle, desc.iManufacturer),
		Product:         usbStringDescriptor(handle, desc.iProduct),
		SerialNumber:    usbStringDescriptor(handle, desc.iSerialNumber),
		Capabilities: transport.CapabilitiesFromList(
			transport.CapabilityDiscovery,
			transport.CapabilityPermission,
			transport.CapabilityUSBHandle,
			transport.CapabilityDescriptorDiscovery,
		),
		Metadata: map[string]string{
			"topology_source":     "libusb",
			"bridge_state":        "experimental",
			"claim_release_state": "not_attempted",
		},
	}
	if strings.TrimSpace(report.Device.Product) == "" {
		report.Device.Product = usbClassName(uint8(desc.bDeviceClass))
	}
	if cfg := activeConfigDescriptor(device, desc.bNumConfigurations); cfg != nil {
		defer C.libusb_free_config_descriptor(cfg)
		interfaces, caps := topologyFromConfig(handle, cfg)
		report.Device.Interfaces = interfaces
		report.Device.Capabilities = mergeTransportCapabilities(report.Device.Capabilities, caps)
	}

	target, ok := findInterfaceDescriptor(report.Device.Interfaces, interfaceNumber)
	if !ok {
		report.Status = diagnostics.StatusFailed
		report.ClaimState = "unsupported"
		report.ReleaseState = "not_attempted"
		report.Warnings = []string{fmt.Sprintf("interface %d was not found in the active configuration", interfaceNumber)}
		report.Limitations = append(report.Limitations, "requested interface not present in topology evidence")
		report.NextStep = "choose an interface reported by the topology bridge before attempting claim/release"
		return report, nil
	}
	if len(target.Endpoints) == 0 {
		report.Professional = append(report.Professional, fmt.Sprintf("interface %d has no data endpoints", interfaceNumber))
	}
	if rc := C.libusb_claim_interface(handle, C.int(interfaceNumber)); rc != 0 {
		report.Status = diagnostics.StatusWarning
		report.ClaimState = "failed"
		report.ReleaseState = "not_attempted"
		report.ClaimError = libusbErrorString(rc)
		report.Warnings = []string{fmt.Sprintf("claim interface %d: %s", interfaceNumber, report.ClaimError)}
		report.Limitations = append(report.Limitations, "interface claim failed before release could be attempted")
		report.NextStep = "use the claim error to decide whether a later milestone should add auto-detach support"
		return report, nil
	}
	report.ClaimState = "claimed"
	if rc := C.libusb_release_interface(handle, C.int(interfaceNumber)); rc != 0 {
		report.Status = diagnostics.StatusWarning
		report.ReleaseState = "failed"
		report.ReleaseError = libusbErrorString(rc)
		report.Warnings = append(report.Warnings, fmt.Sprintf("release interface %d: %s", interfaceNumber, report.ReleaseError))
		report.Limitations = append(report.Limitations, "interface release failed after claim succeeded")
		report.NextStep = "treat the release error as the next architecture question"
		return report, nil
	}
	report.ReleaseState = "released"
	report.Beginner = "TERMUX_USB_FD observed via environment; interface claim/release remains experimental"
	report.Professional = []string{
		"TERMUX_USB_FD was observed and the interface could be claimed and released",
		"no payload transfers were attempted",
	}
	report.NextStep = "use the claim/release evidence to decide whether later transfer diagnostics are safe"
	return report, nil
}

func findInterfaceDescriptor(interfaces []transport.InterfaceSummary, number int) (transport.InterfaceSummary, bool) {
	for _, iface := range interfaces {
		if iface.Number == number {
			return iface, true
		}
	}
	return transport.InterfaceSummary{}, false
}
