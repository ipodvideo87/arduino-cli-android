// Package scanner provides ELF and script compatibility scanning for the
// Android Compatibility Layer (ACL).
//
// # Overview
//
// The scanner classifies binaries and scripts by their Android compatibility,
// assigns a patch class, and emits structured reports (JSON or human-readable
// text) that downstream tools (patcher, verifier, launcher) can consume.
//
// # Compatibility categories
//
//   - native Android compatible — binary already targets Android/Bionic
//   - Linux/glibc executable   — needs interpreter and/or RPATH patching
//   - static ELF               — no dynamic linking concerns
//   - script                   — shebang-detected file (bash, python, perl, …)
//   - unknown                  — classification failed
//   - unsupported              — foreign format (Windows .exe, etc.)
//
// # Script shebang validation
//
// For files classified as "script", the scanner also validates the interpreter
// declared in the shebang line against the Termux $PREFIX and an optional ACL
// runtime directory.  The outcome is recorded in the per-entry
// interpreter_status field of the JSON report.
//
// Resolution order:
//  1. If the interpreter path exists literally on the filesystem → "found".
//  2. If an ACL runtime directory is provided and the interpreter's basename
//     is found under <runtimeDir>/bin/ → "remapped".
//  3. If the shebang uses "/usr/bin/env <cmd>", look up <cmd> in the
//     envDelegates table and check for it under <prefixDir>/bin/ → "remapped",
//     or → "missing" if the table entry exists but the file is absent.
//  4. Check the wellKnownTermuxPaths table for a Termux-relative equivalent
//     under <prefixDir>/ → "remapped" or "missing" with an install hint.
//  5. Basename fallback: try <prefixDir>/bin/<basename> → "remapped".
//  6. Nothing found → "missing".
//
// # JSON report schema
//
// The top-level Report struct is versioned ("schema_version": "1.0").
// Each ReportEntry for a script carries an optional InterpreterStatus object:
//
//	"interpreter_status": {
//	  "declared_path": "/usr/bin/python3",
//	  "args": [],
//	  "status": "remapped",          // "found" | "missing" | "remapped"
//	  "resolved_path": "/data/data/com.termux/files/usr/bin/python3",
//	  "recommendation": "..."
//	}
//
// The summary block contains aggregate counters:
//
//	"script_interpreter_found":   <int>
//	"script_interpreter_missing": <int>
//	"script_interpreter_remapped":<int>
//
// # Entry points
//
//   - NewReportBuilder(target, prefixDir, runtimeDir) *ReportBuilder
//     Creates a builder.  prefixDir should be the Termux $PREFIX; empty string
//     disables live filesystem resolution (useful for offline/CI analysis).
//
//   - (*ReportBuilder).AddScriptEntry(path) error
//     Records a script file, automatically running shebang validation.
//
//   - (*ReportBuilder).AddELFEntry(…) — Records a classified ELF binary.
//
//   - (*ReportBuilder).ScanFile(path) error
//     Auto-detects file type (script vs ELF magic) and calls the appropriate
//     add method.
//
//   - (*ReportBuilder).Build() Report — Finalises and returns the Report.
//
//   - WriteJSON(w, report) error — Serialise to indented JSON.
//   - WriteText(w, report) error — Serialise to human-readable text.
//
// Shebang-only API (lower-level):
//
//   - ReadShebang(path) (string, error) — Read the shebang line from a file.
//   - ParseShebang(line) (path, args)   — Split shebang into interpreter + args.
//   - CheckInterpreter(path, args, prefix, runtime) ShebangResult
//   - ScanShebang(filePath, prefix, runtime) (*ShebangResult, error)
package scanner
