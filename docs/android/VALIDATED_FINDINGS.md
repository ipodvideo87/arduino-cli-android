# Validated Findings

This file is a logbook of tested findings from the project history.

## 2026-06-26

- Environment: native Termux on Samsung A17 / Android 16
- Command/evidence: native Termux compile validation and workflow validation from
  the current project history
- Result: Blink compiles successfully in native Termux
- Confidence: high
- Notes: this is the current native Android compile baseline

## Unknown / from project history

- Environment: native Termux on the target Android device
- Command/evidence: project history references a larger multi-file ESP32-S3 sketch
  including WiFi, ESP Async WebServer, Async TCP, LittleFS, Adafruit NeoPixel, and
  Hash
- Result: native Termux compile works for the esp32ultimate multi-file sketch
- Confidence: medium
- Notes: keep this entry tied to native validation if a fresh reproduction is needed

- Environment: target ESP32-S3 hardware
- Command/evidence: firmware package from the current build was flashed using ESP32
  Flasher
- Result: generated firmware booted and served the expected web response
- Confidence: medium
- Notes: this proves runtime behavior on hardware, but the exact firmware package
  version should be revalidated whenever package semantics change

- Environment: Android USB host / Termux USB access
- Command/evidence: `termux-usb -l` and permission dialogs on the target device
- Result: devices can be enumerated and Android permission behavior is observable,
  but acquisition/open behavior still needs deeper investigation
- Confidence: medium
- Notes: this remains transport work, not a flashing success claim

- Environment: ACL compile workflow
- Command/evidence: workflow compile now generates firmware-package outputs in the
  repository history
- Result: ACL workflow compile produces package outputs, pending the latest native
  validation of full-flash bootloader inclusion
- Confidence: medium
- Notes: package content should be rechecked as firmware-package semantics evolve

- Environment: ESP32-S3 / Arduino-ESP32 flash recipe metadata
- Command/evidence: external flashing history and Arduino-ESP32 recipe conventions
- Result: external flashing previously used bootloader, partitions, boot_app0, and
  application layout
- Confidence: high
- Notes: this is the current canonical flash layout target for full-flash packages

