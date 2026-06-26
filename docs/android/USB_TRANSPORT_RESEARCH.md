# USB Transport Research

This document captures the research basis for a production-quality Android USB
transport framework.

It is intentionally architecture-focused:

- research first
- design second
- implementation later
- no USB flashing, no upload implementation, no serial monitor implementation
- no runtime behavior changes

## Evidence Legend

- Confirmed device evidence: observed on native Termux or on an Android device.
- Confirmed repo evidence: verified through repository code, tests, or docs.
- Inference: a reasoned conclusion that still depends on later implementation.

## 1. Android USB Host API

Android USB host access is app-scoped, permission-gated, and device-oriented.
The app should think in terms of devices, interfaces, endpoints, and transfers
instead of `/dev/tty*` paths.

Relevant API surface:

- `UsbManager`
- `UsbDevice`
- `UsbInterface`
- `UsbEndpoint`
- `UsbDeviceConnection`
- `UsbRequest`

Key behavior:

- device discovery starts with the connected-device inventory
- attach and detach events are delivered as USB host intents
- permission must be requested before opening the connection
- permission is not a global system grant; it is tied to the app/device session
- interfaces and endpoints must be enumerated before choosing a transport path
- bulk, interrupt, and control transfers are the primitive I/O operations

Primary sources:

- Android USB host overview:
  <https://developer.android.com/develop/connectivity/usb/host>
- `UsbManager`:
  <https://developer.android.com/reference/android/hardware/usb/UsbManager>
- `UsbDeviceConnection`:
  <https://developer.android.com/reference/android/hardware/usb/UsbDeviceConnection>
- `UsbInterface`:
  <https://developer.android.com/reference/android/hardware/usb/UsbInterface>
- `UsbEndpoint`:
  <https://developer.android.com/reference/android/hardware/usb/UsbEndpoint>
- `UsbRequest`:
  <https://developer.android.com/reference/android/hardware/usb/UsbRequest>

### Research Conclusions

- Discovery, permission, connection, and protocol handling must be separate
  layers.
- The framework should not assume a serial device exists as a filesystem node.
- A transport can be valid even when it is not a traditional USB-to-UART chip.
- Diagnostics must preserve the selected device, interface, endpoint, and
  permission source.

## 2. Termux USB

Termux remains the most practical Android-native acquisition surface for this
project.

Relevant observations:

- `termux-usb` provides a Termux-native USB acquisition path.
- Termux API uses file-descriptor passing via `TERMUX_USB_FD` and the callback
  mechanism.
- The Termux ecosystem already demonstrates a path for permission-mediated USB
  file descriptors without forcing a companion APK first.

Primary sources:

- Termux API package implementation:
  <https://github.com/termux/termux-api-package/blob/master/termux-api.c>
- Termux project site:
  <https://termux.dev/en/>
- Termux documentation:
  <https://termux.dev/en/docs/>

### Research Conclusions

- Termux USB should be treated as the first acquisition provider, not the only
  possible transport model.
- File-descriptor handoff is a useful boundary because it keeps permission
  acquisition and USB lifetime separate from protocol tooling.
- Native Termux behavior remains the final authority for Android behavior.

## 3. USB Serial Chipsets

The framework needs to support common USB-to-serial chipsets and native USB
device implementations without hardcoding board families into the transport
layer.

Representative families:

| Family | Typical shape | Transport implication |
| --- | --- | --- |
| CP210x | USB-to-UART bridge | generic serial provider with a chipset probe |
| CH340 | USB-to-UART bridge | generic serial provider with a chipset probe |
| CH9102 | USB-to-UART bridge | generic serial provider with a chipset probe |
| FT232 | USB-to-UART bridge | generic serial provider with a chipset probe |
| CDC ACM | class-compliant serial | generic CDC provider |
| Native USB devices | board-native USB serial / debug / composite interfaces | protocol-specific providers layered above the same transport model |
| ESP32-S2/S3 | USB Serial/JTAG or CDC-style USB roles depending on firmware/core mode | serial, JTAG, or composite provider paths |
| RP2040 | native USB CDC or composite USB roles | CDC/serial provider path |
| STM32 | CDC, DFU, HID, or composite USB roles depending on firmware | multiple provider families, not one serial-only path |

Primary source categories:

- usb-serial-for-android:
  <https://github.com/mik3y/usb-serial-for-android>
- Espressif documentation for ESP32-S2/S3 USB roles:
  <https://docs.espressif.com/>
