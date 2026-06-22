package exec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	aclscan "github.com/arduino/arduino-cli/internal/acl/elfscan"
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
	RuntimeRoot      string   `json:"runtime_root,omitempty"`
	CWD              string   `json:"cwd,omitempty"`
	Kept             []string `json:"kept,omitempty"`
	Removed          []string `json:"removed,omitempty"`
	Indicators       []string `json:"indicators,omitempty"`
	SanitizedSummary string   `json:"sanitized_summary"`
}

type RuntimeDiagnostics struct {
	PTInterp string   `json:"pt_interp,omitempty"`
	DTNeeded []string `json:"dt_needed,omitempty"`
	Argv     []string `json:"argv,omitempty"`
}

type TargetDataDiagnostics struct {
	Machine      string `json:"machine,omitempty"`
	IsELF        bool   `json:"is_elf"`
	HasPTInterp  bool   `json:"has_pt_interp"`
	LikelySource string `json:"likely_source,omitempty"`
}

type ResultDiagnostics struct {
	Mode       string `json:"mode"`
	Stdout     string `json:"stdout,omitempty"`
	Stderr     string `json:"stderr,omitempty"`
	ExitCode   *int   `json:"exit_code,omitempty"`
	Errno      string `json:"errno,omitempty"`
	StartError string `json:"start_error,omitempty"`
	Started    bool   `json:"started"`
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
	if r.Environment.CWD != "" {
		fmt.Fprintf(&b, "CWD: %s\n", r.Environment.CWD)
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
		fmt.Fprintln(&b, "Container indicators:")
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
	runtimeRoot := plan.RuntimeRoot
	if runtimeRoot == "" {
		runtimeRoot = planRuntimeRootFromPlan(plan)
	}

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
			RuntimeRoot:      runtimeRoot,
			CWD:              plan.Cwd,
			Kept:             append([]string(nil), audit.Kept...),
			Removed:          append([]string(nil), audit.Removed...),
			Indicators:       append([]string(nil), audit.Indicators...),
			SanitizedSummary: audit.SanitizedSummary,
		},
		Runtime: RuntimeDiagnostics{
			PTInterp: plan.Target.Interpreter,
			DTNeeded: append([]string(nil), plan.Target.ImportedLibraries...),
			Argv:     append([]string(nil), plan.Argv...),
		},
		TargetData: TargetDataDiagnostics{
			Machine:      plan.Target.Machine,
			IsELF:        plan.Target.IsELF,
			HasPTInterp:  strings.TrimSpace(plan.Target.Interpreter) != "",
			LikelySource: likelyTargetSource(plan),
		},
		Result: ResultDiagnostics{
			Mode:       plan.LaunchMode,
			Stdout:     result.Stdout,
			Stderr:     result.Stderr,
			ExitCode:   exitCodePtr,
			Errno:      result.Errno,
			StartError: result.StartError,
			Started:    req.Apply && result.StartError == "",
		},
		Hints: buildLikelyCauseHints(plan, result, exists, execPerm, audit.Indicators),
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
		parts = append(parts, fmt.Sprintf("%d indicators", len(indicators)))
	}
	return strings.Join(parts, ", ")
}

func detectExecutionIndicators() []string {
	var indicators []string
	checks := []struct {
		name string
		ok   bool
	}{
		{"TERMUX_VERSION", strings.TrimSpace(os.Getenv("TERMUX_VERSION")) != ""},
		{"PREFIX", strings.TrimSpace(os.Getenv("PREFIX")) != ""},
		{"PROOT", hasAnyEnvPrefix("PROOT_")},
		{"QEMU", hasAnyEnvPrefix("QEMU_")},
		{"CHROOT", strings.TrimSpace(os.Getenv("CHROOT")) != ""},
	}
	for _, check := range checks {
		if check.ok {
			indicators = append(indicators, check.name)
		}
	}
	return indicators
}

func hasAnyEnvPrefix(prefix string) bool {
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, prefix) {
			return true
		}
	}
	return false
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

func buildLikelyCauseHints(plan ExecutionPlan, result Result, exists, execPerm bool, indicators []string) []string {
	var hints []string
	if !exists {
		hints = append(hints, "target path does not exist")
	}
	if exists && !execPerm {
		hints = append(hints, "target is not marked executable")
	}
	if len(indicators) > 0 {
		hints = append(hints, "execution appears to be inside proot/chroot/container-like environment; native Termux evidence is still required")
	}
	if plan.TargetClass == TargetClassRustLauncher && result.StartError != "" {
		hints = append(hints, "Rust launcher wrappers need direct kernel exec; inspect wrapper-specific /proc/self/exe behavior and target resolution")
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
