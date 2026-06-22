package exec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	aclscan "github.com/arduino/arduino-cli/internal/acl/elfscan"
	aclruntime "github.com/arduino/arduino-cli/internal/acl/runtime"
)

type DiagnosticReport struct {
	Target      TargetDiagnostics      `json:"target"`
	Execution   ExecutionDiagnostics   `json:"execution"`
	Environment EnvironmentDiagnostics `json:"environment"`
	Runtime     RuntimeDiagnostics     `json:"runtime"`
	TargetData  TargetDataDiagnostics  `json:"target_data"`
	Result      ResultDiagnostics      `json:"result"`
	Hints       []string               `json:"hints,omitempty"`
}

type TargetDiagnostics struct {
	Path                 string `json:"path"`
	Basename             string `json:"basename"`
	Exists               bool   `json:"exists"`
	ExecutablePermission bool   `json:"executable_permission"`
	FileMode             string `json:"file_mode"`
	TargetClass          string `json:"target_class"`
}

type ExecutionDiagnostics struct {
	PlannerStrategy       string `json:"planner_strategy"`
	DirectExecution       bool   `json:"direct_execution_selected"`
	ExplicitLoader        bool   `json:"explicit_loader_selected"`
	LoaderPath            string `json:"loader_path,omitempty"`
	DirectExecDescription string `json:"direct_exec_description,omitempty"`
}

type EnvironmentDiagnostics struct {
	RuntimeRoot       string   `json:"runtime_root,omitempty"`
	RuntimeRootSource string   `json:"runtime_root_source,omitempty"`
	CWD               string   `json:"cwd,omitempty"`
	State             string   `json:"state"`
	Description       string   `json:"description"`
	Kept              []string `json:"kept,omitempty"`
	Removed           []string `json:"removed,omitempty"`
	Indicators        []string `json:"indicators,omitempty"`
	SanitizedSummary  string   `json:"sanitized_summary"`
}

type RuntimeDiagnostics struct {
	PTInterp string   `json:"pt_interp,omitempty"`
	DTNeeded []string `json:"dt_needed,omitempty"`
	Argv     []string `json:"argv,omitempty"`
}

type TargetDataDiagnostics struct {
	Machine         string                           `json:"machine,omitempty"`
	IsELF           bool                             `json:"is_elf"`
	HasPTInterp     bool                             `json:"has_pt_interp"`
	LikelySource    string                           `json:"likely_source,omitempty"`
	DelegateTargets []aclscan.LauncherDelegateTarget `json:"delegate_targets,omitempty"`
}

type ResultDiagnostics struct {
	Mode           string `json:"mode"`
	Stdout         string `json:"stdout,omitempty"`
	Stderr         string `json:"stderr,omitempty"`
	ExitCode       *int   `json:"exit_code,omitempty"`
	Errno          string `json:"errno,omitempty"`
	ChildExecErrno string `json:"child_exec_errno,omitempty"`
	StartError     string `json:"start_error,omitempty"`
	Started        bool   `json:"started"`
}

