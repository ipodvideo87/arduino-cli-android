package exec

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	osExec "os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	aclscan "github.com/arduino/arduino-cli/internal/acl/elfscan"
	aclruntime "github.com/arduino/arduino-cli/internal/acl/runtime"
)

type Request struct {
	RuntimeRoot string
	TargetPath  string
	Cwd         string
	Args        []string
	Apply       bool
}

const (
	LaunchModeDirectExec     = "direct-exec"
	LaunchModeExplicitLoader = "explicit-loader"

	TargetClassAndroidNative = "android-native-elf"
	TargetClassRustLauncher  = "rust-launcher"
	TargetClassLinuxDirect   = "linux-direct-elf"
	TargetClassPatchedLinux  = "patched-linux-elf"
)

type ExecutionPlan struct {
	TargetPath        string
	Target            aclscan.Inspection
	TargetClass       string
	RuntimeRoot       string
	RuntimeID         string
	RuntimePath       string
	RuntimeArch       string
	RuntimeValidation aclruntime.ValidationReport
	LoaderPath        string
	LibraryPaths      []string
	LibrarySearchPath string
	LaunchMode        string
	Argv              []string
	Cwd               string
	Environment       []string
	Warnings          []string
	Errors            []string
	Allowed           bool
	Apply             bool
	Command           []string
}

type Result struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	StartError string
	Errno      string
}

type Planner struct {
	runtimeRoot string
	inspect     func(string) (aclscan.Inspection, error)
	runCommand  func(*osExec.Cmd) (Result, error)
}

func NewPlanner(runtimeRoot string) *Planner {
	return &Planner{
		runtimeRoot: runtimeRoot,
		inspect:     aclscan.Inspect,
		runCommand:  runCommand,
	}
}

func NewPlannerWithInspector(runtimeRoot string, inspect func(string) (aclscan.Inspection, error)) *Planner {
	if inspect == nil {
		inspect = aclscan.Inspect
	}
	return &Planner{
		runtimeRoot: runtimeRoot,
		inspect:     inspect,
		runCommand:  runCommand,
	}
}

