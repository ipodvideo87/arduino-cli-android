# ADR 0001: ACL Architecture Direction

## Status
Proposed

## Context
Sprint 1 established a key fact set:

- The copied Termux/glibc runtime is valid ELF, but it is not relocatable as-is.
- The runtime contains hardcoded Termux paths such as `/data/data/com.termux/files/usr/glibc`.
- Copying `ld-linux-aarch64.so.1` and companion libraries is not sufficient to produce a portable runtime.
- ELF patching can succeed while execution still fails at load time.
- The current ACL tooling can detect these conditions reliably.

This means ACL should not continue as a patch-on-top-of-copied-runtime project. The next architecture must be chosen from first principles.

## Executive Summary
The best long-term direction is a layered ACL architecture:

1. A small Android-native ACL core that owns inspection, policy, verification, orchestration, and tool selection.
2. A purpose-built ACL runtime for Linux binaries that is packaged and relocated as an ACL-owned asset, not copied from Termux.
3. Native Android builds against bionic whenever a tool can be rebuilt practically.

This is the only approach that satisfies the project goals simultaneously:

- generic enough for Arduino CLI and future Linux tools,
- maintainable enough for long-term support,
- portable enough to be distributed on Android,
- and honest about the fact that some Linux binaries need a compatibility runtime while others should be rebuilt natively.

## Decision
Adopt a hybrid architecture with three execution paths:

1. **Native Android path** for tools that can be rebuilt against bionic or made portable with modest source changes.
2. **ACL runtime path** for Linux binaries that still require glibc semantics.
3. **Reject path** for binaries that are unsafe, too coupled, or not worth supporting.

Do not treat copied Termux glibc files as the runtime architecture. They are only an investigation artifact.

## Architecture Overview
### Layer 1: ACL Core
Android-native orchestration layer that:

- scans ELF files,
- classifies dependencies and interpreters,
- checks compatibility policy,
- decides whether native, runtime, patch, or reject applies,
- records evidence and error diagnostics.

### Layer 2: Compatibility Runtime
A purpose-built ACL runtime bundle that is:

- built and versioned by ACL,
- relocatable by design,
- free of embedded Termux path assumptions,
- validated independently from patching.

### Layer 3: Native Rebuild Track
Tools and dependencies that can reasonably be rebuilt for bionic should be moved there first. This reduces the total compatibility burden and improves performance and security.

## Approaches Considered
### 1. Purpose-built ACL runtime
**Recommendation: yes, but only as part of a layered design.**

Advantages:
- Best path for generic Linux tool support.
- Can be designed for relocatability from the start.
- Keeps ACL reusable beyond Arduino CLI.
- Preserves glibc semantics where needed.

Disadvantages:
- Highest engineering effort.
- Needs careful packaging, relocation, and validation work.
- Still carries licensing and redistribution responsibilities.

Technical risks:
- Loader/path assumptions may still leak in if not tested rigorously.
- Runtime maintenance becomes a long-term project.

### 2. Rebuild tools against Android bionic
**Recommendation: use whenever practical, not as the sole strategy.**

Advantages:
- Best security and maintainability.
- Best alignment with Android’s native environment.
- Lowest runtime overhead.

Disadvantages:
- Not all Linux tools are realistically portable.
- Glibc-specific assumptions can be deep and time-consuming to remove.
- Arduino ecosystem tools are heterogeneous; one strategy will not fit all.

Technical risks:
- Upstream divergence.
- Toolchain-specific compatibility breakage.

### 3. Compatibility shim libraries
**Recommendation: selective use only.**

Advantages:
- Useful for narrow API gaps.
- Can reduce patching pressure for small incompatibilities.

Disadvantages:
- Easy to become a pile of hacks.
- Hard to reason about long-term behavior.
- Does not solve loader/runtime relocation problems.

Technical risks:
- Hidden behavior differences.
- Maintenance burden grows faster than capability.

### 4. Runtime virtualization
**Recommendation: not the primary architecture.**

Advantages:
- Highest compatibility in the short term.
- Lowest immediate source changes.

Disadvantages:
- Conflicts with the project’s Android-first mission.
- Adds heavy operational and security complexity.
- Reduces portability and user experience.

Technical risks:
- Performance overhead.
- Increased attack surface.
- Dependency on external host services or kernel features.

### 5. Continue copying Termux glibc assets
**Recommendation: reject.**

Advantages:
- Fastest path for experiments.

Disadvantages:
- Not relocatable as-is.
- Couples ACL to Termux filesystem assumptions.
- Does not scale into a production architecture.

Technical risks:
- False confidence from ELF-level success.
- Runtime failures that appear only at execution time.

## Recommendation Rationale
The evidence shows that ACL must separate three concerns:

- binary analysis,
- compatibility policy,
- and runtime execution.

If those remain fused, the project will keep producing partial wins that do not translate into a portable platform. A purpose-built ACL runtime gives the project a controlled compatibility substrate, while native bionic rebuilds remove unnecessary complexity for tools that do not need glibc.

This combination is the best long-term balance of:

- portability,
- maintainability,
- performance,
- security,
- and ecosystem reach.

## Risks
### Long-term maintainability
The main risk is allowing the runtime layer to become a second operating system. ACL should instead remain a thin compatibility substrate with explicit policy and diagnostics.

### Portability
Portability fails if the runtime embeds absolute paths or distro-specific assumptions. Relocatability must be tested separately from ELF patching.

### Performance
Native bionic builds should be preferred when possible. The ACL runtime path should be reserved for binaries that truly need it.

### Security
Runtime shims and loaders expand the attack surface. ACL should validate inputs, minimize privileged behavior, and avoid opaque behavior.

### Licensing / redistribution
Any runtime bundle and copied libraries must be audited for licensing, provenance, and redistribution rights before release. This is especially important if any non-Android system libraries are shipped.

### Arduino ecosystem compatibility
Arduino CLI and its toolchain ecosystem contain a mix of portable Go code, shell scripts, Python, Java, native binaries, and vendor-specific utilities. A layered architecture allows ACL to support each category appropriately instead of forcing one execution model.

### Future Linux tooling
The recommended architecture is generic enough to support future tools because the runtime is not Arduino-specific. The decision engine should classify tools by their actual dependency profile, not by application name.

## Recommended Implementation Roadmap
1. Define ACL compatibility policy and classification rules.
2. Complete the runtime relocation verification harness.
3. Specify ACL runtime packaging requirements, including relocatability.
4. Identify tools that can move to native bionic builds first.
5. Build a minimal purpose-built ACL runtime prototype.
6. Verify one real tool end to end using `scan -> patch plan -> verify -> execute`.
7. Expand coverage to Arduino CLI and related toolchains.

## Next Sprint Recommendation
Do not implement the full runtime yet. The next sprint should produce:

- a runtime design specification,
- a relocation test matrix,
- and a classification policy for deciding native build vs ACL runtime vs reject.

That work will turn the current evidence into a concrete platform strategy before significant runtime engineering begins.