func (r DiagnosticReport) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func FormatDiagnosticReport(r DiagnosticReport) string {
	var b strings.Builder
	fmt.Fprintln(&b, "ACL Execution Diagnostics")
	fmt.Fprintln(&b, "-------------------------")
	fmt.Fprintf(&b, "Target path: %s\n", r.Target.Path)
	fmt.Fprintf(&b, "Target basename: %s\n", r.Target.Basename)
	fmt.Fprintf(&b, "Target exists: %t\n", r.Target.Exists)
	fmt.Fprintf(&b, "Executable permission: %t\n", r.Target.ExecutablePermission)
	fmt.Fprintf(&b, "File mode: %s\n", r.Target.FileMode)
	fmt.Fprintf(&b, "Target classification: %s\n", r.Target.TargetClass)
	fmt.Fprintf(&b, "Planner strategy: %s\n", r.Execution.PlannerStrategy)
	fmt.Fprintf(&b, "Direct execution selected: %t\n", r.Execution.DirectExecution)
	fmt.Fprintf(&b, "Explicit loader selected: %t\n", r.Execution.ExplicitLoader)
	if r.Execution.LoaderPath != "" {
		fmt.Fprintf(&b, "Loader path: %s\n", r.Execution.LoaderPath)
	}
	if r.Environment.RuntimeRoot != "" {
		fmt.Fprintf(&b, "Runtime root: %s\n", r.Environment.RuntimeRoot)
	}
	if r.Environment.RuntimeRootSource != "" {
		fmt.Fprintf(&b, "Runtime root source: %s\n", r.Environment.RuntimeRootSource)
	}
	if r.Environment.CWD != "" {
		fmt.Fprintf(&b, "CWD: %s\n", r.Environment.CWD)
	}
	if r.Environment.Description != "" {
		fmt.Fprintf(&b, "Environment state: %s\n", r.Environment.Description)
	}
	if r.Runtime.PTInterp != "" {
		fmt.Fprintf(&b, "PT_INTERP: %s\n", r.Runtime.PTInterp)
	} else {
		fmt.Fprintln(&b, "PT_INTERP: <none>")
	}
	if len(r.Runtime.DTNeeded) > 0 {
		fmt.Fprintln(&b, "DT_NEEDED:")
		for _, lib := range r.Runtime.DTNeeded {
			fmt.Fprintf(&b, "  - %s\n", lib)
		}
	} else {
		fmt.Fprintln(&b, "DT_NEEDED: <none>")
	}
	if len(r.Runtime.Argv) > 0 {
		fmt.Fprintf(&b, "Argv: %s\n", strings.Join(r.Runtime.Argv, " "))
	}
	if len(r.TargetData.DelegateTargets) > 0 {
		fmt.Fprintln(&b, "Delegate targets:")
		for _, target := range r.TargetData.DelegateTargets {
			fmt.Fprintf(&b, "  - %s", target.Path)
			if target.Symlink {
				fmt.Fprintf(&b, " symlink->%s", target.SymlinkTarget)
			}
			fmt.Fprintf(&b, " exists=%t executable=%t", target.Exists, target.Executable)
			if target.Mode != "" {
				fmt.Fprintf(&b, " mode=%s", target.Mode)
			}
			if target.Source != "" {
				fmt.Fprintf(&b, " source=%s", target.Source)
			}
			fmt.Fprintln(&b)
		}
	}
	fmt.Fprintf(&b, "Sanitized environment summary: %s\n", r.Environment.SanitizedSummary)
	if len(r.Environment.Kept) > 0 {
		fmt.Fprintln(&b, "Kept environment variables:")
		for _, kv := range r.Environment.Kept {
			fmt.Fprintf(&b, "  - %s\n", kv)
		}
	}
	if len(r.Environment.Removed) > 0 {
		fmt.Fprintln(&b, "Removed environment variables:")
		for _, kv := range r.Environment.Removed {
			fmt.Fprintf(&b, "  - %s\n", kv)
		}
	}
	if len(r.Environment.Indicators) > 0 {
		fmt.Fprintln(&b, "Environment evidence:")
		for _, indicator := range r.Environment.Indicators {
			fmt.Fprintf(&b, "  - %s\n", indicator)
		}
	}
	if r.Result.Mode != "" {
		fmt.Fprintf(&b, "Exec mode: %s\n", r.Result.Mode)
	}
	if r.Result.Started {
		fmt.Fprintln(&b, "Exec result: started")
	} else if r.Result.StartError != "" {
		fmt.Fprintf(&b, "Exec result: start error: %s\n", r.Result.StartError)
	} else {
		fmt.Fprintln(&b, "Exec result: dry-run")
	}
	if r.Result.Errno != "" {
		fmt.Fprintf(&b, "Errno: %s\n", r.Result.Errno)
	}
	if r.Result.ChildExecErrno != "" {
		fmt.Fprintf(&b, "Child exec errno: %s\n", r.Result.ChildExecErrno)
	}
	if r.Result.ExitCode != nil {
		fmt.Fprintf(&b, "Exit code: %d\n", *r.Result.ExitCode)
	}
	if r.Result.Stdout != "" {
		fmt.Fprintf(&b, "Stdout: %s\n", r.Result.Stdout)
	}
	if r.Result.Stderr != "" {
		fmt.Fprintf(&b, "Stderr: %s\n", r.Result.Stderr)
	}
	if len(r.Hints) > 0 {
		fmt.Fprintln(&b, "Likely cause hints:")
		for _, hint := range r.Hints {
			fmt.Fprintf(&b, "  - %s\n", hint)
		}
	}
	return b.String()
}

