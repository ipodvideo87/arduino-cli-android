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
	"unsafe"

	"github.com/arduino/arduino-cli/internal/acl/transport"
)

func inspectUSBTopologyImpl(fd int, args []string) (transport.DiscoveredDevice, error) {
	_ = args
	var ctx *C.libusb_context
	if rc := C.libusb_init(&ctx); rc != 0 {
		return transport.DiscoveredDevice{}, fmt.Errorf("libusb_init failed: %s", libusbErrorString(rc))
	}
	defer C.libusb_exit(ctx)

	var handle *C.libusb_device_handle
	if rc := C.libusb_wrap_sys_device(ctx, C.intptr_t(fd), &handle); rc != 0 {
		return transport.DiscoveredDevice{}, fmt.Errorf("libusb_wrap_sys_device failed: %s", libusbErrorString(rc))
	}
	if handle == nil {
		return transport.DiscoveredDevice{}, fmt.Errorf("libusb_wrap_sys_device returned nil handle")
	}
	defer C.libusb_close(handle)

	device := C.libusb_get_device(handle)
	if device == nil {
		return transport.DiscoveredDevice{}, fmt.Errorf("libusb_get_device returned nil")
	}

	var desc C.struct_libusb_device_descriptor
	if rc := C.libusb_get_device_descriptor(device, &desc); rc != 0 {
		return transport.DiscoveredDevice{}, fmt.Errorf("libusb_get_device_descriptor failed: %s", libusbErrorString(rc))
	}

	report := transport.DiscoveredDevice{
		TransportFamily: transport.TransportFamilyUSBSerial,
		Permission:      transport.PermissionStateUnknown,
		Metadata: map[string]string{
			"topology_source":     "libusb",
			"bridge_state":        "experimental",
			"claim_release_state": "not_attempted",
		},
		Capabilities: transport.CapabilitiesFromList(
			transport.CapabilityDiscovery,
			transport.CapabilityPermission,
			transport.CapabilityUSBHandle,
			transport.CapabilityDescriptorDiscovery,
		),
	}
	report.VID = uint16(desc.idVendor)
	report.PID = uint16(desc.idProduct)
	report.Manufacturer = usbStringDescriptor(handle, desc.iManufacturer)
	report.Product = usbStringDescriptor(handle, desc.iProduct)
	report.SerialNumber = usbStringDescriptor(handle, desc.iSerialNumber)
	report.Metadata["usb_spec"] = formatBCD(desc.bcdUSB)
	report.Metadata["device_class"] = usbClassName(uint8(desc.bDeviceClass))
	report.Metadata["device_subclass"] = usbHex(uint8(desc.bDeviceSubClass))
	report.Metadata["device_protocol"] = usbHex(uint8(desc.bDeviceProtocol))
	report.Metadata["max_packet_size_0"] = fmt.Sprintf("%d", desc.bMaxPacketSize0)
	report.Metadata["configuration_count"] = fmt.Sprintf("%d", desc.bNumConfigurations)

	if cfg := activeConfigDescriptor(device, desc.bNumConfigurations); cfg != nil {
		defer C.libusb_free_config_descriptor(cfg)
		interfaces, caps := topologyFromConfig(handle, cfg)
		report.Interfaces = interfaces
		report.Capabilities = mergeTransportCapabilities(report.Capabilities, caps)
		report.Metadata["interface_count"] = fmt.Sprintf("%d", len(report.Interfaces))
		report.Metadata["endpoint_count"] = fmt.Sprintf("%d", countEndpoints(report.Interfaces))
	}

	if strings.TrimSpace(report.Product) == "" {
		report.Product = report.Metadata["device_class"]
	}
	return report, nil
}

func activeConfigDescriptor(device *C.libusb_device, configurations C.uint8_t) *C.struct_libusb_config_descriptor {
	var cfg *C.struct_libusb_config_descriptor
	if rc := C.libusb_get_active_config_descriptor(device, &cfg); rc == 0 && cfg != nil {
		return cfg
	}
	if configurations == 0 {
		return nil
	}
	for i := C.uint8_t(0); i < configurations; i++ {
		if rc := C.libusb_get_config_descriptor(device, i, &cfg); rc == 0 && cfg != nil {
			return cfg
		}
	}
	return nil
}

