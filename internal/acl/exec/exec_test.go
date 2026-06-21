package exec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	osExec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	aclscan "github.com/arduino/arduino-cli/internal/acl/elfscan"
	aclruntime "github.com/arduino/arduino-cli/internal/acl/runtime"
	"github.com/stretchr/testify/require"
)

func TestBuildPlanDryRun(t *testing.T) {
	root := t.TempDir()
	_, _ = installRuntimeFixture(t, root, "acl-exec-runtime", "stable")

	target := mustExecutable(t)
	cwd := filepath.Dir(target)
	planner := NewPlanner(root)
	plan, err := planner.BuildPlan(Request{
		RuntimeRoot: root,
		TargetPath:  target,
		Cwd:         cwd,
		Args:        []string{"--board", "esp32"},
	})
	require.NoError(t, err)
	require.True(t, plan.Allowed)
	require.Equal(t, target, plan.Argv[0])
	require.Equal(t, "--board", plan.Argv[1])
	require.Equal(t, "esp32", plan.Argv[2])
	require.Equal(t, cwd, plan.Cwd)
	require.NotEmpty(t, plan.RuntimeID)
	require.NotEmpty(t, plan.LoaderPath)
	require.NotEmpty(t, plan.LibraryPaths)
	require.Equal(t, LaunchModeDirectExec, plan.LaunchMode)
	require.Contains(t, plan.Environment, "ACL_RUNTIME_ID="+plan.RuntimeID)
	require.Contains(t, plan.Environment, "ACL_RUNTIME_ROOT="+root)
	require.NotContains(t, plan.Environment, "LD_LIBRARY_PATH="+plan.LibrarySearchPath)
	require.Equal(t, filepath.Join(plan.RuntimePath, "loader", "ld-linux-test.so"), plan.LoaderPath)
	require.Equal(t, 1, len(plan.LibraryPaths))
	require.Equal(t, filepath.Join(plan.RuntimePath, "lib", "libacl-test.so"), plan.LibraryPaths[0])
}

func TestBuildPlanRejectsMissingTarget(t *testing.T) {
	planner := NewPlanner(t.TempDir())
	_, err := planner.BuildPlan(Request{RuntimeRoot: t.TempDir(), TargetPath: ""})
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing target executable")
}

func TestBuildPlanRejectsNonELF(t *testing.T) {
	root := t.TempDir()
	_, _ = installRuntimeFixture(t, root, "acl-exec-runtime", "stable")

	target := filepath.Join(t.TempDir(), "not-elf.txt")
	require.NoError(t, os.WriteFile(target, []byte("plain text"), 0o644))

	planner := NewPlanner(root)
	_, err := planner.BuildPlan(Request{RuntimeRoot: root, TargetPath: target})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not an ELF")
}

func TestBuildPlanRejectsMissingActiveRuntime(t *testing.T) {
	target := mustExecutable(t)
	planner := NewPlanner(t.TempDir())
	_, err := planner.BuildPlan(Request{RuntimeRoot: t.TempDir(), TargetPath: target})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no active runtime")
}

func TestBuildPlanRejectsIncompatibleRuntime(t *testing.T) {
	root := t.TempDir()
	_, _ = installRuntimeFixture(t, root, "acl-exec-runtime", "stable")

	target := mustExecutable(t)
	planner := NewPlannerWithInspector(root, func(string) (aclscan.Inspection, error) {
		return aclscan.Inspection{
			Path:                 target,
			Exists:               true,
			IsELF:                true,
			Machine:              "ARM",
			FileType:             "EXEC",
			LooksLikeLinuxTarget: true,
		}, nil
	})
	plan, err := planner.BuildPlan(Request{
		RuntimeRoot: root,
		TargetPath:  target,
	})
	require.Error(t, err)
	require.False(t, plan.Allowed)
	require.Contains(t, err.Error(), "incompatible")
}

