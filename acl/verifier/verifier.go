// Package verifier implements pre-flight runtime assumption checks for the
// Android Compatibility Layer (ACL). It validates the execution environment
// before any tool launch, detecting configuration problems early and surfacing
// actionable remediation hints.
//
// Each check returns a [Result] that carries:
//   - the check name
//   - pass/fail status
//   - an exit code category (see [ExitCode])
//   - a human-readable remediation hint when the check fails
//
// Callers iterate [RunAll] or pick individual checks via [Run].
package verifier

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ExitCode categorises the severity of a failed check.
type ExitCode int

const (
	// ExitOK indicates every check passed.
	ExitOK ExitCode = 0
	// ExitMissingDep indicates a required tool or package is absent.
	ExitMissingDep ExitCode = 2
	// ExitSELinux indicates an SELinux denial or enforcing-mode incompatibility.
	ExitSELinux ExitCode = 3
	// ExitFilesystem indicates a filesystem or permission problem.
	ExitFilesystem ExitCode = 4
	// ExitWX indicates a W^X (write-xor-execute) restriction violation.
	ExitWX ExitCode = 5
	// ExitProcFS indicates /proc is inaccessible or restricted.
	ExitProcFS ExitCode = 6
	// ExitUnknown indicates an unexpected internal error during the check.
	ExitUnknown ExitCode = 99
)

// Result holds the outcome of a single pre-flight check.
type Result struct {
	// Name is the short, machine-readable identifier for this check.
	Name string
	// Passed is true when the check succeeded.
	Passed bool
	// Code is the exit-code category; ExitOK when Passed is true.
	Code ExitCode
	// Message is a one-line description of what was checked.
	Message string
	// Hint contains actionable remediation text; empty when Passed is true.
	Hint string
}

// String returns a human-readable representation of the result.
func (r Result) String() string {
	status := "PASS"
	if !r.Passed {
		status = "FAIL"
	}
	s := fmt.Sprintf("[%s] %s: %s", status, r.Name, r.Message)
	if !r.Passed && r.Hint != "" {
		s += "\n       hint: " + r.Hint
	}
	return s
}

// Check is a named pre-flight validation function.
type Check struct {
	// Name is the short identifier used in Result.Name.
	Name string
	// Description is shown in --list output.
	Description string
	// Run executes the check and returns a Result.
	Run func() Result
}

// All is the ordered list of checks executed by RunAll.
var All = []Check{
	{
		Name:        "prefix-accessible",
		Description: "Verify Termux PREFIX directory exists and is accessible",
		Run:         CheckPrefixAccessible,
	},
	{
		Name:        "selinux-mode",
		Description: "Detect SELinux enforcing mode and known denial patterns",
		Run:         CheckSELinux,
	},
	{
		Name:        "proc-self-exe",
		Description: "Verify /proc/self/exe is readable (needed for self-path detection)",
		Run:         CheckProcSelfExe,
	},
	{
		Name:        "wx-restriction",
		Description: "Check W^X restrictions on Termux bin and lib directories",
		Run:         CheckWX,
	},
	{
		Name:        "patchelf-present",
		Description: "Verify patchelf is installed and executable",
		Run:         CheckPatchelf,
	},
	{
		Name:        "linker-present",
		Description: "Verify the Termux dynamic linker is present",
		Run:         CheckLinker,
	},
}

// RunAll executes every check in All and returns all results.
func RunAll() []Result {
	results := make([]Result, 0, len(All))
	for _, c := range All {
		results = append(results, c.Run())
	}
	return results
}

// RunSelected executes only the checks whose names are in the provided set.
// Unknown names are silently ignored; callers should validate names beforehand.
func RunSelected(names []string) []Result {
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}
	var results []Result
	for _, c := range All {
		if want[c.Name] {
			results = append(results, c.Run())
		}
	}
	return results
}

// OverallExitCode returns the most severe ExitCode seen across results.
// Returns ExitOK when all checks passed.
func OverallExitCode(results []Result) ExitCode {
	code := ExitOK
	for _, r := range results {
		if !r.Passed && r.Code > code {
			code = r.Code
		}
	}
	return code
}

// ─── individual checks ────────────────────────────────────────────────────────

