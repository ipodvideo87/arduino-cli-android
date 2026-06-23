package android

import (
	"debug/elf"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeOSSuffix(t *testing.T) {
	require.Equal(t, "linux", RuntimeOSSuffix("android"))
}

func TestInstallRuntimeCopiesEmbeddedFiles(t *testing.T) {
	root := t.TempDir()
	runtimeDir, err := installRuntime(root)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(runtimeDir, "ld-linux-aarch64.so.1"))
	require.FileExists(t, filepath.Join(runtimeDir, "libc.so.6"))
	require.FileExists(t, filepath.Join(runtimeDir, "libm.so.6"))
	require.FileExists(t, filepath.Join(runtimeDir, "libgcc_s.so.1"))
	info, err := os.Stat(filepath.Join(runtimeDir, "ld-linux-aarch64.so.1"))
	require.NoError(t, err)
	require.NotZero(t, info.Mode().Perm()&0o111)

	linkInfo, err := os.Lstat(filepath.Join(runtimeDir, "libc.so"))
	require.NoError(t, err)
	require.True(t, linkInfo.Mode()&os.ModeSymlink != 0)
}

func TestPatchInstallTreePatchesPlatformTxtForAndroid(t *testing.T) {
	root := t.TempDir()
	platformTxt := filepath.Join(root, "platform.txt")
	require.NoError(t, os.WriteFile(platformTxt, []byte("/usr/bin/env python3\n"), 0o644))

	require.NoError(t, patchInstallTree(root, true, "android"))

	data, err := os.ReadFile(platformTxt)
	require.NoError(t, err)
	require.Equal(t, "env python3\n", string(data))
	require.DirExists(t, filepath.Join(root, aclDirName, aclRuntimeName))
}

func TestPatchInstallTreeLeavesPlatformTxtUntouchedForTools(t *testing.T) {
	root := t.TempDir()
	platformTxt := filepath.Join(root, "platform.txt")
	require.NoError(t, os.WriteFile(platformTxt, []byte("/usr/bin/env python3\n"), 0o644))

	require.NoError(t, patchInstallTree(root, false, "android"))

	data, err := os.ReadFile(platformTxt)
	require.NoError(t, err)
	require.Equal(t, "/usr/bin/env python3\n", string(data))
}

func TestPatchInstallTreeIgnoresScripts(t *testing.T) {
	root := t.TempDir()
	scriptPath := filepath.Join(root, "tool.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/sh\necho ok\n"), 0o755))
	require.NoError(t, patchInstallTree(root, false, "android"))
}

func TestPatchInstallTreeRepairsMissingExecuteBitsForExecutableFiles(t *testing.T) {
	root := t.TempDir()
	hostExe, err := os.Executable()
	require.NoError(t, err)

	elfPath := filepath.Join(root, "bin", "xtensa-esp-elf-gcc")
	require.NoError(t, os.MkdirAll(filepath.Dir(elfPath), 0o755))
	copyFile(t, hostExe, elfPath, 0o600)

	scriptPath := filepath.Join(root, "bin", "tool.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/sh\necho ok\n"), 0o600))

	require.NoError(t, patchInstallTree(root, false, "android"))

	requireExecutableMode(t, elfPath)
	requireExecutableMode(t, scriptPath)
}

func TestPatchInstallTreeSkipsEmptyFiles(t *testing.T) {
	root := t.TempDir()
	empty := filepath.Join(root, "bin", "empty")
	require.NoError(t, os.MkdirAll(filepath.Dir(empty), 0o755))
	require.NoError(t, os.WriteFile(empty, nil, 0o600))

	require.NoError(t, patchInstallTree(root, false, "android"))
}

func TestPatchInstallTreeSkipsOpenOCDToolPackage(t *testing.T) {
	root := filepath.Join(t.TempDir(), "tools", "openocd-esp32", "v0.12.0-esp32-20251215")
	hostExe, err := os.Executable()
	require.NoError(t, err)

	elfPath := filepath.Join(root, "bin", "openocd")
	require.NoError(t, os.MkdirAll(filepath.Dir(elfPath), 0o755))
	copyFile(t, hostExe, elfPath, 0o600)

	require.NoError(t, patchInstallTree(root, false, "android"))
}

func TestPatchInstallTreeSkipsESP32GDBToolPackage(t *testing.T) {
	root := filepath.Join(t.TempDir(), "tools", "riscv32-esp-elf-gdb", "16.3_20250913")
	hostExe, err := os.Executable()
	require.NoError(t, err)

	elfPath := filepath.Join(root, "bin", "riscv32-esp-elf-gdb")
	require.NoError(t, os.MkdirAll(filepath.Dir(elfPath), 0o755))
	copyFile(t, hostExe, elfPath, 0o600)

	require.NoError(t, patchInstallTree(root, false, "android"))
	require.NoDirExists(t, filepath.Join(root, aclDirName))
}