func TestBuildPlanConstructsEnvironmentAndPreservesArgv(t *testing.T) {
	root := t.TempDir()
	_, _ = installRuntimeFixture(t, root, "acl-exec-runtime", "stable")

	target := mustExecutable(t)
	customCwd := t.TempDir()
	planner := NewPlanner(root)
	plan, err := planner.BuildPlan(Request{
		RuntimeRoot: root,
		TargetPath:  target,
		Cwd:         customCwd,
		Args:        []string{"one", "two"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{target, "one", "two"}, plan.Argv)
	require.Equal(t, customCwd, plan.Cwd)
	require.Contains(t, plan.Environment, "ACL_RUNTIME_ID="+plan.RuntimeID)
	require.Contains(t, plan.Environment, "ACL_RUNTIME_DIR="+plan.RuntimePath)
	require.Contains(t, plan.Environment, "ACL_RUNTIME_LOADER="+plan.LoaderPath)
	require.NotContains(t, plan.Environment, "LD_LIBRARY_PATH="+plan.LibrarySearchPath)
	require.Equal(t, LaunchModeDirectExec, plan.LaunchMode)
}

func TestBuildPlanWarnsOnRustLauncherAndUsesDirectExec(t *testing.T) {
	root := t.TempDir()
	_, _ = installRuntimeFixture(t, root, "acl-exec-runtime", "stable")

	target := mustExecutable(t)
	planner := NewPlannerWithInspector(root, func(string) (aclscan.Inspection, error) {
		return aclscan.Inspection{
			Path:                  target,
			Exists:                true,
			IsELF:                 true,
			Machine:               hostArchitecture(t, target),
			FileType:              "EXEC",
			Interpreter:           "/lib64/ld-linux-aarch64.so.1",
			LooksLikeLinuxTarget:  true,
			LooksLikeRustLauncher: true,
		}, nil
	})

	plan, err := planner.BuildPlan(Request{
		RuntimeRoot: root,
		TargetPath:  target,
	})
	require.NoError(t, err)
	require.Equal(t, LaunchModeDirectExec, plan.LaunchMode)
	require.Equal(t, []string{target}, plan.Command[:1])
	require.NotEmpty(t, plan.Warnings)
	require.Contains(t, strings.Join(plan.Warnings, " "), "Rust launcher")
}

func TestBuildPlanRejectsUnsafeCWD(t *testing.T) {
	root := t.TempDir()
	_, _ = installRuntimeFixture(t, root, "acl-exec-runtime", "stable")

	target := mustExecutable(t)
	planner := NewPlanner(root)
	_, err := planner.BuildPlan(Request{
		RuntimeRoot: root,
		TargetPath:  target,
		Cwd:         "../escape",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "path traversal")
}

func TestRunApplyDisabledByDefault(t *testing.T) {
	root := t.TempDir()
	_, _ = installRuntimeFixture(t, root, "acl-exec-runtime", "stable")

	target := mustExecutable(t)
	planner := NewPlanner(root)
	plan, result, err := planner.Run(Request{
		RuntimeRoot: root,
		TargetPath:  target,
		Args:        []string{"--version"},
	})
	require.NoError(t, err)
	require.True(t, plan.Allowed)
	require.Zero(t, result)
}

func TestRunApplyUsesBackendCommandAndPreservesCWD(t *testing.T) {
	root := t.TempDir()
	_, _ = installRuntimeFixture(t, root, "acl-exec-runtime", "stable")

	target := mustExecutable(t)
	customCwd := t.TempDir()
	planner := NewPlanner(root)
	var gotArgs []string
	var gotEnv []string
	var gotDir string
	planner.runCommand = func(cmd *osExec.Cmd) (Result, error) {
		gotArgs = append([]string(nil), cmd.Args...)
		gotEnv = append([]string(nil), cmd.Env...)
		gotDir = cmd.Dir
		return Result{Stdout: "ok", ExitCode: 0}, nil
	}

	plan, result, err := planner.Run(Request{
		RuntimeRoot: root,
		TargetPath:  target,
		Cwd:         customCwd,
		Args:        []string{"one", "two"},
		Apply:       true,
	})
	require.NoError(t, err)
	require.Equal(t, "ok", result.Stdout)
	require.Equal(t, customCwd, gotDir)
	require.Equal(t, plan.Command, gotArgs)
	require.Contains(t, gotEnv, "ACL_RUNTIME_ID="+plan.RuntimeID)
	require.NotContains(t, gotEnv, "LD_LIBRARY_PATH="+plan.LibrarySearchPath)
	require.NotContains(t, gotEnv, "LD_PRELOAD=")
}

func TestRunApplyCapturesStdoutStderrAndExitCode(t *testing.T) {
	target := writeExecutableScript(t, `#!/usr/bin/env bash
echo "loader-stdout:$PWD"
echo "loader-stderr:$1:$2:$3" >&2
exit 7
`)
	hostArch := hostArchitecture(t, mustExecutable(t))
	planner := NewPlanner(t.TempDir())
	planner.inspect = func(string) (aclscan.Inspection, error) {
		return aclscan.Inspection{
			Path:                 target,
			Exists:               true,
			IsELF:                true,
			Machine:              hostArch,
			FileType:             "EXEC",
			LooksLikeLinuxTarget: true,
		}, nil
	}
	plan := ExecutionPlan{
		TargetPath:        target,
		Target:            aclscan.Inspection{Path: target, Exists: true, IsELF: true, Machine: hostArch, FileType: "EXEC", LooksLikeLinuxTarget: true},
		RuntimeID:         "acl-exec-runtime",
		RuntimeArch:       hostArch,
		RuntimeValidation: aclruntime.ValidationReport{RuntimeID: "acl-exec-runtime", Status: aclruntime.StatusPass},
		Cwd:               t.TempDir(),
		Environment:       []string{"ACL_RUNTIME_ID=acl-exec-runtime", "LD_LIBRARY_PATH=/tmp/termux-glibc", "LD_PRELOAD=/tmp/boom.so"},
		Command:           []string{target, "--version"},
		Allowed:           true,
		Apply:             true,
	}

	result, err := planner.executePlan(plan)
	require.Error(t, err)
	require.Contains(t, err.Error(), "execution failed with exit code 7")
	require.Equal(t, 7, result.ExitCode)
	require.Contains(t, result.Stdout, "loader-stdout:")
	require.Contains(t, result.Stderr, "loader-stderr:--version")
}

func TestExecutePlanReturnsStartErrorForMissingCommand(t *testing.T) {
	target := writeExecutableScript(t, "#!/usr/bin/env bash\nexit 0\n")
	hostArch := hostArchitecture(t, mustExecutable(t))
	missingCommand := filepath.Join(t.TempDir(), "missing-command")
	planner := NewPlanner(t.TempDir())
	planner.inspect = func(string) (aclscan.Inspection, error) {
		return aclscan.Inspection{
			Path:                 target,
			Exists:               true,
			IsELF:                true,
			Machine:              hostArch,
			FileType:             "EXEC",
			LooksLikeLinuxTarget: true,
		}, nil
	}
	plan := ExecutionPlan{
		TargetPath:        target,
		Target:            aclscan.Inspection{Path: target, Exists: true, IsELF: true, Machine: hostArch, FileType: "EXEC", LooksLikeLinuxTarget: true},
		RuntimeID:         "acl-exec-runtime",
		RuntimeArch:       hostArch,
		RuntimeValidation: aclruntime.ValidationReport{RuntimeID: "acl-exec-runtime", Status: aclruntime.StatusPass},
		Cwd:               t.TempDir(),
		Environment:       []string{"ACL_RUNTIME_ID=acl-exec-runtime"},
		Command:           []string{missingCommand},
		Allowed:           true,
		Apply:             true,
	}

	result, err := planner.executePlan(plan)
	require.Error(t, err)
	require.Equal(t, 1, result.ExitCode)
	require.Contains(t, err.Error(), "execution failed to start")
}

func TestBuildPlanRejectsInvalidActiveRuntime(t *testing.T) {
	root := t.TempDir()
	installed, _ := installRuntimeFixture(t, root, "acl-exec-runtime", "stable")
	require.NoError(t, os.Remove(filepath.Join(installed.Path, "lib", "libacl-test.so")))

	target := mustExecutable(t)
	planner := NewPlanner(root)
	_, err := planner.BuildPlan(Request{
		RuntimeRoot: root,
		TargetPath:  target,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed validation")
}

func TestExecutePlanDirectExecDoesNotRequireLibrarySearchPath(t *testing.T) {
	target := writeExecutableScript(t, "#!/usr/bin/env bash\nexit 0\n")
	hostArch := hostArchitecture(t, mustExecutable(t))
	planner := NewPlanner(t.TempDir())
	planner.inspect = func(string) (aclscan.Inspection, error) {
		return aclscan.Inspection{
			Path:                 target,
			Exists:               true,
			IsELF:                true,
			Machine:              hostArch,
			FileType:             "EXEC",
			LooksLikeLinuxTarget: true,
		}, nil
	}
	plan := ExecutionPlan{
		TargetPath:        target,
		Target:            aclscan.Inspection{Path: target, Exists: true, IsELF: true, Machine: hostArch, FileType: "EXEC", LooksLikeLinuxTarget: true},
		RuntimeID:         "acl-exec-runtime",
		RuntimeArch:       hostArch,
		RuntimeValidation: aclruntime.ValidationReport{RuntimeID: "acl-exec-runtime", Status: aclruntime.StatusPass},
		Cwd:               t.TempDir(),
		Environment:       []string{"ACL_RUNTIME_ID=acl-exec-runtime"},
		Command:           []string{target},
		Allowed:           true,
		Apply:             true,
	}

	result, err := planner.executePlan(plan)
	require.NoError(t, err)
	require.Equal(t, 0, result.ExitCode)
}

func installRuntimeFixture(t *testing.T, root, runtimeID, compatibility string) (aclruntime.Runtime, string) {
	t.Helper()

	exe := mustExecutable(t)
	packageDir := filepath.Join(t.TempDir(), runtimeID)
	require.NoError(t, os.MkdirAll(filepath.Join(packageDir, "loader"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(packageDir, "lib"), 0o755))

	loaderPath := filepath.Join(packageDir, "loader", "ld-linux-test.so")
	libPath := filepath.Join(packageDir, "lib", "libacl-test.so")
	copyFile(t, exe, loaderPath)
	copyFile(t, exe, libPath)

	loaderHash := fileHash(t, loaderPath)
	libHash := fileHash(t, libPath)
	manifest := aclruntime.Manifest{
		SchemaVersion:      "1.0",
		RuntimeID:          runtimeID,
		RuntimeVersion:     "0.1.0",
		Architecture:       hostArchitecture(t, exe),
		SupportedABIs:      []string{hostABI(runtime.GOARCH)},
		CompatibilityLevel: compatibility,
		CreatedAt:          fixedTime(),
		Loader: aclruntime.RuntimeFile{
			Name:     "ld-linux-test.so",
			Path:     "loader/ld-linux-test.so",
			Kind:     "loader",
			Required: true,
			SHA256:   loaderHash,
		},
		Libraries: []aclruntime.RuntimeFile{{
			Name:     "libacl-test.so",
			Path:     "lib/libacl-test.so",
			Kind:     "library",
			Required: true,
			SHA256:   libHash,
		}},
		Hashes: map[string]string{
			"loader/ld-linux-test.so": loaderHash,
			"lib/libacl-test.so":      libHash,
		},
		Build: aclruntime.BuildInfo{
			Tool:      "acl-exec-test",
			Builder:   "test",
			GoVersion: runtime.Version(),
			BuiltAt:   fixedTime(),
			HostOS:    runtime.GOOS,
			HostArch:  runtime.GOARCH,
		},
	}
	writeManifest(t, filepath.Join(packageDir, aclruntime.ManifestFileName), manifest)

	mgr := aclruntime.NewManager(root)
	installed, err := mgr.InstallFromDir(packageDir)
	require.NoError(t, err)
	require.NoError(t, mgr.Activate(installed.ID))
	return installed, packageDir
}

func writeManifest(t *testing.T, path string, manifest aclruntime.Manifest) {
	t.Helper()
	data, err := json.MarshalIndent(manifest, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(dst, data, 0o755))
}

func fileHash(t *testing.T, path string) string {
	t.Helper()
	sum, err := sha256Hex(path)
	require.NoError(t, err)
	return sum
}

func hostArchitecture(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	var hdr [20]byte
	_, err = f.ReadAt(hdr[:], 0)
	require.NoError(t, err)
	switch hdr[18] {
	case 0xb7:
		return "aarch64"
	case 0x3e:
		return "x86_64"
	case 0x28:
		return "arm"
	default:
		return runtime.GOARCH
	}
}

func hostABI(goarch string) string {
	switch goarch {
	case "arm64":
		return "android-aarch64"
	case "amd64":
		return "android-x86_64"
	case "386":
		return "android-i686"
	case "arm":
		return "android-arm"
	default:
		return "android-" + goarch
	}
}

func fixedTime() string {
	return time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
}

func mustExecutable(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	require.NoError(t, err)
	return exe
}

func sha256Hex(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func writeExecutableScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "loader.sh")
	require.NoError(t, os.WriteFile(path, []byte(strings.TrimSpace(body)+"\n"), 0o755))
	return path
}