func topologyFromConfig(handle *C.libusb_device_handle, cfg *C.struct_libusb_config_descriptor) ([]transport.InterfaceSummary, transport.TransportCapabilities) {
	if cfg == nil {
		return nil, transport.TransportCapabilities{}
	}
	ifaceCount := int(cfg.bNumInterfaces)
	if ifaceCount == 0 {
		return nil, transport.TransportCapabilities{}
	}
	ifaceArray := (*[1 << 28]C.struct_libusb_interface)(unsafe.Pointer(cfg._interface))[:ifaceCount:ifaceCount]
	interfaces := make([]transport.InterfaceSummary, 0)
	var capabilities transport.TransportCapabilities
	for _, iface := range ifaceArray {
		altCount := int(iface.num_altsetting)
		if altCount == 0 {
			continue
		}
		altArray := (*[1 << 28]C.struct_libusb_interface_descriptor)(unsafe.Pointer(iface.altsetting))[:altCount:altCount]
		for _, alt := range altArray {
			interfaceSummary := transport.InterfaceSummary{
				Number:      int(alt.bInterfaceNumber),
				Alternate:   int(alt.bAlternateSetting),
				Class:       usbClassName(uint8(alt.bInterfaceClass)),
				Subclass:    usbHex(uint8(alt.bInterfaceSubClass)),
				Protocol:    usbHex(uint8(alt.bInterfaceProtocol)),
				Description: usbStringDescriptor(handle, alt.iInterface),
				Notes: []string{
					fmt.Sprintf("class-code=0x%02x", uint8(alt.bInterfaceClass)),
				},
			}
			if strings.TrimSpace(interfaceSummary.Description) == "" {
				interfaceSummary.Description = interfaceSummary.Class
			}

			endpointCount := int(alt.bNumEndpoints)
			if endpointCount > 0 && alt.endpoint != nil {
				endpointArray := (*[1 << 28]C.struct_libusb_endpoint_descriptor)(unsafe.Pointer(alt.endpoint))[:endpointCount:endpointCount]
				for _, ep := range endpointArray {
					epSummary, cap := endpointSummaryFromDescriptor(ep)
					interfaceSummary.Endpoints = append(interfaceSummary.Endpoints, epSummary)
					capabilities = mergeTransportCapabilities(capabilities, cap)
				}
			}
			if len(interfaceSummary.Endpoints) == 0 {
				interfaceSummary.Notes = append(interfaceSummary.Notes, "no data endpoints were reported")
			}
			interfaces = append(interfaces, interfaceSummary)
		}
	}
	return interfaces, capabilities
}

func endpointSummaryFromDescriptor(ep C.struct_libusb_endpoint_descriptor) (transport.EndpointSummary, transport.TransportCapabilities) {
	address := uint8(ep.bEndpointAddress)
	direction := "out"
	if address&0x80 != 0 {
		direction = "in"
	}
	transferType := usbTransferTypeName(uint8(ep.bmAttributes & 0x3))
	summary := transport.EndpointSummary{
		Address:       address,
		Direction:     direction,
		Type:          transferType,
		MaxPacketSize: int(ep.wMaxPacketSize),
		Interval:      int(ep.bInterval),
		Usage:         transferType,
		Notes: []string{
			fmt.Sprintf("raw_address=0x%02x", address),
		},
	}
	var caps transport.TransportCapabilities
	switch transferType {
	case "bulk":
		caps.BulkTransfer = true
	case "interrupt":
		caps.InterruptTransfer = true
	case "control":
		caps.ControlTransfer = true
	}
	return summary, caps
}

func usbStringDescriptor(handle *C.libusb_device_handle, index C.uint8_t) string {
	if handle == nil || index == 0 {
		return ""
	}
	buf := make([]byte, 256)
	n := C.libusb_get_string_descriptor_ascii(handle, index, (*C.uchar)(unsafe.Pointer(&buf[0])), C.int(len(buf)))
	if n <= 0 {
		return ""
	}
	return strings.TrimSpace(string(buf[:n]))
}

func libusbErrorString(rc C.int) string {
	name := C.libusb_error_name(rc)
	if name != nil {
		return C.GoString(name)
	}
	return fmt.Sprintf("libusb error %d", int(rc))
}

func usbTransferTypeName(kind uint8) string {
	switch kind {
	case 0:
		return "control"
	case 1:
		return "isochronous"
	case 2:
		return "bulk"
	case 3:
		return "interrupt"
	default:
		return usbHex(kind)
	}
}

func usbClassName(code uint8) string {
	switch code {
	case 0x00:
		return "per-interface"
	case 0x01:
		return "audio"
	case 0x02:
		return "communications"
	case 0x03:
		return "hid"
	case 0x05:
		return "physical"
	case 0x06:
		return "image"
	case 0x07:
		return "printer"
	case 0x08:
		return "mass-storage"
	case 0x09:
		return "hub"
	case 0x0a:
		return "cdc-data"
	case 0x0b:
		return "smart-card"
	case 0x0d:
		return "content-security"
	case 0x0e:
		return "video"
	case 0x0f:
		return "personal-health"
	case 0x10:
		return "audio-video"
	case 0x11:
		return "billboard"
	case 0xdc:
		return "diagnostic"
	case 0xe0:
		return "wireless"
	case 0xef:
		return "miscellaneous"
	case 0xfe:
		return "application-specific"
	case 0xff:
		return "vendor-specific"
	default:
		return usbHex(code)
	}
}

func usbHex(v uint8) string {
	return fmt.Sprintf("0x%02x", v)
}

func formatBCD(value C.uint16_t) string {
	return fmt.Sprintf("%x.%02x", int(value>>8), int(value&0xff))
}