func (p *Planner) BuildPlan(req Request) (ExecutionPlan, error) {
	plan := ExecutionPlan{
		TargetPath: strings.TrimSpace(req.TargetPath),
		Apply:      req.Apply,
	}

	targetPath, err := validateTargetPath(req.TargetPath)
	if err != nil {
		plan.Errors = append(plan.Errors, err.Error())
		return plan, err
	}
	plan.TargetPath = targetPath

	target, err := p.inspect(targetPath)
	if err != nil {
		plan.Errors = append(plan.Errors, err.Error())
		return plan, err
	}
	if !target.IsELF {
		err := fmt.Errorf("target %q is not an ELF executable", targetPath)
		plan.Errors = append(plan.Errors, err.Error())
		return plan, err
	}
	plan.Target = target

	cwd, err := resolveCWD(req.Cwd, targetPath)
	if err != nil {
		plan.Errors = append(plan.Errors, err.Error())
		return plan, err
	}

	runtimeRoot := strings.TrimSpace(req.RuntimeRoot)
	if runtimeRoot == "" {
		runtimeRoot = strings.TrimSpace(p.runtimeRoot)
	}
	if runtimeRoot != "" {
		plan.RuntimeRoot = runtimeRoot
		plan.Environment = append(plan.Environment, "ACL_RUNTIME_ROOT="+runtimeRoot)
	}

	argv := append([]string{targetPath}, req.Args...)
	plan.Argv = argv
	plan.Cwd = cwd
	plan.Allowed = true

	if target.LooksLikeRustLauncher {
		plan.Warnings = append(plan.Warnings, "Rust launcher wrapper detected; direct kernel exec is required to preserve executable identity")
	}

	if shouldUseExplicitLoader(target) {
		plan.TargetClass = TargetClassPatchedLinux
		if runtimeRoot == "" {
			runtimeRoot, err = aclruntime.DefaultRoot()
			if err != nil {
				plan.Errors = append(plan.Errors, err.Error())
				return plan, err
			}
			plan.RuntimeRoot = runtimeRoot
			plan.Environment = append(plan.Environment, "ACL_RUNTIME_ROOT="+runtimeRoot)
		}

		mgr := aclruntime.NewManager(runtimeRoot)
		activeID, err := mgr.ActiveRuntimeID()
		if err != nil {
			plan.Errors = append(plan.Errors, err.Error())
			return plan, err
		}
		if strings.TrimSpace(activeID) == "" {
			err := errors.New("no active runtime is selected")
			plan.Errors = append(plan.Errors, err.Error())
			return plan, err
		}

		rt, err := mgr.Load(activeID)
		if err != nil {
			plan.Errors = append(plan.Errors, err.Error())
			return plan, err
		}
		report, err := mgr.ValidateRuntime(rt)
		if err != nil {
			plan.Errors = append(plan.Errors, err.Error())
			return plan, err
		}
		plan.RuntimeValidation = report
		if report.Status == aclruntime.StatusFail {
			err := fmt.Errorf("active runtime %q failed validation", activeID)
			plan.Errors = append(plan.Errors, err.Error())
			return plan, err
		}

		if !architectureCompatible(rt.Manifest.Architecture, target.Machine) {
			err := fmt.Errorf("target architecture %q is incompatible with runtime architecture %q", target.Machine, rt.Manifest.Architecture)
			plan.Errors = append(plan.Errors, err.Error())
			return plan, err
		}

		if len(rt.Manifest.Loader.Path) == 0 {
			err := errors.New("active runtime is missing a loader")
			plan.Errors = append(plan.Errors, err.Error())
			return plan, err
		}
		loaderPath := filepath.Join(rt.Path, rt.Manifest.Loader.Path)
		if err := ensureRegularFile(loaderPath, "loader"); err != nil {
			plan.Errors = append(plan.Errors, err.Error())
			return plan, err
		}

		libraryPaths, librarySearchPath, err := collectRuntimeLibraries(rt)
		if err != nil {
			plan.Errors = append(plan.Errors, err.Error())
			return plan, err
		}

		plan.RuntimeID = rt.ID
		plan.RuntimePath = rt.Path
		plan.RuntimeArch = rt.Manifest.Architecture
		plan.LoaderPath = loaderPath
		plan.LibraryPaths = libraryPaths
		plan.LibrarySearchPath = librarySearchPath
		plan.Environment = append(plan.Environment,
			"ACL_RUNTIME_ID="+rt.ID,
			"ACL_RUNTIME_DIR="+rt.Path,
			"ACL_RUNTIME_LOADER="+loaderPath,
		)
		plan.LaunchMode = LaunchModeExplicitLoader
		plan.Command = append([]string{loaderPath, "--library-path", librarySearchPath, targetPath}, req.Args...)
		return plan, nil
	}

	if target.LooksLikeRustLauncher {
		plan.TargetClass = TargetClassRustLauncher
	} else if target.LooksLikeLinuxTarget {
		plan.TargetClass = TargetClassLinuxDirect
	} else {
		plan.TargetClass = TargetClassAndroidNative
	}
	plan.LaunchMode = LaunchModeDirectExec
	plan.Command = append([]string{targetPath}, req.Args...)

	return plan, nil
}

func (p *Planner) Run(req Request) (ExecutionPlan, Result, error) {
	plan, err := p.BuildPlan(req)
	if err != nil {
		return plan, Result{}, err
	}
	if !req.Apply {
		return plan, Result{}, nil
	}
	result, err := p.executePlan(plan)
	return plan, result, err
}

