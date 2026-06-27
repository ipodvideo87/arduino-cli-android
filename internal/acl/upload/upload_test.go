package upload

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/acl/firmware"
	"github.com/arduino/arduino-cli/internal/acl/transport"
	"github.com/stretchr/testify/require"
)

func TestEngineDryRunPlansFullFlashPackage(t *testing.T) {
	packageDir := createPackageDir(t, "full-flash", true)
	report, err := NewEngine().DryRun(context.Background(), UploadRequest{PackageDir: packageDir})
	require.NoError(t, err)
	require.Equal(t, diagnostics.StatusPassed, report.Status)
	require.True(t, report.DryRun)
	require.True(t, report.Result.Ready)
	require.Equal(t, 4, report.Result.StepCount)
	require.Len(t, report.Plan.Steps, 4)
	require.Equal(t, uint32(0x1000), report.Plan.Steps[0].Offset)
	require.Equal(t, uint32(0x10000), report.Plan.Steps[len(report.Plan.Steps)-1].Offset)
	require.Contains(t, report.BeginnerSummary(), "dry-run")
	require.NotEmpty(t, report.ProfessionalDetails())
	require.Len(t, report.Progress, 1+1+4+1)
}

func TestEngineDryRunPlansAppOnlyPackage(t *testing.T) {
	packageDir := createPackageDir(t, "app-only", true)
	report, err := NewEngine().DryRun(context.Background(), UploadRequest{PackageDir: packageDir})
	require.NoError(t, err)
	require.Equal(t, diagnostics.StatusWarning, report.Status)
	require.True(t, report.Result.Ready)
	require.Equal(t, 1, report.Result.StepCount)
	require.Len(t, report.Plan.Steps, 1)
	require.Equal(t, firmware.ArtifactApplicationBinary, report.Plan.Steps[0].Artifact)
	require.Contains(t, report.Warnings, "bootloader artifact is not present in app-only package mode")
}

func TestEngineDryRunFailsWhenManifestIsMissing(t *testing.T) {
	packageDir := createPackageDir(t, "full-flash", true)
	require.NoError(t, os.Remove(filepath.Join(packageDir, "manifest.json")))

	report, err := NewEngine().DryRun(context.Background(), UploadRequest{PackageDir: packageDir})
	require.Error(t, err)
	require.Equal(t, diagnostics.StatusFailed, report.Status)
	require.False(t, report.Result.Ready)
	require.Contains(t, report.BeginnerSummary(), "manifest")
}

func TestEngineDryRunFailsWhenFlashPlanIsMissing(t *testing.T) {
	packageDir := createPackageDir(t, "full-flash", true)
	require.NoError(t, os.Remove(filepath.Join(packageDir, "flash-plan.json")))

	report, err := NewEngine().DryRun(context.Background(), UploadRequest{PackageDir: packageDir})
	require.Error(t, err)
	require.Equal(t, diagnostics.StatusFailed, report.Status)
	require.Contains(t, report.BeginnerSummary(), "flash plan")
}

func TestEngineDryRunFailsWhenArtifactIsMissing(t *testing.T) {
	packageDir := createPackageDir(t, "full-flash", true)
	pkg, err := firmware.LoadFirmwarePackage(packageDir)
	require.NoError(t, err)
	require.NoError(t, os.Remove(pkg.Manifest.Artifacts[firmware.ArtifactBootloaderBinary].Path))

	report, err := NewEngine().DryRun(context.Background(), UploadRequest{PackageDir: packageDir})
	require.Error(t, err)
	require.Equal(t, diagnostics.StatusFailed, report.Status)
	require.Contains(t, report.BeginnerSummary(), "bootloader")
}

