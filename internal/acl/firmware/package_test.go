package firmware

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/stretchr/testify/require"
)

func TestBuildManifestValidateRequiresCoreFields(t *testing.T) {
	manifest := BuildManifest{}
	require.Error(t, manifest.Validate())

	manifest.Board = "esp32s3"
	manifest.FQBN = "esp32:esp32:esp32s3"
	manifest.CoreVersion = "3.3.10"
	manifest.ToolchainVersion = "14.2.0"
	manifest.Artifacts = sampleArtifacts(t)

	require.NoError(t, manifest.Validate())
}

func TestFlashPlanValidateRequiresRequiredArtifacts(t *testing.T) {
	plan := FlashPlan{
		Entries: []FlashPlanEntry{
			{Offset: 0x1000, Artifact: ArtifactApplicationBinary, Path: "app.bin"},
			{Offset: 0x8000, Artifact: ArtifactPartitionTableBinary, Path: "partitions.bin"},
		},
	}
	require.Error(t, plan.Validate())

	plan.Entries = append(plan.Entries, FlashPlanEntry{
		Offset:   0x10000,
		Artifact: ArtifactBootloaderBinary,
		Path:     "bootloader.bin",
	})
	require.NoError(t, plan.Validate())
	require.Equal(t, uint32(0x1000), plan.SortedEntries()[0].Offset)
}

func TestFirmwarePackageValidateAndBinaryValidator(t *testing.T) {
	root := t.TempDir()
	artifacts := createFirmwareArtifacts(t, root)
	pkg := FirmwarePackage{
		Manifest: BuildManifest{
			PackageMode:      "full-flash",
			Board:            "esp32s3",
			FQBN:             "esp32:esp32:esp32s3",
			CoreVersion:      "3.3.10",
			ToolchainVersion: "14.2.0",
			Artifacts:        artifacts,
			MemoryUsage: MemoryUsage{
				ProgramUsedBytes:  1033681,
				ProgramTotalBytes: 1310720,
				ProgramPercent:    78,
				RAMUsedBytes:      44948,
				RAMTotalBytes:     327680,
				RAMPercent:        13,
			},
			TargetChip: "ESP32-S3",
		},
		FlashPlan: FlashPlan{
			PackageMode: "full-flash",
			TargetChip:  "ESP32-S3",
			Entries: []FlashPlanEntry{
				{Offset: 0x1000, Artifact: ArtifactBootloaderBinary, Path: artifacts[ArtifactBootloaderBinary].Path},
				{Offset: 0x8000, Artifact: ArtifactPartitionTableBinary, Path: artifacts[ArtifactPartitionTableBinary].Path},
				{Offset: 0xE000, Artifact: ArtifactBootApp0Binary, Path: artifacts[ArtifactBootApp0Binary].Path},
				{Offset: 0x10000, Artifact: ArtifactApplicationBinary, Path: artifacts[ArtifactApplicationBinary].Path},
			},
		},
	}

	require.NoError(t, pkg.Validate())

	validator := NewBinaryValidator()
	report := validator.Validate(pkg)
	require.Equal(t, diagnostics.StatusPassed, report.Status)
	require.Empty(t, report.Errors)
	require.Contains(t, report.Checks[0].Name, "manifest")
	require.Contains(t, report.Checks[1].Name, "flash-plan")
	require.Equal(t, "ESP32-S3", report.TargetChip)
	require.Equal(t, "full-flash", report.PackageMode)
}

