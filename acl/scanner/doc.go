// Package scanner provides ELF inspection, compatibility classification, and
// machine-readable JSON report generation for the Android Compatibility Layer
// (ACL).
//
// The primary entry point for callers that want the full pipeline is ScanPaths,
// which accepts a list of file paths and returns a ScanReport suitable for
// JSON serialisation via MarshalReport.
//
// Individual pipeline stages are exported for use in other ACL packages:
//   - InspectFile     — raw ELF metadata extraction
//   - ClassifyFile    — compatibility category assignment
//   - FindMissingSymbols — glibc-only DT_NEEDED detection
//   - Recommend       — concrete patch action recommendation
//   - BuildSummary    — aggregate statistics
//
// See README.md for full documentation, JSON schema, and usage examples.
package scanner
