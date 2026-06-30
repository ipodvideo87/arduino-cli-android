//go:build !android || !cgo

package termuxusb

import "fmt"

import "github.com/arduino/arduino-cli/internal/acl/transport"

func inspectUSBTopologyImpl(fd int, _ []string) (transport.DiscoveredDevice, error) {
	_ = fd
	return transport.DiscoveredDevice{}, fmt.Errorf("USB topology inspection requires an Android/Termux build with cgo and libusb support")
}
