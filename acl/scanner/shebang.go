// Package scanner provides ELF and script compatibility scanning for the
// Android Compatibility Layer (ACL).
//
// This file implements shebang/script interpreter detection and validation.
// It checks whether the interpreter declared in a script's shebang line exists
// under the Termux $PREFIX or an ACL runtime directory, and suggests
// Termux-relative replacement paths when it does not.
package scanner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InterpreterStatus describes whether the shebang interpreter was found,
// is missing, or was remapped to a Termux-relative path.
type InterpreterStatus string

const (
	// InterpreterFound means the interpreter path exists as-is (already
	// under Termux PREFIX or the ACL runtime dir).
	InterpreterFound InterpreterStatus = "found"

	// InterpreterMissing means the interpreter was not found anywhere
	// under PREFIX or the ACL runtime, and no known remap is available.
	InterpreterMissing InterpreterStatus = "missing"

	// InterpreterRemapped means the interpreter was not found at the
	// declared path but a Termux-relative equivalent was located.
	InterpreterRemapped InterpreterStatus = "remapped"
)

// ShebangResult holds the parsed shebang line and the validation outcome.
type ShebangResult struct {
	// Raw is the full shebang line including the "#!" prefix.
	Raw string

	// InterpreterPath is the interpreter executable path extracted from the
	// shebang (e.g. "/usr/bin/python3", "/bin/bash", "/usr/bin/env").
	InterpreterPath string

	// InterpreterArgs are any arguments after the interpreter path on the
	// shebang line (e.g. "python3" when the shebang is "#!/usr/bin/env python3").
	InterpreterArgs []string

	// Status is the resolution outcome: found, missing, or remapped.
	Status InterpreterStatus

	// ResolvedPath is the effective path after resolution.  It equals
	// InterpreterPath when Status == InterpreterFound, the Termux-relative
	// path when Status == InterpreterRemapped, and empty when missing.
	ResolvedPath string

	// Recommendation is a human-readable description of what should be done.
	Recommendation string
}

// wellKnownTermuxPaths maps common Linux interpreter paths to their
// Termux-relative equivalents under $PREFIX.  All values are relative to
// the Termux PREFIX root (i.e. what you would find under
// /data/data/com.termux/files/usr).
var wellKnownTermuxPaths = map[string]string{
	// Shells
	"/bin/sh":              "bin/sh",
	"/bin/bash":            "bin/bash",
	"/usr/bin/bash":        "bin/bash",
	"/usr/bin/sh":          "bin/sh",
	"/usr/local/bin/bash":  "bin/bash",
	"/usr/local/bin/sh":    "bin/sh",
	"/usr/bin/zsh":         "bin/zsh",
	"/bin/zsh":             "bin/zsh",
	"/usr/local/bin/zsh":   "bin/zsh",
	"/usr/bin/fish":        "bin/fish",
	"/usr/bin/dash":        "bin/dash",
	"/bin/dash":            "bin/dash",
	"/usr/bin/ksh":         "bin/ksh",
	"/bin/ksh":             "bin/ksh",

	// Python
	"/usr/bin/python":       "bin/python",
	"/usr/bin/python3":      "bin/python3",
	"/usr/bin/python2":      "bin/python2",
	"/usr/local/bin/python": "bin/python",
	"/usr/local/bin/python3": "bin/python3",
	"/usr/local/bin/python2": "bin/python2",
	"/usr/bin/python3.11":   "bin/python3.11",
	"/usr/bin/python3.10":   "bin/python3.10",
	"/usr/bin/python3.9":    "bin/python3.9",

	// Perl
	"/usr/bin/perl":        "bin/perl",
	"/usr/local/bin/perl":  "bin/perl",

	// Ruby
	"/usr/bin/ruby":        "bin/ruby",
	"/usr/local/bin/ruby":  "bin/ruby",

	// Node.js
	"/usr/bin/node":        "bin/node",
	"/usr/local/bin/node":  "bin/node",
	"/usr/bin/nodejs":      "bin/node",

	// Awk / sed / other POSIX utilities often used as script interpreters
	"/usr/bin/awk":         "bin/awk",
	"/bin/awk":             "bin/awk",
	"/usr/bin/gawk":        "bin/gawk",

	// env — the most portable shebang helper
	"/usr/bin/env":         "bin/env",
	"/bin/env":             "bin/env",
}