func TestEngineDryRunReordersUnsortedFlashPlan(t *testing.T) {
	packageDir := createPackageDir(t, "full-flash", false)
	report, err := NewEngine().DryRun(context.Background(), UploadRequest{PackageDir: packageDir})
	require.NoError(t, err)
	require.Equal(t, diagnostics.StatusWarning, report.Status)
	require.False(t, report.Plan.Ordered)
	require.True(t, report.Plan.Complete)
	require.Equal(t, uint32(0x1000), report.Plan.Steps[0].Offset)
	require.Equal(t, uint32(0x10000), report.Plan.Steps[len(report.Plan.Steps)-1].Offset)
	require.Contains(t, report.Warnings, "flash plan entries were reordered for dry-run output")
}

func TestEngineDryRunDoesNotTouchTransportStream(t *testing.T) {
	packageDir := createPackageDir(t, "full-flash", true)
	session := panicUploadSession{stream: panicTransportStream{}}
	report, err := NewEngine().DryRun(context.Background(), UploadRequest{
		PackageDir: packageDir,
		Session:    session,
	})
	require.NoError(t, err)
	require.Equal(t, diagnostics.StatusPassed, report.Status)
}

func TestEngineDryRunProducesCanonicalProfessionalDetails(t *testing.T) {
	packageDir := createPackageDir(t, "full-flash", true)
	report, err := NewEngine().DryRun(context.Background(), UploadRequest{PackageDir: packageDir})
	require.NoError(t, err)
	require.Empty(t, report.Result.Professional)

	details := report.ProfessionalDetails()
	joined := strings.Join(details, "\n")
	require.Equal(t, 1, strings.Count(joined, "package dir: "))
	require.Equal(t, 1, strings.Count(joined, "dry-run: true"))
	require.Equal(t, 1, strings.Count(joined, "planned bytes: "))
	require.Contains(t, joined, "upload steps: 4")
	require.Contains(t, joined, "result status: passed")
}

func TestUploadReportJSONRoundTrip(t *testing.T) {
	report := UploadReport{
		SchemaVersion: "1",
		Status:        diagnostics.StatusWarning,
		DryRun:        true,
		Plan: UploadPlan{
			PackageDir: "pkg",
			Steps: []UploadStep{{
				Name:     "application",
				Artifact: firmware.ArtifactApplicationBinary,
				Offset:   0x10000,
			}},
		},
	}
	data, err := json.Marshal(report)
	require.NoError(t, err)
	var decoded UploadReport
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, report.Plan.PackageDir, decoded.Plan.PackageDir)
}

type panicUploadSession struct {
	stream transport.TransportStream
}

func (s panicUploadSession) Close() error { return nil }

func (s panicUploadSession) Diagnostics() UploadDiagnostics {
	return UploadDiagnostics{SchemaVersion: "1", Status: diagnostics.StatusWarning}
}

func (s panicUploadSession) Stream() (transport.TransportStream, error) {
	return s.stream, nil
}

type panicTransportStream struct{}

func (panicTransportStream) Read([]byte) (int, error) {
	panic("dry-run upload must not read transport streams")
}

func (panicTransportStream) Write([]byte) (int, error) {
	panic("dry-run upload must not write transport streams")
}

func (panicTransportStream) Close() error {
	panic("dry-run upload must not close transport streams")
}

func (panicTransportStream) Capabilities() transport.TransportStreamCapabilities {
	panic("dry-run upload must not inspect transport streams")
}

func (panicTransportStream) Diagnostics() transport.TransportStreamDiagnosticsReport {
	panic("dry-run upload must not inspect transport streams")
}

