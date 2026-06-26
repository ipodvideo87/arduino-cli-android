# Serial Monitor Preview

This is a preview of the future serial monitor workflow that will reuse the
Android USB transport framework.

It is not an implementation.
It does not open a live monitor session.
It does not change runtime behavior.

The transport/session/export contracts now exist as compile-safe ACL
infrastructure, but live monitoring remains a future workflow.

## Goal

The serial monitor workflow should provide a tool-facing view of the same
transport/session model used for upload.

## Intended Flow

```text
Transport discovery
  ↓
Permission acquisition
  ↓
Session open
  ↓
Endpoint export or serial adapter selection
  ↓
Live stream capture
  ↓
Control-line handling where supported
  ↓
Disconnect / reconnect handling
  ↓
Workflow report
```

## Monitor Responsibilities

The monitor workflow should:

- select a transport by capability
- preserve the selected device and interface in diagnostics
- stream data through the exported endpoint
- preserve timestamps when useful
- report control-line state when supported
- handle disconnect and re-enumeration events cleanly
- keep board-specific logic out of the monitor command surface

## Monitor Is Not

Serial monitoring is not:

- upload
- flashing
- board identification
- raw USB descriptor parsing
- transport acquisition

Those are separate responsibilities.

## Behavior To Preserve

The preview should support:

- line coding visibility
- DTR/RTS state when the transport supports them
- raw mode versus filtered mode
- capture-to-file behavior
- reconnect awareness
- tool-friendly terminal output

## Transport Families

The monitor workflow should be able to work across:

- USB serial
- CDC ACM
- RFC2217-compatible endpoints
- PTY-backed exports
- future network transports

## Report Expectations

The serial monitor report should include:

- selected provider
- endpoint type
- capture mode
- control-line state
- reconnect behavior
- warnings
- limitations
- validation level
- beginner summary and professional details

## ACL Integration

The monitor workflow should sit behind the ACL Engine so the GUI can consume a
single workflow report rather than learning Android USB details.

## Practical Conclusion

The monitor architecture should reuse the same provider model as upload.
That keeps serial monitoring consistent with the rest of the transport stack.