func (p *Planner) executePlan(plan ExecutionPlan) (Result, error) {
	if !plan.Allowed {
		return Result{ExitCode: 1}, errors.New("execution plan is not allowed")
	}
	if len(plan.Errors) > 0 {
		return Result{ExitCode: 1}, fmt.Errorf("execution plan contains errors: %s", strings.Join(plan.Errors, "; "))
	}
	if err := ensureRegularFile(plan.TargetPath, "target"); err != nil {
		return Result{ExitCode: 1}, err
	}
	target, err := p.inspect(plan.TargetPath)
	if err != nil {
		return Result{ExitCode: 1}, err
	}
	if !target.IsELF {
		return Result{ExitCode: 1}, fmt.Errorf("target %q is not an ELF executable", plan.TargetPath)
	}
	if strings.TrimSpace(plan.Cwd) == "" {
		return Result{ExitCode: 1}, errors.New("execution plan is missing cwd")
	}
	info, err := os.Stat(plan.Cwd)
	if err != nil {
		return Result{ExitCode: 1}, fmt.Errorf("cwd %q: %w", plan.Cwd, err)
	}
	if !info.IsDir() {
		return Result{ExitCode: 1}, fmt.Errorf("cwd %q is not a directory", plan.Cwd)
	}
	if len(plan.Command) == 0 {
		return Result{ExitCode: 1}, errors.New("execution plan is missing a command")
	}
	if plan.LaunchMode == LaunchModeExplicitLoader {
		if strings.TrimSpace(plan.RuntimeID) == "" {
			return Result{ExitCode: 1}, errors.New("execution plan is missing an active runtime")
		}
		if plan.RuntimeValidation.Status == aclruntime.StatusFail {
			return Result{ExitCode: 1}, fmt.Errorf("runtime %q failed validation", plan.RuntimeID)
		}
		if !architectureCompatible(plan.RuntimeArch, target.Machine) {
			return Result{ExitCode: 1}, fmt.Errorf("target architecture %q is incompatible with runtime architecture %q", target.Machine, plan.RuntimeArch)
		}
		if err := ensureLibrarySearchPath(plan.LibrarySearchPath); err != nil {
			return Result{ExitCode: 1}, err
		}
	}
	env, _ := sanitizeExecutionEnvWithAudit(os.Environ(), plan.Environment)
	cmd := osExec.Command(plan.Command[0], plan.Command[1:]...)
	cmd.Dir = plan.Cwd
	cmd.Env = env
	return p.runCommand(cmd)
}

func ensureLibrarySearchPath(searchPath string) error {
	if strings.TrimSpace(searchPath) == "" {
		return errors.New("execution plan is missing library search paths")
	}
	for _, dir := range strings.Split(searchPath, ":") {
		if strings.TrimSpace(dir) == "" {
			return errors.New("execution plan contains an empty library search path")
		}
		info, err := os.Stat(dir)
		if err != nil {
			return fmt.Errorf("library search path %q: %w", dir, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("library search path %q is not a directory", dir)
		}
	}
	return nil
}

func runCommand(cmd *osExec.Cmd) (Result, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}
	if err == nil {
		return result, nil
	}

	var exitErr *osExec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, fmt.Errorf("execution failed with exit code %d", result.ExitCode)
	}

	result.ExitCode = 1
	result.StartError = err.Error()
	result.Errno = errnoFromError(err)
	return result, fmt.Errorf("execution failed to start: %w", err)
}

type EnvironmentAudit struct {
	BaseCount        int
	KeptCount        int
	RemovedCount     int
	Kept             []string
	Removed          []string
	Indicators       []string
	SanitizedSummary string
}

func sanitizeExecutionEnv(baseEnv []string, additions []string) []string {
	env, _ := sanitizeExecutionEnvWithAudit(baseEnv, additions)
	return env
}

func sanitizeExecutionEnvWithAudit(baseEnv []string, additions []string) ([]string, EnvironmentAudit) {
	env := make([]string, 0, len(baseEnv)+len(additions))
	audit := EnvironmentAudit{BaseCount: len(baseEnv)}
	for _, kv := range baseEnv {
		if isSanitizedExecutionVariable(kv) {
			audit.Removed = append(audit.Removed, kv)
			continue
		}
		audit.Kept = append(audit.Kept, kv)
		env = append(env, kv)
	}
	for _, kv := range additions {
		if isSanitizedExecutionVariable(kv) {
			audit.Removed = append(audit.Removed, kv)
			continue
		}
		audit.Kept = append(audit.Kept, kv)
		env = append(env, kv)
	}
	audit.KeptCount = len(audit.Kept)
	audit.RemovedCount = len(audit.Removed)
	audit.Indicators = detectExecutionIndicators()
	audit.SanitizedSummary = fmt.Sprintf("%d kept, %d removed, %d indicators", audit.KeptCount, audit.RemovedCount, len(audit.Indicators))
	return env, audit
}