- Raspberry Pi documentation for RP2040 USB behavior:
  <https://www.raspberrypi.com/documentation/>
- ST documentation for STM32 USB roles:
  <https://www.st.com/>

### Research Conclusions

- Board family should not be the first selector.
- Capability and interface shape should drive selection.
- The transport layer should be able to express serial, CDC ACM, DFU, HID,
  JTAG/SWD, and future composite roles without redesigning the selection model.

## 4. Existing Open-Source Architecture Patterns

The useful pattern across the existing ecosystem is not any one codebase, but
the split between:

- discovery
- transport selection
- protocol-specific adapters
- tool-facing serial endpoints
- optional remote or socket bridging

### usb-serial-for-android

- Uses a serial abstraction with device-specific drivers under a common API.
- Demonstrates that driver-style discovery is more scalable than hardcoding a
  single VID/PID path.

Source:

- <https://github.com/mik3y/usb-serial-for-android>

### PlatformIO

- Separates upload and monitor configuration from board identity.
- Auto-detects upload and monitor ports when possible.
- Supports explicit upload/monitor port configuration, filters, and remote
  serial workflows.

Sources:

- `upload_port`:
  <https://docs.platformio.org/en/latest/projectconf/sections/env/options/upload/upload_port.html>
- `pio run` upload/monitor options:
  <https://docs.platformio.org/en/latest/core/userguide/cmd_run.html>
- `pio device monitor`:
  <https://docs.platformio.org/en/stable/core/userguide/device/cmd_monitor.html>
- `pio remote device`:
  <https://docs.platformio.org/en/latest/core/userguide/remote/cmd_device.html>

### Arduino IDE / Arduino CLI

- Upload behavior is recipe-driven.
- The board definition decides the command and flags; the user-facing tool
  should not need to know transport internals.

Relevant repository evidence:

- `commands/service_upload.go`
- ESP32 platform recipe test data in this repository

### esptool

- Uses a serial port abstraction.
- Supports RFC2217 remote serial ports.
- Exposes the right mental model for a transport bridge that is not literally
  a `/dev/tty*` device.

Source:

- <https://docs.espressif.com/projects/esptool/en/latest/esp32/remote-serial-ports.html>

### pySerial

- Reinforces the serial-port abstraction and shows that RFC2217 is a viable
  compatibility boundary for non-local serial endpoints.

Source:

- <https://pyserial.readthedocs.io/>

### libusb

- Reinforces low-level device, interface, endpoint, and hotplug semantics.
- Useful when the bridge needs descriptor access beyond the serial abstraction.

Source:

- <https://libusb.info/>

### Research Conclusions

- A transport framework should own discovery and acquisition.
- Protocol adapters should sit on top of a transport session, not inside the
  transport registry itself.
- A serial-shaped tool endpoint remains the best default contract.

## 5. Future Transport Types

The architecture must be able to host more than USB serial.

Planned provider families:

- USB Serial
- CDC ACM
- DFU
- HID
- CMSIS-DAP
- JTAG
- SWD
- Network transports
- Bluetooth transports
- future wireless transports

### Research Conclusions

- The selection model should describe capabilities, not just board families.
- Multiple providers may coexist for one device.
- The same device may expose different valid providers depending on firmware
  mode and permission state.

## 6. ACL Integration

The transport subsystem should integrate with ACL as a reusable layer, not as
board-specific command glue.

Current repository shape:

- `internal/acl/transport` is already a capability-based selector and registry.
- `internal/acl/engine` is the workflow orchestration boundary.
- `internal/acl/compatibility` already carries transport-related compatibility
  decisions.
- `FirmwarePackage` already carries a transport hint field for future tooling
  layers.

Integration points:

- Workflow Engine
- Firmware Package
- Validation Providers
- Compatibility Layer
- Future Validation Engine
- Future GUI/workspaces

### Research Conclusions

- The GUI should consume transport summaries and workflow reports, not Android
  USB APIs.
- The engine should choose transports by capability and policy.
- Transport diagnostics should be machine-readable and preserved alongside
  workflow evidence.

## 7. Stable Architecture Conclusion

The stable architecture is:

1. discover transport candidates
2. request and record permission
3. open and hold a session
4. identify protocol and endpoint shape
5. adapt to the tool-facing contract
6. emit diagnostics and lifecycle events
7. let workflows consume the selected transport, not the raw USB API

This is the foundation for future upload, monitor, diagnostics, and device
interaction work.

It is not yet implementation.