// CheckPrefixAccessible verifies that the Termux PREFIX directory can be found
// and that the expected sub-directories (bin, lib) are accessible.
func CheckPrefixAccessible() Result {
	name := "prefix-accessible"
	prefix := termuxPrefix()
	if prefix == "" {
		return Result{
			Name:    name,
			Passed:  false,
			Code:    ExitFilesystem,
			Message: "Termux PREFIX not set and default path not found",
			Hint: "Set the TERMUX_PREFIX (or PREFIX) environment variable to your Termux " +
				"installation root, e.g. /data/data/com.termux/files/usr",
		}
	}
	for _, sub := range []string{"bin", "lib"} {
		p := filepath.Join(prefix, sub)
		if _, err := os.Stat(p); err != nil {
			return Result{
				Name:    name,
				Passed:  false,
				Code:    ExitFilesystem,
				Message: fmt.Sprintf("PREFIX sub-directory not accessible: %s (%v)", p, err),
				Hint: fmt.Sprintf(
					"Ensure Termux is properly installed. Expected directory: %s", p),
			}
		}
	}
	return Result{
		Name:    name,
		Passed:  true,
		Code:    ExitOK,
		Message: fmt.Sprintf("PREFIX accessible: %s", prefix),
	}
}

// CheckSELinux inspects /sys/fs/selinux/enforce and /proc/self/attr/current to
// detect whether SELinux is in enforcing mode and whether the process context
// is likely to trigger known denials.
func CheckSELinux() Result {
	name := "selinux-mode"

	// Non-Android platforms: skip gracefully.
	if !isAndroidLike() {
		return Result{
			Name:    name,
			Passed:  true,
			Code:    ExitOK,
			Message: "SELinux check skipped on non-Android platform",
		}
	}

	enforceFile := "/sys/fs/selinux/enforce"
	data, err := os.ReadFile(enforceFile)
	if err != nil {
		// SELinux may not be mounted at all (permissive kernels, emulators).
		return Result{
			Name:    name,
			Passed:  true,
			Code:    ExitOK,
			Message: "SELinux enforce file not readable; assuming permissive or disabled",
		}
	}

	enforcing := strings.TrimSpace(string(data)) == "1"
	if !enforcing {
		return Result{
			Name:    name,
			Passed:  true,
			Code:    ExitOK,
			Message: "SELinux is in permissive mode",
		}
	}

	// Enforcing: inspect our own process context.
	context, ctxErr := readSELinuxContext()
	if ctxErr == nil && isKnownDangerousContext(context) {
		return Result{
			Name:   name,
			Passed: false,
			Code:   ExitSELinux,
			Message: fmt.Sprintf(
				"SELinux is enforcing and process context may cause denials: %s", context),
			Hint: "Run inside a Termux session (context: u:r:untrusted_app:*). " +
				"ADB shell or root shells often run under stricter contexts that " +
				"deny mmap/execute on Termux paths. " +
				"If you are in Termux and still see this, check for custom SELinux policies.",
		}
	}

	// Enforcing but context looks OK (or unreadable – we give benefit of doubt).
	msg := "SELinux is enforcing"
	if ctxErr == nil {
		msg = fmt.Sprintf("SELinux is enforcing (context: %s)", context)
	}
	return Result{
		Name:    name,
		Passed:  true,
		Code:    ExitOK,
		Message: msg,
	}
}

// CheckProcSelfExe verifies that /proc/self/exe is readable and points to a
// regular file.
func CheckProcSelfExe() Result {
	name := "proc-self-exe"

	if !isAndroidLike() {
		// On Linux/macOS development machines we still validate.
	}

	target, err := os.Readlink("/proc/self/exe")
	if err != nil {
		return Result{
			Name:   name,
			Passed: false,
			Code:   ExitProcFS,
			Message: fmt.Sprintf(
				"/proc/self/exe not readable: %v", err),
			Hint: "Some restrictive Android policies deny access to /proc/self/exe. " +
				"Ensure your kernel has CONFIG_PROC_SELF_MAPS enabled and that your " +
				"SELinux policy permits procattr reads for Termux app contexts.",
		}
	}

	if _, statErr := os.Stat(target); statErr != nil {
		return Result{
			Name:   name,
			Passed: false,
			Code:   ExitProcFS,
			Message: fmt.Sprintf(
				"/proc/self/exe -> %s is not accessible: %v", target, statErr),
			Hint: "The binary pointed to by /proc/self/exe cannot be stat'd. " +
				"This may indicate a deleted executable or a bind-mount issue.",
		}
	}

	return Result{
		Name:    name,
		Passed:  true,
		Code:    ExitOK,
		Message: fmt.Sprintf("/proc/self/exe -> %s", target),
	}
}

