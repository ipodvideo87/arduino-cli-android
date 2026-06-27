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

func TestExecutorPrepareOnlyBuildsCanonicalReport(t *testing.T) {
	packageDir := createPackageDir(t, "full-flash", true)
	report, err := NewExecutor().PrepareOnly(context.Background(), UploadExecutionRequest{
		PackageDir: packageDir,
	})
	require.NoError(t, err)
	require.Equal(t, diagnostics.StatusPassed, report.Status)
	require.True(t, report.DryRun)
	require.True(t, report.PrepareOnly)
	require.Equal(t, "1", report.SchemaVersion)
	require.Equal(t, packageDir, report.Plan.PackageDir)
	require.Len(t, report.Plan.Operations, 4)
	require.True(t, report.Plan.Ordered)
	require.True(t, report.Plan.Complete)
	require.NotEmpty(t, report.BeginnerSummary())
	require.NotEmpty(t, report.ProfessionalDetails())
	require.NotContains(t, strings.Join(report.ProfessionalDetails(), "\n"), report.BeginnerSummary())
	require.Equal(t, UploadExecutionPhaseInspecting, report.Progress[0].Phase)
	require.Equal(t, UploadExecutionPhaseByteWriteStopped, report.Progress[len(report.Progress)-1].Phase)
}

func TestExecutorPrepareOnlyReportsMissingArtifact(t *testing.T) {
	packageDir := createPackageDir(t, "full-flash", true)
	pkg, err := firmware.LoadFirmwarePackage(packageDir)
	require.NoError(t, err)
	require.NoError(t, os.Remove(pkg.Manifest.Artifacts[firmware.ArtifactBootloaderBinary].Path))

	report, err := NewExecutor().PrepareOnly(context.Background(), UploadExecutionRequest{
		PackageDir: packageDir,
	})
	require.Error(t, err)
	require.Equal(t, diagnostics.StatusFailed, report.Status)
	require.Contains(t, report.BeginnerSummary(), "bootloader")
}

func TestExecutorPrepareOnlyReportsHashMismatch(t *testing.T) {
	packageDir := createPackageDir(t, "full-flash", true)
	pkg, err := firmware.LoadFirmwarePackage(packageDir)
	require.NoError(t, err)
	pkg.Manifest.Artifacts[firmware.ArtifactApplicationBinary] = firmware.Artifact{
		Kind:     firmware.ArtifactApplicationBinary,
		Path:     pkg.Manifest.Artifacts[firmware.ArtifactApplicationBinary].Path,
		SHA256:   "deadbeef",
		Size:     pkg.Manifest.Artifacts[firmware.ArtifactApplicationBinary].Size,
		Required: true,
	}
	data, err := json.MarshalIndent(pkg.Manifest, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(packageDir, "manifest.json"), append(data, '\n'), 0o644))

	report, err := NewExecutor().PrepareOnly(context.Background(), UploadExecutionRequest{
		PackageDir: packageDir,
	})
	require.Error(t, err)
	require.Equal(t, diagnostics.StatusFailed, report.Status)
	require.Contains(t, strings.ToLower(report.BeginnerSummary()), "hash")
}

func TestExecutorPrepareOnlyReordersUnsortedPlan(t *testing.T) {
	packageDir := createPackageDir(t, "full-flash", false)
	report, err := NewExecutor().PrepareOnly(context.Background(), UploadExecutionRequest{
		PackageDir: packageDir,
	})
	require.NoError(t, err)
	require.Equal(t, diagnostics.StatusWarning, report.Status)
	require.Contains(t, strings.Join(report.Warnings, "\n"), "reordered")
	require.Equal(t, uint32(0x1000), report.Plan.Operations[0].Offset)
	require.Equal(t, uint32(0x10000), report.Plan.Operations[len(report.Plan.Operations)-1].Offset)
}

func TestExecutorPrepareOnlyRequiresStreamWhenRequested(t *testing.T) {
	packageDir := createPackageDir(t, "full-flash", true)
	report, err := NewExecutor().PrepareOnly(context.Background(), UploadExecutionRequest{
		PackageDir:    packageDir,
		RequireStream: true,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "transport stream capability is required")
	require.Equal(t, diagnostics.StatusFailed, report.Status)
}

func TestExecutorPrepareOnlyAcceptsStreamCapabilityWithoutWriting(t *testing.T) {
	packageDir := createPackageDir(t, "full-flash", true)
	session := testExecutionSession{
		diag: UploadExecutionSessionDiagnostics{
			SchemaVersion:   "1",
			Status:          diagnostics.StatusPassed,
			StreamSupported: true,
			StreamAvailable: true,
			Beginner:        "transport session ready",
			Professional:    []string{"stream supported: true"},
		},
		stream: panicTransportStream{},
	}
	report, err := NewExecutor().PrepareOnly(context.Background(), UploadExecutionRequest{
		PackageDir:    packageDir,
		RequireStream: true,
		Session:       session,
	})
	require.NoError(t, err)
	require.Equal(t, diagnostics.StatusPassed, report.Status)
	require.True(t, report.Plan.StreamAvailable)
	require.True(t, report.Plan.StreamRequired)
	require.Equal(t, "transport stream capability ready", report.Plan.StreamStatus)
}

func TestExecutorPrepareOnlyJSONRoundTrip(t *testing.T) {
	report := UploadExecutionReport{
		SchemaVersion: "1",
		Status:        diagnostics.StatusWarning,
		DryRun:        true,
		PrepareOnly:   true,
		Plan: UploadExecutionPlan{
			SchemaVersion: "1",
			PackageDir:    "pkg",
			Operations: []UploadOperation{{
				Name:     "application",
				Artifact: "application.bin",
				Offset:   0x10000,
				Path:     "/tmp/application.bin",
				Size:     123,
				SHA256:   "abc",
			}},
		},
		Beginner: "upload execution prepared with warnings",
	}
	data, err := json.Marshal(report)
	require.NoError(t, err)
	var decoded UploadExecutionReport
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, report.Plan.PackageDir, decoded.Plan.PackageDir)
	require.Equal(t, report.PrepareOnly, decoded.PrepareOnly)
}

type testExecutionSession struct {
	diag   UploadExecutionSessionDiagnostics
	stream transport.TransportStream
}

func (s testExecutionSession) Close() error { return nil }

func (s testExecutionSession) Diagnostics() UploadExecutionSessionDiagnostics { return s.diag }

func (s testExecutionSession) Stream() (transport.TransportStream, error) { return s.stream, nil }
