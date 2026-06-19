package exec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	aclscan "github.com/arduino/arduino-cli/internal/acl/elfscan"
	aclruntime "github.com/arduino/arduino-cli/internal/acl/runtime"
)

var ErrBackendNotImplemented = errors.New("execution backend not implemented")

type Request struct {
	RuntimeRoot string
	TargetPath  string
	Cwd         string
	Args        []string
	Apply       bool
}

type ExecutionPlan struct {
	TargetPath        string
	Target            aclscan.Inspection
	RuntimeID         string
	RuntimePath       string
	RuntimeValidation aclruntime.ValidationReport
	LoaderPath        string
	LibraryPaths      []string
	LibrarySearchPath string
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
	Stdout   string
	Stderr   string
	ExitCode int
}

type Planner struct {
	runtimeRoot string
	inspect     func(string) (aclscan.Inspection, error)
}

func NewPlanner(runtimeRoot string) *Planner {
	return &Planner{
		runtimeRoot: runtimeRoot,
		inspect:     aclscan.Inspect,
	}
}

func NewPlannerWithInspector(runtimeRoot string, inspect func(string) (aclscan.Inspection, error)) *Planner {
	if inspect == nil {
		inspect = aclscan.Inspect
	}
	return &Planner{
		runtimeRoot: runtimeRoot,
		inspect:     inspect,
	}
}

func (p *Planner) BuildPlan(req Request) (ExecutionPlan, error) {
	plan := ExecutionPlan{
		TargetPath: strings.TrimSpace(req.TargetPath),
		Apply:      req.Apply,
	}
	runtimeRoot := strings.TrimSpace(req.RuntimeRoot)
	if runtimeRoot == "" {
		runtimeRoot = strings.TrimSpace(p.runtimeRoot)
	}
	if runtimeRoot == "" {
		plan.Errors = append(plan.Errors, "missing runtime root")
		return plan, errors.New("missing runtime root")
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

	cwd, err := resolveCWD(req.Cwd, targetPath)
	if err != nil {
		plan.Errors = append(plan.Errors, err.Error())
		return plan, err
	}

	argv := append([]string{targetPath}, req.Args...)
	env := []string{
		"ACL_RUNTIME_ROOT=" + runtimeRoot,
		"ACL_RUNTIME_ID=" + rt.ID,
		"ACL_RUNTIME_DIR=" + rt.Path,
		"ACL_RUNTIME_LOADER=" + loaderPath,
		"LD_LIBRARY_PATH=" + librarySearchPath,
	}

	plan.RuntimeID = rt.ID
	plan.RuntimePath = rt.Path
	plan.LoaderPath = loaderPath
	plan.LibraryPaths = libraryPaths
	plan.LibrarySearchPath = librarySearchPath
	plan.Argv = argv
	plan.Cwd = cwd
	plan.Environment = env
	plan.Command = append([]string{loaderPath, "--library-path", librarySearchPath, targetPath}, req.Args...)
	plan.Allowed = true

	if target.Interpreter == "" {
		plan.Warnings = append(plan.Warnings, "target has no PT_INTERP entry")
	}
	if len(target.HardcodedAbsolutePaths) > 0 {
		plan.Warnings = append(plan.Warnings, "target contains hardcoded absolute paths")
	}
	if !target.LooksLikeLinuxTarget {
		plan.Warnings = append(plan.Warnings, "target does not look like a glibc/Linux binary")
	}

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
	return plan, Result{}, ErrBackendNotImplemented
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