// envDelegates maps the argument that follows "env" in an env-style shebang
// (e.g. "#!/usr/bin/env python3") to the Termux-relative path of that binary.
var envDelegates = map[string]string{
	"bash":      "bin/bash",
	"sh":        "bin/sh",
	"zsh":       "bin/zsh",
	"fish":      "bin/fish",
	"dash":      "bin/dash",
	"ksh":       "bin/ksh",
	"python":    "bin/python",
	"python3":   "bin/python3",
	"python2":   "bin/python2",
	"perl":      "bin/perl",
	"ruby":      "bin/ruby",
	"node":      "bin/node",
	"nodejs":    "bin/node",
	"awk":       "bin/awk",
	"gawk":      "bin/gawk",
}

// ReadShebang reads the first line of the file at path and returns it if it
// starts with "#!".  Returns an empty string if the file is not a script or
// cannot be read.
func ReadShebang(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return "", nil
	}
	line := scanner.Text()
	if !strings.HasPrefix(line, "#!") {
		return "", nil
	}
	return line, nil
}

// ParseShebang splits a shebang line into the interpreter path and any
// trailing arguments.  The "#!" prefix must already be present.
// Returns empty strings if the line is not a valid shebang.
func ParseShebang(line string) (interpreterPath string, args []string) {
	if !strings.HasPrefix(line, "#!") {
		return "", nil
	}
	rest := strings.TrimPrefix(line, "#!")
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", nil
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], fields[1:]
}