func TestIsOptionalAndroidDebugToolRootMatchesOpenOCDPackageLayout(t *testing.T) {
	root := filepath.Join("/data/data/com.termux/files/home/.arduino15/packages/esp32/tools/openocd-esp32/v0.12.0-esp32-20251215")
	require.True(t, isOptionalAndroidDebugToolRoot(root))
}

func TestIsOptionalAndroidDebugToolRootMatchesESP32GDBPackageLayout(t *testing.T) {
	root := filepath.Join("/data/data/com.termux/files/home/.arduino15/packages/esp32/tools/riscv32-esp-elf-gdb/16.3_20250913")
	require.True(t, isOptionalAndroidDebugToolRoot(root))
}

func TestEnsureRuntimeDependenciesAvailableAcceptsCompleteRuntime(t *testing.T) {
	runtimeDir := t.TempDir()
	for _, name := range []string{"libc.so.6", "libdl.so.2", "libm.so.6"} {
		require.NoError(t, os.WriteFile(filepath.Join(runtimeDir, name), []byte("x"), 0o644))
	}

	err := ensureRuntimeDependenciesAvailable("/tmp/cc1plus", runtimeDir, []string{"libc.so.6", "libdl.so.2", "libm.so.6"})
	require.NoError(t, err)
}

func TestEnsureRuntimeDependenciesAvailableReportsMissingLibrary(t *testing.T) {
	runtimeDir := t.TempDir()
	for _, name := range []string{"libc.so.6", "libdl.so.2"} {
		require.NoError(t, os.WriteFile(filepath.Join(runtimeDir, name), []byte("x"), 0o644))
	}

	err := ensureRuntimeDependenciesAvailable("/tmp/cc1plus", runtimeDir, []string{"libc.so.6", "libdl.so.2", "libm.so.6"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "/tmp/cc1plus")
	require.Contains(t, err.Error(), "libm.so.6")
}

func TestPatchExecutableWrapsFilePathOnMalformedELF(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "bin", "broken")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("\x7fELF"), 0o755))

	err := patchExecutable(path, filepath.Join(root, ".acl", "runtime"))
	require.Error(t, err)
	require.Contains(t, err.Error(), path)
}

func TestPatchSpecForELFSkipsNonAArch64(t *testing.T) {
	spec, ok := patchSpecForELFFields("/tmp/tool", elf.EM_X86_64, "EXEC", "/lib64/ld-linux-x86-64.so.2", []string{"libc.so.6"}, "/tmp/runtime")
	require.False(t, ok)
	require.Equal(t, patchSpec{}, spec)
}

func TestPatchSpecForELFUsesInterpreterForExecutables(t *testing.T) {
	spec, ok := patchSpecForELFFields("/tmp/tool", elf.EM_AARCH64, "EXEC", "/lib/ld-linux-aarch64.so.1", []string{"libc.so.6"}, "/tmp/runtime")
	require.True(t, ok)
	require.True(t, spec.setInterpreter)
	require.Equal(t, filepath.Join("/tmp/runtime", "ld-linux-aarch64.so.1"), spec.interpreter)
	require.Contains(t, spec.rpath, "/tmp/runtime")
}

func TestPatchSpecForELFSkipsInterpreterForSharedLibraries(t *testing.T) {
	spec, ok := patchSpecForELFFields("/tmp/tool", elf.EM_AARCH64, "DYN", "", []string{"libc.so.6"}, "/tmp/runtime")
	require.True(t, ok)
	require.False(t, spec.setInterpreter)
	require.Empty(t, spec.interpreter)
	require.Contains(t, spec.rpath, "/tmp/runtime")
}

func TestBuildRPathAddsGCCInternalSearchPaths(t *testing.T) {
	rpath := buildRPath("/opt/arduino-toolchain/libexec/gcc/aarch64-linux-android/12.2.0/cc1plus", "/tmp/runtime")
	require.Equal(t, "$ORIGIN/../../../../.acl/runtime:$ORIGIN/../../../../lib:$ORIGIN/../../../../lib64:$ORIGIN/../../../../libs:$ORIGIN/..", rpath)
}

func TestPlanPatchForELFUsesDifferentStrategyForGCCInternalBinaries(t *testing.T) {
	runtimeDir := "/tmp/runtime"
	driver := sampleAArch64ELFAnalysis("/opt/arduino-toolchain/bin/xtensa-esp-elf-g++", "bin")
	internal := sampleAArch64ELFAnalysis("/opt/arduino-toolchain/libexec/gcc/xtensa-esp-elf/14.2.0/cc1plus", "gcc-libexec")

	driverPlan := planPatchForELF(driver, runtimeDir)
	internalPlan := planPatchForELF(internal, runtimeDir)

	require.Equal(t, patchActionLoaderAndPath, driverPlan.Action)
	require.True(t, driverPlan.Spec.setInterpreter)
	require.Equal(t, patchActionWrapperLaunch, internalPlan.Action)
	require.False(t, internalPlan.Spec.setInterpreter)
	require.Contains(t, internalPlan.Reason, "patchelf --set-interpreter")
}