func BuildDiagnosticReport(plan ExecutionPlan, req Request, result Result) DiagnosticReport {
	targetPath := strings.TrimSpace(plan.TargetPath)
	targetBase := filepath.Base(targetPath)

	info, statErr := os.Lstat(targetPath)
	exists := statErr == nil
	fileMode := ""
	execPerm := false
	if exists {
		fileMode = info.Mode().String()
		execPerm = info.Mode()&0o111 != 0
	} else if targetPath != "" {
		fileMode = "missing"
	}

	_, audit := sanitizeExecutionEnvWithAudit(os.Environ(), plan.Environment)
	envDetection := executionEnvironmentDetector()
	runtimeRoot, runtimeRootSource := resolveRuntimeRootForReport(plan)

	var exitCodePtr *int
	if req.Apply {
		exitCode := result.ExitCode
		exitCodePtr = &exitCode
	}

	d := DiagnosticReport{
		Target: TargetDiagnostics{
			Path:                 targetPath,
			Basename:             targetBase,
			Exists:               exists,
			ExecutablePermission: execPerm,
			FileMode:             fileMode,
			TargetClass:          plan.TargetClass,
		},
		Execution: ExecutionDiagnostics{
			PlannerStrategy:       plan.LaunchMode,
			DirectExecution:       plan.LaunchMode == LaunchModeDirectExec,
			ExplicitLoader:        plan.LaunchMode == LaunchModeExplicitLoader,
			LoaderPath:            plan.LoaderPath,
			DirectExecDescription: directExecutionDescription(plan.Target),
		},
		Environment: EnvironmentDiagnostics{
			RuntimeRoot:       runtimeRoot,
			RuntimeRootSource: runtimeRootSource,
			CWD:               plan.Cwd,
			State:             string(envDetection.State),
			Description:       envDetection.Description,
			Kept:              append([]string(nil), audit.Kept...),
			Removed:           append([]string(nil), audit.Removed...),
			Indicators:        append([]string(nil), envDetection.Evidence...),
			SanitizedSummary:  audit.SanitizedSummary,
		},
		Runtime: RuntimeDiagnostics{
			PTInterp: plan.Target.Interpreter,
			DTNeeded: append([]string(nil), plan.Target.ImportedLibraries...),
			Argv:     append([]string(nil), plan.Argv...),
		},
		TargetData: TargetDataDiagnostics{
			Machine:         plan.Target.Machine,
			IsELF:           plan.Target.IsELF,
			HasPTInterp:     strings.TrimSpace(plan.Target.Interpreter) != "",
			LikelySource:    likelyTargetSource(plan),
			DelegateTargets: append([]aclscan.LauncherDelegateTarget(nil), plan.Target.LauncherDelegateTargets...),
		},
		Result: ResultDiagnostics{
			Mode:           plan.LaunchMode,
			Stdout:         result.Stdout,
			Stderr:         result.Stderr,
			ExitCode:       exitCodePtr,
			Errno:          result.Errno,
			ChildExecErrno: detectChildExecErrno(result),
			StartError:     result.StartError,
			Started:        req.Apply && result.StartError == "",
		},
		Hints: buildLikelyCauseHints(plan, result, exists, execPerm, envDetection),
	}
	return d
}

