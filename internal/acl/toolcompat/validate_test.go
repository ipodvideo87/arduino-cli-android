package toolcompat

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateReportAcceptsLoaderAndRPathExecutable(t *testing.T) {
	report := Report{
		Root:    "/tmp/packages",
		Summary: Summary{TotalEntries: 1},
		Entries: []Entry{{
			RelativePath:          "tools/host-tool",
			ExecutableType:        "elf",
			CompatibilityCategory: CategoryLinuxGlibc,
			PatchClass:            PatchClassLoaderAndRPath,
			Architecture:          "AArch64",
			Interpreter:           "/lib/ld-linux-aarch64.so.1",
			SharedLibraries:       []string{"libc.so.6"},
			RPath:                 "$ORIGIN/../lib",
			RequiresRuntime:       true,
		}},
	}

	validation := ValidateReport(report)
	require.True(t, validation.Summary.Passed)
	require.Equal(t, 1, validation.Summary.ExecutableELFs)
	require.Equal(t, 0, validation.Summary.SharedLibraryRuntimeELFs)
	require.Equal(t, 0, validation.Summary.ScriptCount)
	require.Equal(t, 0, validation.Summary.UnsupportedIgnoredCount)
	require.Empty(t, validation.Findings)
}

func TestValidateReportRejectsLoaderAndRPathWithoutInterpreter(t *testing.T) {
	report := Report{
		Root:    "/tmp/packages",
		Summary: Summary{TotalEntries: 1},
		Entries: []Entry{{
			RelativePath:          "tools/host-tool",
			ExecutableType:        "elf",
			CompatibilityCategory: CategoryLinuxGlibc,
			PatchClass:            PatchClassLoaderAndRPath,
			Architecture:          "AArch64",
			SharedLibraries:       []string{"libc.so.6"},
			RPath:                 "$ORIGIN/../lib",
			RequiresRuntime:       true,
		}},
	}

	validation := ValidateReport(report)
	require.False(t, validation.Summary.Passed)
	require.Equal(t, 1, validation.Summary.Errors)
	require.NotEmpty(t, validation.Findings)
	require.Contains(t, strings.Join(validation.Findings[0].Messages, " "), "PT_INTERP")
}

func TestValidateReportAcceptsRuntimeDependencyOnlySharedLibrary(t *testing.T) {
	report := Report{
		Root:    "/tmp/packages",
		Summary: Summary{TotalEntries: 1},
		Entries: []Entry{{
			RelativePath:          "tools/libdep.so",
			ExecutableType:        "elf",
			CompatibilityCategory: CategoryUnsupported,
			PatchClass:            PatchClassRuntimeDependency,
			Architecture:          "AArch64",
			SharedLibraries:       []string{"libc.so.6"},
			RunPath:               "$ORIGIN",
			RequiresRuntime:       true,
		}},
	}

	validation := ValidateReport(report)
	require.True(t, validation.Summary.Passed)
	require.Equal(t, 1, validation.Summary.SharedLibraryRuntimeELFs)
}

func TestValidateReportRejectsRuntimeDependencyWithInterpreter(t *testing.T) {
	report := Report{
		Root:    "/tmp/packages",
		Summary: Summary{TotalEntries: 1},
		Entries: []Entry{{
			RelativePath:          "tools/libdep.so",
			ExecutableType:        "elf",
			CompatibilityCategory: CategoryUnsupported,
			PatchClass:            PatchClassRPathOnly,
			Architecture:          "AArch64",
			Interpreter:           "/lib/ld-linux-aarch64.so.1",
			SharedLibraries:       []string{"libc.so.6"},
			RunPath:               "$ORIGIN",
			RequiresRuntime:       true,
		}},
	}

	validation := ValidateReport(report)
	require.False(t, validation.Summary.Passed)
	require.Equal(t, 1, validation.Summary.Errors)
	require.Contains(t, strings.Join(validation.Findings[0].Messages, " "), "did not expect PT_INTERP")
}

func TestValidateReportAcceptsScriptNoELFPatch(t *testing.T) {
	report := Report{
		Root:    "/tmp/packages",
		Summary: Summary{TotalEntries: 1},
		Entries: []Entry{{
			RelativePath:          "tools/tool.sh",
			ExecutableType:        "shell-script",
			CompatibilityCategory: CategoryScript,
			PatchClass:            PatchClassScript,
		}},
	}

	validation := ValidateReport(report)
	require.True(t, validation.Summary.Passed)
	require.Equal(t, 1, validation.Summary.ScriptCount)
}

func TestValidateReportIgnoresUnsupportedFirmwareELF(t *testing.T) {
	report := Report{
		Root:    "/tmp/packages",
		Summary: Summary{TotalEntries: 1},
		Entries: []Entry{{
			RelativePath:          "packages/esp32/tools/firmware.elf",
			ExecutableType:        "elf",
			CompatibilityCategory: CategoryUnsupported,
			PatchClass:            PatchClassUnsupported,
			Architecture:          "Xtensa",
		}},
	}

	validation := ValidateReport(report)
	require.True(t, validation.Summary.Passed)
	require.Equal(t, 1, validation.Summary.UnsupportedIgnoredCount)
	require.Empty(t, validation.Findings)
}

func TestValidateReportErrorsOnUnsupportedHostExecutable(t *testing.T) {
	report := Report{
		Root:    "/tmp/packages",
		Summary: Summary{TotalEntries: 1},
		Entries: []Entry{{
			RelativePath:          "tools/unknown-tool",
			ExecutableType:        "binary",
			CompatibilityCategory: CategoryUnsupported,
			PatchClass:            PatchClassUnsupported,
		}},
	}

	validation := ValidateReport(report)
	require.False(t, validation.Summary.Passed)
	require.Equal(t, 1, validation.Summary.Errors)
	require.Contains(t, strings.Join(validation.Findings[0].Messages, " "), "host executable")
}
