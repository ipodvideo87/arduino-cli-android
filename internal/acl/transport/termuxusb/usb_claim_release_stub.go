//go:build !android || !cgo

package termuxusb

import "fmt"

func inspectUSBClaimRelease(fd int, interfaceNumber int) (USBClaimReleaseReport, error) {
	_ = fd
	_ = interfaceNumber
	return USBClaimReleaseReport{}, fmt.Errorf("interface claim/release requires an Android/Termux build with cgo and libusb support")
}