func planRuntimeRootFromPlan(plan ExecutionPlan) string {
	for _, kv := range plan.Environment {
		if strings.HasPrefix(kv, "ACL_RUNTIME_ROOT=") {
			return strings.TrimPrefix(kv, "ACL_RUNTIME_ROOT=")
		}
	}
	return ""
}

func resolveRuntimeRootForReport(plan ExecutionPlan) (string, string) {
	if root := strings.TrimSpace(plan.RuntimeRoot); root != "" {
		return root, "configured"
	}
	if root := strings.TrimSpace(planRuntimeRootFromPlan(plan)); root != "" {
		return root, "plan-environment"
	}
	if root, err := aclruntime.DefaultRoot(); err == nil && strings.TrimSpace(root) != "" {
		return root, "detected"
	}
	return "", "unknown"
}

func splitExecutionEnv(env []string) (removed []string, kept []string) {
	for _, kv := range env {
		if isSanitizedExecutionVariable(kv) {
			removed = append(removed, kv)
			continue
		}
		kept = append(kept, kv)
	}
	return removed, kept
}

func summarizeSanitizedEnvironment(kept, removed, indicators []string) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("%d kept", len(kept)))
	parts = append(parts, fmt.Sprintf("%d removed", len(removed)))
	if len(indicators) > 0 {
		parts = append(parts, fmt.Sprintf("%d evidence items", len(indicators)))
	}
	return strings.Join(parts, ", ")
}

type EnvironmentState string

const (
	EnvironmentStateUnknown      EnvironmentState = "unknown"
	EnvironmentStateNativeTermux EnvironmentState = "native-termux"
	EnvironmentStateProot        EnvironmentState = "proot"
)

type EnvironmentDetection struct {
	State       EnvironmentState
	Description string
	Evidence    []string
}

var procSelfRootReader = currentProcSelfRoot
var executionEnvironmentDetector = detectExecutionEnvironment

func detectExecutionEnvironment() EnvironmentDetection {
	return detectExecutionEnvironmentFromEnv(os.Environ(), procSelfRootReader())
}

