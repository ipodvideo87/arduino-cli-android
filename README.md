Arduino CLI Android

A fork of Arduino CLI with experimental Android/Termux support.

Overview

This project explores running Arduino CLI directly on Android devices through Termux, with a focus on compiling and flashing ESP32 and ESP32-S3 firmware without requiring a desktop computer.

The goal is to make Arduino development fully portable and enable web-based IDEs, local development environments, and mobile workflows that can compile and upload firmware directly from Android devices.

Current Status

Working

- Arduino CLI builds successfully on Android (Termux)
- ESP32 board manager index downloads successfully
- ESP32 core installation works
- ESP32-S3 toolchains install successfully
- Xtensa compiler toolchains install successfully
- OpenOCD installs successfully
- GDB installs successfully
- LittleFS and SPIFFS tools install successfully

Android Compatibility Improvements

This fork currently includes Android host compatibility improvements that allow Arduino CLI to use Linux ARM64-compatible toolchains when running under Android.

These changes enable installation of ESP32 platform packages that were previously rejected due to unsupported host detection.

Motivation

The long-term goal is to support projects such as:

- Web-based Arduino IDEs
- Mobile ESP32 development environments
- Browser-based ESP32 flashing tools
- Self-contained embedded development platforms
- Android-native firmware development workflows

Tested Environment

- Android (Termux)
- ARM64 architecture
- ESP32 core version 3.3.10
- ESP32-S3 toolchain installation

Example Result

Successful installation:

Platform esp32:esp32@3.3.10 installed

Installed tools include:

- esp-x32
- esp-rv32
- xtensa-esp-elf-gdb
- riscv32-esp-elf-gdb
- openocd-esp32
- esptool_py
- mkspiffs
- mklittlefs
- ESP32 libraries
- ESP32-S3 libraries

Building

git clone https://github.com/ipodvideo87/arduino-cli-android.git
cd arduino-cli-android

go build -o arduino-cli .

Disclaimer

This project is experimental and is not affiliated with Arduino SA.

Use at your own risk. Android support is still under active investigation and development.

Future Work

- Complete Android platform support
- Improve package installation compatibility
- Validate sketch compilation on Android
- Validate flashing ESP32 devices from Android
- Integrate with web-based IDE projects
- Upstream compatible improvements where possible

License

This project remains subject to the original Arduino CLI license and any licenses of included dependencies.