func TestBinaryValidatorFailsForMissingArtifactFile(t *testing.T) {
	root := t.TempDir()
	artifacts := createFirmwareArtifacts(t, root)
	missingPath := filepath.Join(root, "missing.bin")
	artifacts[ArtifactApplicationBinary] = Artifact{
		Kind:     ArtifactApplicationBinary,
		Path:     missingPath,
		SHA256:   "deadbeef",
		Required: true,
	}

	pkg := FirmwarePackage{
		Manifest: BuildManifest{
			Board:            "uno",
			FQBN:             "arduino:avr:uno",
			CoreVersion:      "1.8.6",
			ToolchainVersion: "7.3.0",
			Artifacts:        artifacts,
		},
		FlashPlan: FlashPlan{
			Entries: []FlashPlanEntry{
				{Offset: 0x0, Artifact: ArtifactApplicationBinary, Path: missingPath},
				{Offset: 0x1000, Artifact: ArtifactBootloaderBinary, Path: artifacts[ArtifactBootloaderBinary].Path},
				{Offset: 0x8000, Artifact: ArtifactPartitionTableBinary, Path: artifacts[ArtifactPartitionTableBinary].Path},
			},
		},
	}

	report := NewBinaryValidator().Validate(pkg)
	require.Equal(t, diagnostics.StatusFailed, report.Status)
	require.NotEmpty(t, report.Errors)
	require.True(t, report.HasFailures())
}

func TestBinaryValidatorWarnsForAppOnlyPackageWithoutBootloader(t *testing.T) {
	root := t.TempDir()
	artifacts := createFirmwareArtifacts(t, root)
	delete(artifacts, ArtifactBootloaderBinary)
	delete(artifacts, ArtifactBootApp0Binary)

	pkg := FirmwarePackage{
		Manifest: BuildManifest{
			PackageMode:      "app-only",
			Board:            "mkr1000",
			FQBN:             "arduino:samd:mkr1000",
			CoreVersion:      "1.8.13",
			ToolchainVersion: "11.3.1",
			Artifacts:        artifacts,
		},
		FlashPlan: FlashPlan{
			PackageMode: "app-only",
			Entries: []FlashPlanEntry{
				{Offset: 0x10000, Artifact: ArtifactApplicationBinary, Path: artifacts[ArtifactApplicationBinary].Path},
			},
		},
	}

	report := NewBinaryValidator().Validate(pkg)
	require.Equal(t, diagnostics.StatusWarning, report.Status)
	require.Contains(t, report.Warnings[0], "bootloader")
}

func TestFirmwarePackageArtifactLookup(t *testing.T) {
	artifacts := sampleArtifacts(t)
	pkg := FirmwarePackage{
		Manifest: BuildManifest{
			Board:            "mkr1000",
			FQBN:             "arduino:samd:mkr1000",
			CoreVersion:      "1.8.13",
			ToolchainVersion: "11.3.1",
			Artifacts:        artifacts,
		},
	}

	artifact, ok := pkg.Artifact(ArtifactApplicationBinary)
	require.True(t, ok)
	require.Equal(t, "application-binary.bin", filepath.Base(artifact.Path))
	require.Equal(t, []ArtifactKind{
		ArtifactApplicationBinary,
		ArtifactBootApp0Binary,
		ArtifactBootloaderBinary,
		ArtifactELF,
		ArtifactMAP,
		ArtifactPartitionTableBinary,
	}, pkg.ArtifactKinds())
}

func sampleArtifacts(t *testing.T) map[ArtifactKind]Artifact {
	t.Helper()
	root := t.TempDir()
	return createFirmwareArtifacts(t, root)
}

func createFirmwareArtifacts(t *testing.T, root string) map[ArtifactKind]Artifact {
	t.Helper()
	kinds := []ArtifactKind{
		ArtifactApplicationBinary,
		ArtifactBootloaderBinary,
		ArtifactPartitionTableBinary,
		ArtifactBootApp0Binary,
		ArtifactELF,
		ArtifactMAP,
	}
	artifacts := make(map[ArtifactKind]Artifact, len(kinds))
	for _, kind := range kinds {
		path := filepath.Join(root, string(kind)+".bin")
		require.NoError(t, os.WriteFile(path, []byte(kind), 0o644))
		info, err := os.Stat(path)
		require.NoError(t, err)
		artifacts[kind] = Artifact{
			Kind:     kind,
			Path:     path,
			SHA256:   "hash-" + string(kind),
			Size:     info.Size(),
			Required: true,
		}
	}
	return artifacts
}