func detectExecutionEnvironmentFromEnv(env []string, procSelfRoot string) EnvironmentDetection {
	values := make(map[string]string, len(env))
	for _, kv := range env {
		key := kv
		value := ""
		if idx := strings.IndexByte(kv, '='); idx >= 0 {
			key = kv[:idx]
			value = kv[idx+1:]
		}
		values[key] = value
	}

	var termuxEvidence []string
	var prootEvidence []string

	addTermuxEvidence := func(label, value string) {
		if value == "" {
			termuxEvidence = append(termuxEvidence, label)
			return
		}
		termuxEvidence = append(termuxEvidence, fmt.Sprintf("%s=%s", label, value))
	}

	if v := strings.TrimSpace(values["TERMUX_VERSION"]); v != "" {
		addTermuxEvidence("TERMUX_VERSION", v)
	}
	if v := strings.TrimSpace(values["PREFIX"]); isNativeTermuxPrefix(v) {
		addTermuxEvidence("PREFIX", v)
	}
	if v := strings.TrimSpace(values["TERMUX_PREFIX"]); isNativeTermuxPrefix(v) {
		addTermuxEvidence("TERMUX_PREFIX", v)
	}
	if v := strings.TrimSpace(values["HOME"]); isNativeTermuxHome(v) {
		addTermuxEvidence("HOME", v)
	}
	if v := strings.TrimSpace(values["TERMUX_HOME"]); isNativeTermuxHome(v) {
		addTermuxEvidence("TERMUX_HOME", v)
	}
	if v := strings.TrimSpace(values["TERMUX__PREFIX"]); isNativeTermuxPrefix(v) {
		addTermuxEvidence("TERMUX__PREFIX", v)
	}
	if v := strings.TrimSpace(values["TERMUX__ROOTFS_DIR"]); v != "" && strings.Contains(v, "/data/data/com.termux/files") {
		addTermuxEvidence("TERMUX__ROOTFS_DIR", v)
	}
	if v := strings.TrimSpace(values["TERMUX_MAIN_PACKAGE_FORMAT"]); v != "" {
		addTermuxEvidence("TERMUX_MAIN_PACKAGE_FORMAT", v)
	}
	if v := strings.TrimSpace(values["TERMUX__HOME"]); isNativeTermuxHome(v) {
		addTermuxEvidence("TERMUX__HOME", v)
	}

	if hasAnyEnvPrefixInValues(values, "PROOT_") {
		prootEvidence = append(prootEvidence, "PROOT_*")
	}
	if hasAnyEnvPrefixInValues(values, "QEMU_") {
		prootEvidence = append(prootEvidence, "QEMU_*")
	}
	if strings.TrimSpace(values["CHROOT"]) != "" {
		prootEvidence = append(prootEvidence, "CHROOT")
	}
	if rootEvidence := prootRootEvidence(procSelfRoot); rootEvidence != "" {
		prootEvidence = append(prootEvidence, rootEvidence)
	}
	if rootfsEvidence := prootRootfsEnvEvidence(values); rootfsEvidence != "" {
		prootEvidence = append(prootEvidence, rootfsEvidence)
	}

	switch {
	case len(prootEvidence) > 0:
		return EnvironmentDetection{
			State:       EnvironmentStateProot,
			Description: "PRoot/proot-distro detected",
			Evidence:    dedupeStrings(append(termuxEvidence, prootEvidence...)),
		}
	case len(termuxEvidence) > 0:
		return EnvironmentDetection{
			State:       EnvironmentStateNativeTermux,
			Description: "native Termux detected",
			Evidence:    dedupeStrings(termuxEvidence),
		}
	default:
		return EnvironmentDetection{
			State:       EnvironmentStateUnknown,
			Description: "environment unknown",
		}
	}
}

func currentProcSelfRoot() string {
	root, err := os.Readlink("/proc/self/root")
	if err != nil {
		return ""
	}
	return root
}