// CheckInterpreter validates the shebang interpreter against a Termux PREFIX
// directory and an optional ACL runtime directory.
//
// prefixDir is the Termux $PREFIX (e.g. /data/data/com.termux/files/usr).
// runtimeDir is an optional ACL runtime directory (may be empty).
//
// Resolution order:
//  1. If the interpreter path exists literally on the filesystem, status = found.
//  2. If the path is under an absolute Linux root (/usr/bin, /bin, etc.) and a
//     Termux-relative equivalent is known and present under prefixDir,
//     status = remapped.
//  3. If the shebang uses "/usr/bin/env <cmd>", resolve <cmd> via envDelegates
//     and check for it under prefixDir; status = remapped when found.
//  4. Otherwise status = missing.
func CheckInterpreter(interpreterPath string, args []string, prefixDir, runtimeDir string) ShebangResult {
	result := ShebangResult{
		InterpreterPath: interpreterPath,
		InterpreterArgs: args,
	}

	isEnv := isEnvInterpreter(interpreterPath)

	// --- Step 1: literal path exists on the filesystem ---
	if fileExists(interpreterPath) {
		result.Status = InterpreterFound
		result.ResolvedPath = interpreterPath
		if isEnv {
			result.Recommendation = fmt.Sprintf(
				"Interpreter %q (env-style) is present at its declared path; no change needed.",
				interpreterPath,
			)
		} else {
			result.Recommendation = fmt.Sprintf(
				"Interpreter %q exists at its declared path; no change needed.",
				interpreterPath,
			)
		}
		return result
	}

	// --- Step 2: check runtimeDir ---
	if runtimeDir != "" {
		base := filepath.Base(interpreterPath)
		runtimeCandidate := filepath.Join(runtimeDir, "bin", base)
		if fileExists(runtimeCandidate) {
			result.Status = InterpreterRemapped
			result.ResolvedPath = runtimeCandidate
			result.Recommendation = fmt.Sprintf(
				"Interpreter %q not found at declared path; ACL runtime provides it at %q.",
				interpreterPath, runtimeCandidate,
			)
			return result
		}
	}

	// --- Step 3: env-style shebang delegation ---
	if isEnv && len(args) > 0 {
		delegateName := args[0]
		if relPath, ok := envDelegates[delegateName]; ok && prefixDir != "" {
			candidate := filepath.Join(prefixDir, relPath)
			if fileExists(candidate) {
				result.Status = InterpreterRemapped
				result.ResolvedPath = candidate
				result.Recommendation = fmt.Sprintf(
					"Env-style shebang %q delegates to %q; found Termux equivalent at %q.",
					interpreterPath, delegateName, candidate,
				)
				return result
			}
			// Known mapping but not installed — report missing with hint.
			result.Status = InterpreterMissing
			result.ResolvedPath = ""
			result.Recommendation = fmt.Sprintf(
				"Env-style shebang %q delegates to %q; expected Termux path %q does not exist. "+
					"Install the package that provides %q under Termux PREFIX.",
				interpreterPath, delegateName, candidate, delegateName,
			)
			return result
		}
		// Unknown delegate name — try a direct lookup under PREFIX/bin.
		if prefixDir != "" {
			candidate := filepath.Join(prefixDir, "bin", delegateName)
			if fileExists(candidate) {
				result.Status = InterpreterRemapped
				result.ResolvedPath = candidate
				result.Recommendation = fmt.Sprintf(
					"Env-style shebang %q delegates to %q; found at Termux path %q.",
					interpreterPath, delegateName, candidate,
				)
				return result
			}
		}
	}

	// --- Step 4: well-known static remap table ---
	if relPath, ok := wellKnownTermuxPaths[interpreterPath]; ok && prefixDir != "" {
		candidate := filepath.Join(prefixDir, relPath)
		if fileExists(candidate) {
			result.Status = InterpreterRemapped
			result.ResolvedPath = candidate
			result.Recommendation = fmt.Sprintf(
				"Interpreter %q is a standard Linux path; Termux equivalent found at %q. "+
					"Update the shebang or use a wrapper.",
				interpreterPath, candidate,
			)
			return result
		}
		// Known mapping, package not installed.
		result.Status = InterpreterMissing
		result.ResolvedPath = ""
		result.Recommendation = fmt.Sprintf(
			"Interpreter %q maps to Termux path %q which does not exist. "+
				"Install the relevant Termux package.",
			interpreterPath, candidate,
		)
		return result
	}

	// --- Fallback: try to find the basename under PREFIX/bin ---
	if prefixDir != "" {
		base := filepath.Base(interpreterPath)
		candidate := filepath.Join(prefixDir, "bin", base)
		if fileExists(candidate) {
			result.Status = InterpreterRemapped
			result.ResolvedPath = candidate
			result.Recommendation = fmt.Sprintf(
				"Interpreter %q not found at declared path; located by basename at Termux path %q.",
				interpreterPath, candidate,
			)
			return result
		}
	}

	// Nothing found.
	result.Status = InterpreterMissing
	result.ResolvedPath = ""
	result.Recommendation = fmt.Sprintf(
		"Interpreter %q was not found at its declared path or under the Termux PREFIX. "+
			"Ensure the interpreter is installed and accessible.",
		interpreterPath,
	)
	return result
}

// ScanShebang is the top-level helper that reads a file, parses its shebang,
// and runs CheckInterpreter.  It returns nil when the file has no shebang.
func ScanShebang(filePath, prefixDir, runtimeDir string) (*ShebangResult, error) {
	line, err := ReadShebang(filePath)
	if err != nil {
		return nil, err
	}
	if line == "" {
		return nil, nil
	}

	interpPath, args := ParseShebang(line)
	if interpPath == "" {
		return nil, nil
	}

	res := CheckInterpreter(interpPath, args, prefixDir, runtimeDir)
	res.Raw = line
	return &res, nil
}

// isEnvInterpreter returns true when the interpreter path refers to /usr/bin/env
// or /bin/env (the portable delegation helper).
func isEnvInterpreter(path string) bool {
	base := filepath.Base(path)
	return base == "env"
}

// fileExists returns true if path exists and is accessible.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
