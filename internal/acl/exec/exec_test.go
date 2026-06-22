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
	target := mustExecutable(t)
	hostArch := hostArchitecture(t, target)
	cwd := filepath.Dir(target)
	root := t.TempDir()
	planner := NewPlannerWithInspector(root, directExecInspection(target, hostArch))
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
	require.Empty(t, plan.RuntimeID)
	require.Empty(t, plan.LoaderPath)
	require.Empty(t, plan.LibraryPaths)
	require.Equal(t, TargetClassAndroidNative, plan.TargetClass)
	require.Equal(t, LaunchModeDirectExec, plan.LaunchMode)
	require.Contains(t, plan.Environment, "ACL_RUNTIME_ROOT="+root)
	require.Equal(t, []string{target, "--board", "esp32"}, plan.Command)
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
	hostArch := hostArchitecture(t, target)
	planner := NewPlannerWithInspector(t.TempDir(), explicitLoaderInspection(target, hostArch))
	_, err := planner.BuildPlan(Request{TargetPath: target})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no active runtime")
}

func TestBuildPlanRejectsIncompatibleRuntime(t *testing.T) {
	root := t.TempDir()
	_, _ = installRuntimeFixture(t, root, "acl-exec-runtime", "stable")

	target := mustExecutable(t)
	planner := NewPlannerWithInspector(root, func(string) (aclscan.Inspection, error) {
		return explicitLoaderInspectionWithMachine(target, "ARM"), nil
	})
	_, err := planner.BuildPlan(Request{
		RuntimeRoot: root,
		TargetPath:  target,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "incompatible")
}

func TestBuildPlanConstructsEnvironmentAndPreservesArgv(t *testing.T) {
	root := t.TempDir()

	target := mustExecutable(t)
	hostArch := hostArchitecture(t, target)
	customCwd := t.TempDir()
	planner := NewPlannerWithInspector(root, directExecInspection(target, hostArch))
	plan, err := planner.BuildPlan(Request{
		RuntimeRoot: root,
		TargetPath:  target,
		Cwd:         customCwd,
		Args:        []string{"one", "two"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{target, "one", "two"}, plan.Argv)
	require.Equal(t, customCwd, plan.Cwd)
	require.Equal(t, TargetClassAndroidNative, plan.TargetClass)
	require.Equal(t, LaunchModeDirectExec, plan.LaunchMode)
	require.Contains(t, plan.Environment, "ACL_RUNTIME_ROOT="+root)
}

func TestBuildPlanWarnsOnRustLauncherAndUsesDirectExec(t *testing.T) {
	root := t.TempDir()
	target := mustExecutable(t)
	hostArch := hostArchitecture(t, target)
	planner := NewPlannerWithInspector(root, rustLauncherInspection(target, hostArch))

	plan, err := planner.BuildPlan(Request{
		RuntimeRoot: root,
		TargetPath:  target,
	})
	require.NoError(t, err)
	require.Equal(t, LaunchModeDirectExec, plan.LaunchMode)
	require.Equal(t, TargetClassRustLauncher, plan.TargetClass)
	require.Empty(t, plan.LoaderPath)
	require.Equal(t, []string{target}, plan.Command)
	require.NotEmpty(t, plan.Warnings)
	require.Contains(t, strings.Join(plan.Warnings, " "), "Rust launcher")
}

func TestBuildPlanUsesExplicitLoaderForPatchedLinuxELF(t *testing.T) {
	root := t.TempDir()
	installed, _ := installRuntimeFixture(t, root, "acl-exec-runtime", "stable")

	target := mustExecutable(t)
	hostArch := hostArchitecture(t, target)
	planner := NewPlannerWithInspector(root, explicitLoaderInspection(target, hostArch))

	plan, err := planner.BuildPlan(Request{
		RuntimeRoot: root,
		TargetPath:  target,
		Args:        []string{"--version"},
	})
	require.NoError(t, err)
	require.Equal(t, LaunchModeExplicitLoader, plan.LaunchMode)
	require.Equal(t, TargetClassPatchedLinux, plan.TargetClass)
	require.NotEmpty(t, plan.RuntimeID)
	require.Equal(t, installed.ID, plan.RuntimeID)
	require.NotEmpty(t, plan.LoaderPath)
	require.NotEmpty(t, plan.LibrarySearchPath)
	require.Equal(t, filepath.Join(plan.RuntimePath, "loader", "ld-linux-test.so"), plan.LoaderPath)
	require.Contains(t, plan.Environment, "ACL_RUNTIME_ID="+plan.RuntimeID)
	require.Contains(t, plan.Environment, "ACL_RUNTIME_DIR="+plan.RuntimePath)
	require.Contains(t, plan.Environment, "ACL_RUNTIME_LOADER="+plan.LoaderPath)
	require.Equal(t, []string{plan.LoaderPath, "--library-path", plan.LibrarySearchPath, target, "--version"}, plan.Command)
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

	target := mustExecutable(t)
	hostArch := hostArchitecture(t, target)
	customCwd := t.TempDir()
	t.Setenv("HOME", "/home/test")
	t.Setenv("PATH", "/usr/bin:/bin")
	t.Setenv("TMPDIR", "/tmp")
	t.Setenv("ANDROID_ROOT", "/system")
	t.Setenv("ANDROID_DATA", "/data")
	t.Setenv("TERMUX_VERSION", "0.118.0")
	t.Setenv("ACL_RUNTIME_ROOT", root)
	t.Setenv("LD_LIBRARY_PATH", "/tmp/termux-glibc")
	t.Setenv("LD_PRELOAD", "/tmp/boom.so")
	t.Setenv("LD_AUDIT", "/tmp/audit.so")
	t.Setenv("QEMU_LD_PREFIX", "/tmp/qemu")
	t.Setenv("PROOT_NO_SECCOMP", "1")
	planner := NewPlannerWithInspector(root, directExecInspection(target, hostArch))
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
	require.Contains(t, gotEnv, "ACL_RUNTIME_ROOT="+root)
	require.Contains(t, gotEnv, "HOME=/home/test")
	require.Contains(t, gotEnv, "PATH=/usr/bin:/bin")
	require.Contains(t, gotEnv, "TMPDIR=/tmp")
	require.Contains(t, gotEnv, "ANDROID_ROOT=/system")
	require.Contains(t, gotEnv, "ANDROID_DATA=/data")
	require.Contains(t, gotEnv, "TERMUX_VERSION=0.118.0")
	require.NotContains(t, gotEnv, "LD_LIBRARY_PATH=")
	require.NotContains(t, gotEnv, "LD_PRELOAD=")
	require.NotContains(t, gotEnv, "LD_AUDIT=")
	require.NotContains(t, gotEnv, "QEMU_LD_PREFIX=")
	require.NotContains(t, gotEnv, "PROOT_NO_SECCOMP=")
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
	hostArch := hostArchitecture(t, target)
	planner := NewPlannerWithInspector(root, explicitLoaderInspection(target, hostArch))
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

func TestSanitizeExecutionEnvDropsLeakyVariables(t *testing.T) {
	baseEnv := []string{
		"HOME=/home/user",
		"PATH=/usr/bin",
		"TMPDIR=/tmp",
		"ANDROID_ROOT=/system",
		"ANDROID_DATA=/data",
		"TERMUX_VERSION=0.118.0",
		"ACL_RUNTIME_ROOT=/data/data/com.termux/files/home/.arduino-cli-android/acl-runtime",
		"LD_LIBRARY_PATH=/tmp/termux-glibc",
		"LD_PRELOAD=/tmp/boom.so",
		"LD_AUDIT=/tmp/audit.so",
		"QEMU_LD_PREFIX=/tmp/qemu",
		"PROOT_NO_SECCOMP=1",
	}
	additions := []string{
		"ACL_RUNTIME_ID=acl-exec-runtime",
		"LD_PRELOAD=/tmp/override.so",
		"QEMU_CPU=host",
	}

	env := sanitizeExecutionEnv(baseEnv, additions)
	require.Contains(t, env, "HOME=/home/user")
	require.Contains(t, env, "PATH=/usr/bin")
	require.Contains(t, env, "TMPDIR=/tmp")
	require.Contains(t, env, "ANDROID_ROOT=/system")
	require.Contains(t, env, "ANDROID_DATA=/data")
	require.Contains(t, env, "TERMUX_VERSION=0.118.0")
	require.Contains(t, env, "ACL_RUNTIME_ROOT=/data/data/com.termux/files/home/.arduino-cli-android/acl-runtime")
	require.Contains(t, env, "ACL_RUNTIME_ID=acl-exec-runtime")
	require.NotContains(t, env, "LD_LIBRARY_PATH=/tmp/termux-glibc")
	require.NotContains(t, env, "LD_PRELOAD=/tmp/boom.so")
	require.NotContains(t, env, "LD_AUDIT=/tmp/audit.so")
	require.NotContains(t, env, "QEMU_LD_PREFIX=/tmp/qemu")
	require.NotContains(t, env, "PROOT_NO_SECCOMP=1")
	require.NotContains(t, env, "QEMU_CPU=host")
}

func TestBuildDiagnosticReportMatchesDirectExecPlan(t *testing.T) {
	root := t.TempDir()
	target := mustExecutable(t)
	hostArch := hostArchitecture(t, target)
	originalReader := procSelfRootReader
	originalDetector := executionEnvironmentDetector
	procSelfRootReader = func() string { return "/" }
	executionEnvironmentDetector = func() EnvironmentDetection {
		return EnvironmentDetection{
			State:       EnvironmentStateNativeTermux,
			Description: "native Termux detected",
			Evidence: []string{
				"TERMUX_VERSION=0.118.0",
				"PREFIX=/data/data/com.termux/files/usr",
				"TERMUX_PREFIX=/data/data/com.termux/files/usr",
				"TERMUX_HOME=/data/data/com.termux/files/home",
			},
		}
	}
	t.Cleanup(func() {
		procSelfRootReader = originalReader
		executionEnvironmentDetector = originalDetector
	})
	t.Setenv("HOME", "/home/test")
	t.Setenv("PATH", "/usr/bin:/bin")
	t.Setenv("TMPDIR", "/tmp")
	t.Setenv("ANDROID_ROOT", "/system")
	t.Setenv("ANDROID_DATA", "/data")
	t.Setenv("TERMUX_VERSION", "0.118.0")
	t.Setenv("PREFIX", "/data/data/com.termux/files/usr")
	t.Setenv("TERMUX__PREFIX", "/data/data/com.termux/files/usr")
	t.Setenv("TERMUX__HOME", "/data/data/com.termux/files/home")
	t.Setenv("TERMUX__ROOTFS_DIR", "/data/data/com.termux/files")
	t.Setenv("ACL_RUNTIME_ROOT", root)
	t.Setenv("LD_LIBRARY_PATH", "/tmp/termux-glibc")

	planner := NewPlannerWithInspector(root, directExecInspection(target, hostArch))
	plan, err := planner.BuildPlan(Request{
		RuntimeRoot: root,
		TargetPath:  target,
		Cwd:         filepath.Dir(target),
		Args:        []string{"--verbose"},
	})
	require.NoError(t, err)

	report := BuildDiagnosticReport(plan, Request{
		RuntimeRoot: root,
		TargetPath:  target,
		Cwd:         filepath.Dir(target),
		Args:        []string{"--verbose"},
		Apply:       false,
	}, Result{})

	require.Equal(t, target, report.Target.Path)
	require.Equal(t, filepath.Base(target), report.Target.Basename)
	require.True(t, report.Target.Exists)
	require.Equal(t, TargetClassAndroidNative, report.Target.TargetClass)
	require.Equal(t, string(EnvironmentStateNativeTermux), report.Environment.State)
	require.Equal(t, "native Termux detected", report.Environment.Description)
	require.Equal(t, "configured", report.Environment.RuntimeRootSource)
	require.Equal(t, root, report.Environment.RuntimeRoot)
	require.True(t, report.Execution.DirectExecution)
	require.False(t, report.Execution.ExplicitLoader)
	require.Empty(t, report.Execution.LoaderPath)
	require.Contains(t, report.Environment.SanitizedSummary, "removed")
	require.Contains(t, strings.Join(report.Environment.Removed, " "), "LD_LIBRARY_PATH=/tmp/termux-glibc")
	require.Contains(t, strings.Join(report.Environment.Indicators, " "), "TERMUX_VERSION=0.118.0")
	require.Contains(t, strings.Join(report.Environment.Indicators, " "), "TERMUX_PREFIX=/data/data/com.termux/files/usr")
	require.Contains(t, strings.Join(report.Environment.Indicators, " "), "TERMUX_HOME=/data/data/com.termux/files/home")
	require.NotContains(t, strings.Join(report.Environment.Indicators, " "), "PROOT")
	require.NotContains(t, strings.Join(report.Hints, " "), "PRoot/proot-distro")
	require.Equal(t, []string{target, "--verbose"}, report.Runtime.Argv)
	require.Equal(t, hostArch, report.TargetData.Machine)
	require.True(t, report.Result.Started == false)
	require.Empty(t, report.Result.StartError)
	require.Nil(t, report.Result.ExitCode)
}

func TestBuildDiagnosticReportDetectsDefaultRuntimeRoot(t *testing.T) {
	target := mustExecutable(t)
	hostArch := hostArchitecture(t, target)
	t.Setenv("HOME", "/home/test")
	t.Setenv("PREFIX", "/data/data/com.termux/files/usr")
	t.Setenv("ACL_RUNTIME_ROOT", "")

	originalDetector := executionEnvironmentDetector
	executionEnvironmentDetector = func() EnvironmentDetection {
		return EnvironmentDetection{
			State:       EnvironmentStateUnknown,
			Description: "environment unknown",
		}
	}
	t.Cleanup(func() {
		executionEnvironmentDetector = originalDetector
	})

	plan := ExecutionPlan{
		TargetPath: target,
		Target: aclscan.Inspection{
			Path:                 target,
			IsELF:                true,
			Machine:              hostArch,
			FileType:             "EXEC",
			LooksLikeLinuxTarget: false,
		},
		LaunchMode: LaunchModeDirectExec,
		Argv:       []string{target},
	}

	report := BuildDiagnosticReport(plan, Request{TargetPath: target}, Result{})
	require.Equal(t, "/home/test/.arduino-cli-android/acl-runtime", report.Environment.RuntimeRoot)
	require.Equal(t, "detected", report.Environment.RuntimeRootSource)
	require.Equal(t, "environment unknown", report.Environment.Description)
}

func TestBuildDiagnosticReportClassifiesProotEnvironment(t *testing.T) {
	env := detectExecutionEnvironmentFromEnv([]string{
		"TERMUX_VERSION=0.118.0",
		"PREFIX=/data/data/com.termux/files/usr",
		"TERMUX_PREFIX=/data/data/com.termux/files/usr",
		"HOME=/data/data/com.termux/files/home",
		"TERMUX_HOME=/data/data/com.termux/files/home",
		"PROOT_NO_SECCOMP=1",
		"QEMU_LD_PREFIX=/data/data/com.termux/files/usr/var/lib/proot-distro/installed-rootfs/debian",
	}, "/data/data/com.termux/files/usr/var/lib/proot-distro/installed-rootfs/debian")
	require.Equal(t, EnvironmentStateProot, env.State)
	require.Equal(t, "PRoot/proot-distro detected", env.Description)
	require.Contains(t, strings.Join(env.Evidence, " "), "PROOT_*")
	require.Contains(t, strings.Join(env.Evidence, " "), "TERMUX_PREFIX=/data/data/com.termux/files/usr")
	require.Contains(t, strings.Join(env.Evidence, " "), "TERMUX_HOME=/data/data/com.termux/files/home")
	require.Contains(t, strings.Join(env.Evidence, " "), "/proc/self/root=")
}

func TestDetectExecutionEnvironmentUnknown(t *testing.T) {
	env := detectExecutionEnvironmentFromEnv([]string{
		"HOME=/home/test",
		"PATH=/usr/bin:/bin",
		"LANG=C",
	}, "/")

	require.Equal(t, EnvironmentStateUnknown, env.State)
	require.Equal(t, "environment unknown", env.Description)
	require.Empty(t, env.Evidence)
}

func TestBuildDiagnosticReportIncludesStartFailureEvidence(t *testing.T) {
	target := "/tmp/missing-tool"
	originalReader := procSelfRootReader
	originalDetector := executionEnvironmentDetector
	procSelfRootReader = func() string { return "/" }
	executionEnvironmentDetector = func() EnvironmentDetection {
		return EnvironmentDetection{
			State:       EnvironmentStateUnknown,
			Description: "environment unknown",
		}
	}
	t.Cleanup(func() {
		procSelfRootReader = originalReader
		executionEnvironmentDetector = originalDetector
	})
	t.Setenv("TERMUX_VERSION", "")
	t.Setenv("PREFIX", "/usr")
	t.Setenv("HOME", "/home/test")
	t.Setenv("TERMUX__PREFIX", "")
	t.Setenv("TERMUX__HOME", "")
	t.Setenv("TERMUX__ROOTFS_DIR", "")
	t.Setenv("PROOT_NO_SECCOMP", "")
	t.Setenv("QEMU_LD_PREFIX", "")
	report := BuildDiagnosticReport(ExecutionPlan{
		TargetPath:  target,
		TargetClass: TargetClassRustLauncher,
		LaunchMode:  LaunchModeDirectExec,
		Target: aclscan.Inspection{
			Path:                  target,
			IsELF:                 true,
			LooksLikeRustLauncher: true,
			LooksLikeLinuxTarget:  true,
			Interpreter:           "/lib64/ld-linux-aarch64.so.1",
			ImportedLibraries:     []string{"libc.so.6"},
		},
		Argv: []string{target, "--version"},
	}, Request{TargetPath: target, Apply: true}, Result{
		ExitCode:   1,
		StartError: "execution failed to start: fork/exec /tmp/missing-tool: no such file or directory",
		Errno:      "no such file or directory",
	})

	require.Contains(t, strings.Join(report.Hints, " "), "environment could not be classified")
	require.Contains(t, strings.Join(report.Hints, " "), "Rust launcher wrappers need direct kernel exec")
	require.Equal(t, "no such file or directory", report.Result.Errno)
	require.Equal(t, 1, *report.Result.ExitCode)
	require.False(t, report.Result.Started)
}

func TestBuildDiagnosticReportIncludesRustLauncherDelegateTargetsAndEACCESHint(t *testing.T) {
	target := mustExecutable(t)
	hostArch := hostArchitecture(t, target)
	report := BuildDiagnosticReport(ExecutionPlan{
		TargetPath:  target,
		TargetClass: TargetClassRustLauncher,
		LaunchMode:  LaunchModeDirectExec,
		Target: aclscan.Inspection{
			Path:                  target,
			IsELF:                 true,
			Machine:               hostArch,
			LooksLikeRustLauncher: true,
			LooksLikeLinuxTarget:  true,
			Interpreter:           "/lib64/ld-linux-aarch64.so.1",
			ImportedLibraries:     []string{"libc.so.6"},
			LauncherDelegateTargets: []aclscan.LauncherDelegateTarget{{
				Path:       "/tmp/xtensa-esp32s3-elf-gcc.real",
				Exists:     true,
				Executable: false,
				Mode:       "-rw-r--r--",
				Source:     "basename-variant",
			}},
		},
		Argv: []string{target, "--version"},
	}, Request{TargetPath: target}, Result{
		Stdout:     "execv errno (13)",
		Stderr:     "Rust panic at main.rs:210 internal unreachable code",
		StartError: "",
	})

	require.Len(t, report.TargetData.DelegateTargets, 1)
	require.Equal(t, "/tmp/xtensa-esp32s3-elf-gcc.real", report.TargetData.DelegateTargets[0].Path)
	require.Equal(t, "EACCES", report.Result.ChildExecErrno)
	require.Contains(t, strings.Join(report.Hints, " "), "delegate path existence")
	require.Contains(t, strings.Join(report.Hints, " "), "EACCES")
	require.Contains(t, strings.Join(report.Hints, " "), "not executable")
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

func directExecInspection(target, machine string) func(string) (aclscan.Inspection, error) {
	return func(string) (aclscan.Inspection, error) {
		return aclscan.Inspection{
			Path:                 target,
			Exists:               true,
			IsELF:                true,
			Machine:              machine,
			FileType:             "EXEC",
			LooksLikeLinuxTarget: false,
		}, nil
	}
}

func rustLauncherInspection(target, machine string) func(string) (aclscan.Inspection, error) {
	return func(string) (aclscan.Inspection, error) {
		return aclscan.Inspection{
			Path:                  target,
			Exists:                true,
			IsELF:                 true,
			Machine:               machine,
			FileType:              "EXEC",
			Interpreter:           "/lib64/ld-linux-aarch64.so.1",
			LooksLikeLinuxTarget:  true,
			LooksLikeRustLauncher: true,
		}, nil
	}
}

func explicitLoaderInspection(target, machine string) func(string) (aclscan.Inspection, error) {
	return func(string) (aclscan.Inspection, error) {
		return explicitLoaderInspectionWithMachine(target, machine), nil
	}
}

func explicitLoaderInspectionWithMachine(target, machine string) aclscan.Inspection {
	return aclscan.Inspection{
		Path:                 target,
		Exists:               true,
		IsELF:                true,
		Machine:              machine,
		FileType:             "EXEC",
		Interpreter:          "/lib64/ld-linux-aarch64.so.1",
		LooksLikeLinuxTarget: true,
	}
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