func hasAnyEnvPrefixInValues(values map[string]string, prefix string) bool {
	for key := range values {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func isNativeTermuxPrefix(value string) bool {
	value = filepath.Clean(strings.TrimSpace(value))
	return strings.HasPrefix(value, "/data/data/com.termux/files/usr")
}

func isNativeTermuxHome(value string) bool {
	value = filepath.Clean(strings.TrimSpace(value))
	return strings.HasPrefix(value, "/data/data/com.termux/files/home")
}

func prootRootEvidence(procSelfRoot string) string {
	root := strings.TrimSpace(procSelfRoot)
	if root == "" || root == "/" {
		return ""
	}
	lower := strings.ToLower(root)
	switch {
	case strings.Contains(lower, "proot-distro"),
		strings.Contains(lower, "/installed-rootfs/"),
		strings.Contains(lower, "/rootfs/"),
		strings.Contains(lower, "ubuntu"),
		strings.Contains(lower, "debian"),
		strings.Contains(lower, "alpine"),
		strings.Contains(lower, "fedora"),
		strings.Contains(lower, "arch"):
		return "/proc/self/root=" + root
	default:
		return ""
	}
}

func prootRootfsEnvEvidence(values map[string]string) string {
	for _, key := range []string{"PROOT_ROOTFS", "TERMUX__ROOTFS_DIR"} {
		if v := strings.TrimSpace(values[key]); v != "" && (strings.Contains(strings.ToLower(v), "proot-distro") || strings.Contains(strings.ToLower(v), "rootfs")) {
			return key + "=" + v
		}
	}
	return ""
}

func directExecutionDescription(ins aclscan.Inspection) string {
	switch {
	case ins.LooksLikeRustLauncher:
		return "Rust launcher wrapper should keep its own executable identity"
	case !ins.LooksLikeLinuxTarget:
		return "Android-native or non-Linux target can run directly"
	default:
		return "direct kernel exec selected"
	}
}

func likelyTargetSource(plan ExecutionPlan) string {
	switch plan.TargetClass {
	case TargetClassRustLauncher:
		return "rust-launcher"
	case TargetClassPatchedLinux:
		return "patched-linux-elf"
	case TargetClassLinuxDirect:
		return "linux-direct-elf"
	case TargetClassAndroidNative:
		return "android-native-elf"
	default:
		return ""
	}
}

func buildLikelyCauseHints(plan ExecutionPlan, result Result, exists, execPerm bool, env EnvironmentDetection) []string {
	var hints []string
	if !exists {
		hints = append(hints, "target path does not exist")
	}
	if exists && !execPerm {
		hints = append(hints, "target is not marked executable")
	}
	switch env.State {
	case EnvironmentStateNativeTermux:
		// Native Termux is the target runtime; do not suggest container-like evidence is required.
	case EnvironmentStateProot:
		hints = append(hints, "execution appears to be inside PRoot/proot-distro or chroot-like environment")
	case EnvironmentStateUnknown:
		hints = append(hints, "environment could not be classified as native Termux or PRoot/proot-distro")
	}
	if plan.TargetClass == TargetClassRustLauncher && result.StartError != "" {
		hints = append(hints, "Rust launcher wrappers need direct kernel exec; inspect wrapper-specific /proc/self/exe behavior and target resolution")
	}
	if plan.TargetClass == TargetClassRustLauncher {
		if childErrno := detectChildExecErrno(result); childErrno != "" {
			hints = append(hints, "Rust launcher child exec returned "+childErrno+"; check delegate path existence, execute bits, symlink resolution, and noexec/SELinux constraints")
		}
		if len(plan.Target.LauncherDelegateTargets) > 0 {
			var issues []string
			for _, target := range plan.Target.LauncherDelegateTargets {
				if !target.Exists {
					issues = append(issues, target.Path+" missing")
					continue
				}
				if !target.Executable {
					issues = append(issues, target.Path+" not executable")
				}
			}
			if len(issues) > 0 {
				hints = append(hints, "Rust launcher delegate candidates: "+strings.Join(issues, "; "))
			}
		}
	}
	if plan.LaunchMode == LaunchModeExplicitLoader {
		hints = append(hints, "explicit loader path depends on a valid runtime tree and loader-visible libraries")
	}
	if result.StartError != "" && strings.Contains(strings.ToLower(result.StartError), "permission denied") {
		hints = append(hints, "permission denied usually means the binary lacks execute permission or the filesystem forbids execution")
	}
	if result.StartError != "" && strings.Contains(strings.ToLower(result.StartError), "no such file") {
		hints = append(hints, "missing file or loader path is the most likely cause")
	}
	if result.ExitCode != 0 && result.StartError == "" && plan.TargetClass == TargetClassRustLauncher {
		hints = append(hints, "nonzero exit on a direct rust-launcher run is evidence about the tool itself, not the loader path")
	}
	return dedupeStrings(hints)
}

func detectChildExecErrno(result Result) string {
	combined := strings.ToLower(result.Stdout + "\n" + result.Stderr + "\n" + result.StartError)
	switch {
	case strings.Contains(combined, "execv errno (13)"),
		strings.Contains(combined, "errno 13"),
		strings.Contains(combined, "eacces"),
		strings.Contains(combined, "permission denied"):
		return "EACCES"
	case strings.Contains(combined, "execv errno (2)"),
		strings.Contains(combined, "enoent"),
		strings.Contains(combined, "no such file"):
		return "ENOENT"
	default:
		return ""
	}
}

func dedupeStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	sort.Strings(in)
	out := in[:0]
	var prev string
	for i, s := range in {
		if i == 0 || s != prev {
			out = append(out, s)
			prev = s
		}
	}
	return out
}