func createPackageDir(t *testing.T, packageMode string, ordered bool) string {
	t.Helper()
	root := t.TempDir()
	artifacts := createArtifacts(t, root)
	entries := []firmware.FlashPlanEntry{
		{Offset: 0x1000, Artifact: firmware.ArtifactBootloaderBinary, Path: artifacts[firmware.ArtifactBootloaderBinary].Path, Required: true},
		{Offset: 0x8000, Artifact: firmware.ArtifactPartitionTableBinary, Path: artifacts[firmware.ArtifactPartitionTableBinary].Path, Required: true},
		{Offset: 0xE000, Artifact: firmware.ArtifactBootApp0Binary, Path: artifacts[firmware.ArtifactBootApp0Binary].Path, Required: true},
		{Offset: 0x10000, Artifact: firmware.ArtifactApplicationBinary, Path: artifacts[firmware.ArtifactApplicationBinary].Path, Required: true},
	}
	if packageMode == "app-only" {
		entries = []firmware.FlashPlanEntry{
			{Offset: 0x10000, Artifact: firmware.ArtifactApplicationBinary, Path: artifacts[firmware.ArtifactApplicationBinary].Path, Required: true},
		}
		delete(artifacts, firmware.ArtifactBootloaderBinary)
		delete(artifacts, firmware.ArtifactPartitionTableBinary)
		delete(artifacts, firmware.ArtifactBootApp0Binary)
	}
	if !ordered && len(entries) > 1 {
		entries[0], entries[len(entries)-1] = entries[len(entries)-1], entries[0]
	}
	pkg := firmware.FirmwarePackage{
		Manifest: firmware.BuildManifest{
			SchemaVersion:    "1",
			PackageMode:      packageMode,
			SketchName:       "Blink",
			ProjectName:      "Blink",
			Board:            "esp32s3",
			FQBN:             "esp32:esp32:esp32s3",
			CoreVersion:      "3.3.10",
			ToolchainVersion: "14.2.0",
			Artifacts:        artifacts,
			TargetChip:       "ESP32-S3",
		},
		FlashPlan: firmware.FlashPlan{
			PackageMode: packageMode,
			TargetChip:  "ESP32-S3",
			Entries:     entries,
		},
	}
	pkg.Validation = firmware.NewBinaryValidator().Validate(pkg)
	require.NoError(t, pkg.Validate())
	packageDir := filepath.Join(root, "firmware-package")
	written, err := pkg.WriteToDir(packageDir)
	require.NoError(t, err)
	require.NoError(t, writePackageMetadata(packageDir, written))
	require.NoError(t, written.Validate())
	return packageDir
}

func writePackageMetadata(dir string, pkg firmware.FirmwarePackage) error {
	manifestPath := filepath.Join(dir, "manifest.json")
	flashPlanPath := filepath.Join(dir, "flash-plan.json")
	validationPath := filepath.Join(dir, "validation-report.json")
	analysisPath := filepath.Join(dir, "analysis.json")
	readmePath := filepath.Join(dir, "README_FLASHING.txt")

	write := func(path string, value any) error {
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(path, append(data, '\n'), 0o644)
	}

	if err := write(manifestPath, pkg.Manifest); err != nil {
		return err
	}
	if err := write(flashPlanPath, pkg.FlashPlan); err != nil {
		return err
	}
	if err := write(validationPath, pkg.Validation); err != nil {
		return err
	}
	if err := write(analysisPath, pkg.Analysis); err != nil {
		return err
	}
	if err := os.WriteFile(readmePath, []byte(pkg.Readme), 0o644); err != nil {
		return err
	}
	return nil
}

func createArtifacts(t *testing.T, root string) map[firmware.ArtifactKind]firmware.Artifact {
	t.Helper()
	kinds := []firmware.ArtifactKind{
		firmware.ArtifactApplicationBinary,
		firmware.ArtifactBootloaderBinary,
		firmware.ArtifactPartitionTableBinary,
		firmware.ArtifactBootApp0Binary,
		firmware.ArtifactELF,
		firmware.ArtifactMAP,
	}
	artifacts := make(map[firmware.ArtifactKind]firmware.Artifact, len(kinds))
	for _, kind := range kinds {
		path := filepath.Join(root, string(kind)+".bin")
		require.NoError(t, os.WriteFile(path, []byte(kind), 0o644))
		info, err := os.Stat(path)
		require.NoError(t, err)
		artifacts[kind] = firmware.Artifact{
			Kind:     kind,
			Path:     path,
			SHA256:   "hash-" + string(kind),
			Size:     info.Size(),
			Required: true,
		}
	}
	return artifacts
}