// CheckWX verifies that the Termux bin and lib directories do not have
// simultaneous write+execute permission for the current process, which would
// violate Android's W^X policy and prevent execution.
func CheckWX() Result {
	name := "wx-restriction"
	prefix := termuxPrefix()
	if prefix == "" {
		return Result{
			Name:    name,
			Passed:  true,
			Code:    ExitOK,
			Message: "PREFIX not found; W^X check skipped",
		}
	}

	dirs := []string{
		filepath.Join(prefix, "bin"),
		filepath.Join(prefix, "lib"),
	}

	var problems []string
	for _, d := range dirs {
		info, err := os.Stat(d)
		if err != nil {
			continue // already reported by prefix-accessible
		}
		mode := info.Mode().Perm()
		// Check if the directory itself has both write and execute bits for
		// world (other), which is unusual and may trigger denials.
		worldWX := fs.FileMode(0o003) // o+wx
		if mode&worldWX == worldWX {
			problems = append(problems,
				fmt.Sprintf("%s has world-writable+executable bits (%s)", d, mode))
		}
	}

	if len(problems) > 0 {
		return Result{
			Name:   name,
			Passed: false,
			Code:   ExitWX,
			Message: fmt.Sprintf(
				"W^X concern in Termux directories: %s", strings.Join(problems, "; ")),
			Hint: "Android enforces W^X: a memory region cannot be both writable and " +
				"executable simultaneously. Remove world-write permission from Termux " +
				"directories: chmod o-w " + strings.Join(dirs, " "),
		}
	}

	return Result{
		Name:    name,
		Passed:  true,
		Code:    ExitOK,
		Message: fmt.Sprintf("No W^X violations detected in %s", strings.Join(dirs, ", ")),
	}
}

// CheckPatchelf verifies that patchelf is available on PATH or under the
// Termux prefix and is actually executable.
func CheckPatchelf() Result {
	name := "patchelf-present"

	path, err := findTool("patchelf")
	if err != nil {
		hint := "Install patchelf in Termux: pkg install patchelf\n" +
			"       patchelf is required for ACL apply-mode ELF patching."
		return Result{
			Name:    name,
			Passed:  false,
			Code:    ExitMissingDep,
			Message: "patchelf not found on PATH or under Termux PREFIX",
			Hint:    hint,
		}
	}

	// Verify it actually runs.
	out, runErr := exec.Command(path, "--version").CombinedOutput()
	if runErr != nil {
		return Result{
			Name:   name,
			Passed: false,
			Code:   ExitMissingDep,
			Message: fmt.Sprintf(
				"patchelf found at %s but failed to execute: %v", path, runErr),
			Hint: "The patchelf binary may be corrupt or built for a different ABI. " +
				"Reinstall: pkg reinstall patchelf",
		}
	}

	version := strings.TrimSpace(string(out))
	if len(version) > 60 {
		version = version[:60] + "…"
	}
	return Result{
		Name:    name,
		Passed:  true,
		Code:    ExitOK,
		Message: fmt.Sprintf("patchelf found: %s (%s)", path, version),
	}
}