func TestPlanPatchForBuiltinCtagsUsesLoaderAndRPath(t *testing.T) {
	runtimeDir := "/tmp/runtime"
	analysis := sampleAArch64ELFAnalysis("/data/data/com.termux/files/home/.arduino15/packages/builtin/tools/ctags/5.8-arduino11/ctags", "other")

	plan := planPatchForELF(analysis, runtimeDir)

	require.Equal(t, patchActionLoaderAndPath, plan.Action)
	require.True(t, plan.Spec.setInterpreter)
	require.Equal(t, filepath.Join(runtimeDir, "ld-linux-aarch64.so.1"), plan.Spec.interpreter)
	require.Contains(t, plan.Spec.rpath, runtimeDir)
	require.NotEqual(t, patchActionWrapperLaunch, plan.Action)
}

func TestPlanPatchForELFIsDeterministic(t *testing.T) {
	runtimeDir := "/tmp/runtime"
	analysis := sampleAArch64ELFAnalysis("/opt/arduino-toolchain/libexec/gcc/xtensa-esp-elf/14.2.0/cc1plus", "gcc-libexec")

	first := planPatchForELF(analysis, runtimeDir)
	second := planPatchForELF(analysis, runtimeDir)

	require.Equal(t, first, second)
}

func TestApplyPatchelfPlanRejectsInterpreterRewriteForGCCInternalBinaries(t *testing.T) {
	path := "/opt/arduino-toolchain/libexec/gcc/xtensa-esp-elf/14.2.0/cc1plus"
	err := applyPatchelfPlan(path, patchSpec{
		setInterpreter: true,
		interpreter:    "/tmp/runtime/ld-linux-aarch64.so.1",
		rpath:          "/tmp/runtime",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "refusing to apply --set-interpreter")
	require.Contains(t, err.Error(), path)
}

func TestApplyWrapperLaunchCreatesLoaderWrapper(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "libexec", "gcc", "xtensa-esp-elf", "14.2.0", "cc1plus")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("original"), 0o755))

	plan := patchPlan{
		Action: patchActionWrapperLaunch,
		Analysis: elfAnalysis{
			Path:      path,
			PathClass: "gcc-libexec",
		},
		WrapperBackup:  filepath.Join(root, "libexec", "gcc", "xtensa-esp-elf", "14.2.0", ".acl", "original", "cc1plus"),
		WrapperTarget:  path,
		WrapperRuntime: filepath.Join(root, ".acl", "runtime"),
	}

	require.NoError(t, os.MkdirAll(plan.WrapperRuntime, 0o755))
	require.NoError(t, applyWrapperLaunch(plan))

	backup, err := os.ReadFile(plan.WrapperBackup)
	require.NoError(t, err)
	require.Equal(t, "original", string(backup))

	wrapper, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(wrapper), "ld-linux-aarch64.so.1")
	require.Contains(t, string(wrapper), ".acl/original/cc1plus")
	require.Contains(t, string(wrapper), "--library-path")
	require.Contains(t, string(wrapper), "unset LD_PRELOAD")
	require.Contains(t, string(wrapper), "TERMUX_PREFIX")
}

func sampleAArch64ELFAnalysis(path string, pathClass string) elfAnalysis {
	return elfAnalysis{
		Path:              path,
		PathClass:         pathClass,
		Machine:           elf.EM_AARCH64,
		FileType:          "EXEC",
		Interpreter:       "/lib/ld-linux-aarch64.so.1",
		ImportedLibraries: []string{"libdl.so.2", "libm.so.6", "libc.so.6"},
		RunPath:           "$ORIGIN/../lib",
		RPath:             "",
		ProgramHeaders: []elfProgramHeader{
			{Type: "PT_LOAD", Offset: 0x0, VAddr: 0x400000, FileSize: 0x1000, MemSize: 0x1000, Flags: "R E", Align: 0x10000},
			{Type: "PT_INTERP", Offset: 0x270, VAddr: 0x400270, FileSize: 0x1b, MemSize: 0x1b, Flags: "R", Align: 0x1},
			{Type: "PT_GNU_RELRO", Offset: 0x2000, VAddr: 0x420000, FileSize: 0x100, MemSize: 0x100, Flags: "R", Align: 0x1},
			{Type: "PT_TLS", Offset: 0x3000, VAddr: 0x430000, FileSize: 0x10, MemSize: 0x10, Flags: "R", Align: 0x8},
		},
		HasTLS:           true,
		HasGNURelro:      true,
		PageAligned:      true,
		LoadSegmentCount: 1,
	}
}

func copyFile(t *testing.T, src, dst string, mode os.FileMode) {
	t.Helper()
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(dst, data, mode))
}

func requireExecutableMode(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&0o111, "expected executable bits for %s", path)
}
