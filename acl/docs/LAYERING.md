# ACL Layering Guide

This document describes where ACL-related code belongs and how the project layers are
intended to interact. Use it when adding new code, moving behavior, or deciding whether
a change belongs in Arduino CLI itself or in ACL.

## Layer Stack

The intended flow is:

`Arduino CLI Layer -> ACL Integration Layer -> ACL Compatibility Layer -> ACL Runtime Layer -> ACL Execution Layer`

The `Platform Layer` is a constraint surface that affects all other layers.

## 1. Arduino CLI Layer

This layer owns the normal Arduino user experience:

- `core install`
- `compile`
- `upload`
- `board`
- `lib`
- `sketch`

Responsibilities:

- stay as close to upstream Arduino CLI as practical
- preserve upstream command structure, flags, and behavior where possible
- call into ACL through narrow integration points when Android-specific behavior is needed

This layer should not contain low-level ELF parsing, runtime package logic, or generic
compatibility classification. If code is inspecting binaries, interpreting loaders, or
reasoning about runtime layout, it does not belong here.

## 2. ACL Integration Layer

This layer connects Arduino CLI lifecycle events to ACL.

Examples:

- install-time patching hooks
- pre-compile validation hooks
- future execution routing hooks

Responsibilities:

- translate Arduino CLI events into ACL requests
- keep the bridge thin and explicit
- avoid hiding ACL behavior inside unrelated CLI code

This layer may know when ACL should run, but it should not reimplement scanner, runtime,
or execution logic. The integration layer should orchestrate, not decide.

## 3. ACL Compatibility Layer

This layer understands toolchains and binary compatibility, not Arduino business logic.

Examples:

- scanner
- compatibility classification
- patch intent
- validation
- toolchain reports

Responsibilities:

- inspect ELF files and related executable artifacts
- classify tool compatibility
- describe patch intent and runtime requirements
- produce reports that can be consumed by integration and runtime layers

Do not put Arduino-specific assumptions into generic ACL compatibility packages unless
they are clearly isolated and documented. The compatibility layer should remain reusable
outside Arduino when practical.

## 4. ACL Runtime Layer

This layer manages runtime packages and the assets they contain.

Examples:

- runtime package builder
- runtime manager
- runtime installer
- loader/runtime assets
- runtime validation

Responsibilities:

- package and verify runtime assets
- install and discover runtime packages
- track runtime selection and activation state
- validate runtime structure and metadata

The runtime layer should not be responsible for Arduino command behavior. It should be
focused on the lifecycle of runtime packages and their verification.

## 5. ACL Execution Layer

This layer decides how a compatible tool should be launched and how execution results
are surfaced.

Examples:

- execution planner
- execution backend
- environment sanitization
- direct exec vs explicit loader strategy
- stdout/stderr/exit-code handling

Responsibilities:

- build a launch plan from compatibility and runtime data
- choose an execution strategy based on validated inputs
- sanitize the environment before launch
- return execution results in a structured way

The execution layer should not silently absorb compatibility or runtime-policy logic.
It consumes validated inputs and turns them into a concrete launch plan or process
invocation.

## 6. Platform Layer

This layer captures host-specific constraints that ACL must respect.

Examples:

- Android and Termux filesystem behavior
- Bionic versus glibc differences
- permissions
- `LD_PRELOAD`
- `LD_LIBRARY_PATH`
- hardware and device restrictions

Responsibilities:

- describe real platform limitations
- constrain what ACL can safely assume
- keep Android-specific behavior visible instead of implicit

The platform layer is not a code bucket. It is the set of host realities that the other
layers must handle explicitly.

## Placement Rules

- Do not put ELF parsing in Arduino CLI command code.
- Do not put Arduino-specific assumptions into generic ACL packages unless they are
  clearly isolated.
- Keep ACL reusable outside Arduino where practical.
- Prefer evidence-based validation over assumptions.
- `STATUS.md` records what is proven.
- `MISSION.md` records why the project exists.
- `AGENTS.md` records how contributors should work.
- `LAYERING.md` records where code belongs.

## Practical Guidance

If you are unsure where code belongs, ask these questions:

1. Is this normal Arduino user-facing behavior? Put it in the Arduino CLI layer.
2. Is this glue between CLI events and ACL? Put it in the integration layer.
3. Is this about identifying or classifying binaries? Put it in the compatibility layer.
4. Is this about packaged runtime assets or runtime state? Put it in the runtime layer.
5. Is this about turning validated inputs into a launch plan or process execution? Put it
   in the execution layer.
6. Is the behavior really a host/platform constraint? Model it in the platform layer and
   keep that constraint explicit.

When a change would span layers, keep each layer responsible for its own part of the
problem and pass structured data across the boundary.