func errnoFromError(err error) string {
	if err == nil {
		return ""
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno.Error()
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		if errno, ok := pathErr.Err.(syscall.Errno); ok {
			return errno.Error()
		}
		return pathErr.Err.Error()
	}
	return ""
}

func isSanitizedExecutionVariable(kv string) bool {
	key := kv
	if idx := strings.IndexByte(kv, '='); idx >= 0 {
		key = kv[:idx]
	}
	switch {
	case key == "LD_PRELOAD",
		key == "LD_LIBRARY_PATH",
		key == "LD_AUDIT",
		strings.HasPrefix(key, "QEMU_"),
		strings.HasPrefix(key, "PROOT_"):
		return true
	default:
		return false
	}
}

func shouldUseExplicitLoader(target aclscan.Inspection) bool {
	if target.LooksLikeRustLauncher {
		return false
	}
	if !target.LooksLikeLinuxTarget {
		return false
	}
	return strings.TrimSpace(target.Interpreter) != ""
}

func validateTargetPath(raw string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", errors.New("missing target executable")
	}
	if containsTraversal(path) {
		return "", fmt.Errorf("target path %q contains path traversal", raw)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve target path %q: %w", raw, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("target %q: %w", abs, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("target %q is a directory", abs)
	}
	return abs, nil
}

func resolveCWD(raw, targetPath string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return filepath.Dir(targetPath), nil
	}
	if containsTraversal(raw) {
		return "", fmt.Errorf("cwd %q contains path traversal", raw)
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return "", fmt.Errorf("resolve cwd %q: %w", raw, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("cwd %q: %w", abs, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("cwd %q is not a directory", abs)
	}
	return abs, nil
}

func containsTraversal(path string) bool {
	for _, segment := range strings.Split(filepath.ToSlash(path), "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

func architectureCompatible(runtimeArch, targetMachine string) bool {
	runtimeArch = strings.ToLower(strings.TrimSpace(runtimeArch))
	targetArch := strings.ToLower(strings.TrimSpace(targetMachine))

	switch {
	case strings.Contains(targetArch, "aarch64"), strings.Contains(targetArch, "arm64"):
		return runtimeArch == "aarch64" || runtimeArch == "arm64"
	case strings.Contains(targetArch, "x86-64"), strings.Contains(targetArch, "x86_64"):
		return runtimeArch == "x86_64"
	case strings.Contains(targetArch, "80386"), strings.Contains(targetArch, "i386"), strings.Contains(targetArch, "x86"):
		return runtimeArch == "i386" || runtimeArch == "x86"
	case strings.Contains(targetArch, "arm"):
		return runtimeArch == "arm"
	case strings.Contains(targetArch, "risc-v"), strings.Contains(targetArch, "riscv"):
		return runtimeArch == "riscv64"
	default:
		return runtimeArch == targetArch || runtimeArch == strings.ReplaceAll(targetArch, "-", "")
	}
}

func ensureRegularFile(path, label string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("missing %s %q: %w", label, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s %q is a directory", label, path)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s %q must not be a symlink", label, path)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s %q is not a regular file", label, path)
	}
	return nil
}

func collectRuntimeLibraries(rt aclruntime.Runtime) ([]string, string, error) {
	paths := make([]string, 0, len(rt.Manifest.Libraries))
	searchDirs := map[string]struct{}{}

	addFile := func(file aclruntime.RuntimeFile) error {
		if strings.TrimSpace(file.Path) == "" {
			return fmt.Errorf("runtime %q is missing a declared file path", rt.ID)
		}
		path := filepath.Join(rt.Path, file.Path)
		if err := ensureRegularFile(path, file.Name); err != nil {
			return err
		}
		if file.Kind == "loader" {
			searchDirs[filepath.Dir(path)] = struct{}{}
			return nil
		}
		paths = append(paths, path)
		searchDirs[filepath.Dir(path)] = struct{}{}
		return nil
	}

	if err := addFile(rt.Manifest.Loader); err != nil {
		return nil, "", err
	}
	for _, lib := range rt.Manifest.Libraries {
		if err := addFile(lib); err != nil {
			return nil, "", err
		}
	}

	dirs := make([]string, 0, len(searchDirs))
	for dir := range searchDirs {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	return paths, strings.Join(dirs, ":"), nil
}