// CheckLinker verifies that the Termux dynamic linker (ld-linux or ld-android)
// is present under the Termux prefix.
func CheckLinker() Result {
	name := "linker-present"
	prefix := termuxPrefix()
	if prefix == "" {
		return Result{
			Name:   name,
			Passed: false,
			Code:   ExitMissingDep,
			Message: "Cannot check linker: Termux PREFIX not found",
			Hint: "Set TERMUX_PREFIX to your Termux installation root and ensure " +
				"Termux is installed.",
		}
	}

	// Candidate linker paths in priority order.
	candidates := linkerCandidates(prefix)
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return Result{
				Name:    name,
				Passed:  true,
				Code:    ExitOK,
				Message: fmt.Sprintf("Linker found: %s", p),
			}
		}
	}

	return Result{
		Name:   name,
		Passed: false,
		Code:   ExitMissingDep,
		Message: fmt.Sprintf(
			"No dynamic linker found under PREFIX (%s); checked: %s",
			prefix, strings.Join(candidates, ", ")),
		Hint: "Install the glibc linker from the Termux glibc-repo:\n" +
			"       pkg install glibc-repo\n" +
			"       pkg install glibc\n" +
			"       Or install the Bionic linker via: pkg install binutils",
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// termuxPrefix returns the Termux PREFIX path, checking environment variables
// and the default installation path.
func termuxPrefix() string {
	for _, key := range []string{"TERMUX_PREFIX", "TERMUX__PREFIX", "PREFIX"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			if _, err := os.Stat(v); err == nil {
				return v
			}
		}
	}
	// Default Termux installation path.
	def := "/data/data/com.termux/files/usr"
	if _, err := os.Stat(def); err == nil {
		return def
	}
	return ""
}

// isAndroidLike returns true when the process appears to be running on Android
// or a Linux system where Android-specific checks make sense.
func isAndroidLike() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	// Heuristic: check for Android-specific paths.
	for _, p := range []string{
		"/system/build.prop",
		"/data/data/com.termux",
		"/proc/sys/kernel/android",
	} {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return true // assume Linux == potential Android for CI compatibility
}

// readSELinuxContext reads the current process's SELinux context from
// /proc/self/attr/current.
func readSELinuxContext() (string, error) {
	data, err := os.ReadFile("/proc/self/attr/current")
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(data), "\x00\n"), nil
}

// isKnownDangerousContext returns true for SELinux contexts that are known to
// trigger exec/mmap denials on Termux paths.
func isKnownDangerousContext(ctx string) bool {
	dangerous := []string{
		"u:r:shell:",
		"u:r:su:",
		"u:r:init:",
		"u:r:kernel:",
		"u:r:recovery:",
		"u:r:system_server:",
	}
	for _, d := range dangerous {
		if strings.Contains(ctx, d) {
			return true
		}
	}
	return false
}

// findTool searches for a tool by name on PATH and, if not found, under the
// Termux PREFIX bin directory.
func findTool(name string) (string, error) {
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	prefix := termuxPrefix()
	if prefix != "" {
		candidate := filepath.Join(prefix, "bin", name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			if info.Mode()&0o111 != 0 {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("%s not found", name)
}

// linkerCandidates returns candidate dynamic linker paths for the given prefix,
// ordered from most to least preferred.
func linkerCandidates(prefix string) []string {
	arch := runtime.GOARCH
	var linkers []string

	switch arch {
	case "arm64":
		linkers = []string{
			// glibc-repo linker
			filepath.Join(prefix, "glibc", "lib", "ld-linux-aarch64.so.1"),
			// Termux Bionic (older setups)
			filepath.Join(prefix, "lib", "ld-linux-aarch64.so.1"),
			// Fallback system linker
			"/system/bin/linker64",
		}
	case "amd64":
		linkers = []string{
			filepath.Join(prefix, "glibc", "lib", "ld-linux-x86-64.so.2"),
			filepath.Join(prefix, "lib", "ld-linux-x86-64.so.2"),
			"/system/bin/linker64",
		}
	case "arm":
		linkers = []string{
			filepath.Join(prefix, "glibc", "lib", "ld-linux-armhf.so.3"),
			filepath.Join(prefix, "lib", "ld-linux-armhf.so.3"),
			"/system/bin/linker",
		}
	default:
		linkers = []string{
			filepath.Join(prefix, "glibc", "lib", "ld.so.1"),
			filepath.Join(prefix, "lib", "ld.so.1"),
		}
	}
	return linkers
}

// ParseSELinuxDenials scans r line by line and returns a slice of lines that
// look like SELinux AVC denial messages. This is exported for use in test
// helpers and diagnostic tools.
func ParseSELinuxDenials(r *bufio.Scanner) []string {
	var denials []string
	for r.Scan() {
		line := r.Text()
		if strings.Contains(line, "avc:") && strings.Contains(line, "denied") {
			denials = append(denials, line)
		}
	}
	return denials
}
