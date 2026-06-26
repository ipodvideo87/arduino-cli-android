package install

import (
	"context"
	"errors"
	"testing"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/stretchr/testify/require"
)

type fakeExecutor struct {
	results map[StageName]StageResult
}

func (f fakeExecutor) Execute(_ context.Context, req StageRequest) (StageResult, error) {
	if result, ok := f.results[req.Stage]; ok {
		return result, nil
	}
	return StageResult{Status: diagnostics.StatusPassed}, nil
}

func TestAndroidInstallPatchPipelineTracksStagesAndFixes(t *testing.T) {
	manifest := &PatchManifest{
		PackageName:    "esp32",
		PackageVersion: "3.3.10",
		Source:         "core install",
	}
	pipeline := NewAndroidInstallPatchPipeline(fakeExecutor{
		results: map[StageName]StageResult{
			StagePermissionRuntimeFixes: {
				Status:  diagnostics.StatusWarning,
				Message: "patched builtin loader permissions",
				Fixes: []PatchFix{{
					Path:   ".acl/runtime/ld-linux-aarch64.so.1",
					Action: "repair-permissions",
					Reason: "loader executable was present but not executable",
					Status: diagnostics.StatusWarning,
				}},
			},
		},
	})

	require.NoError(t, pipeline.Run(context.Background(), manifest))
	require.Equal(t, diagnostics.StatusWarning, manifest.Status)
	require.Equal(t, len(DefaultStages()), len(manifest.Stages))

	stage, ok := manifest.Stage(StagePermissionRuntimeFixes)
	require.True(t, ok)
	require.Equal(t, diagnostics.StatusWarning, stage.Status)
	require.Contains(t, manifest.Summary(), ".acl/runtime/ld-linux-aarch64.so.1")
	require.Contains(t, manifest.Summary(), "patched builtin loader permissions")
}

func TestAndroidInstallPatchPipelineRequiresExecutor(t *testing.T) {
	pipeline := NewAndroidInstallPatchPipeline(nil)
	err := pipeline.Run(context.Background(), &PatchManifest{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "stage executor")
}

func TestPatchManifestCloneIsIndependent(t *testing.T) {
	manifest := PatchManifest{
		PackageName: "board",
		Stages: []PatchStage{{
			Name:    StageDownload,
			Status:  diagnostics.StatusPassed,
			Message: "ok",
		}},
		Metadata: map[string]string{"source": "install"},
	}
	clone := manifest.Clone()
	clone.Metadata["source"] = "edited"
	clone.Stages[0].Message = "changed"
	require.Equal(t, "install", manifest.Metadata["source"])
	require.Equal(t, "ok", manifest.Stages[0].Message)
}

func TestAndroidInstallPatchPipelineFailsOnExecutorError(t *testing.T) {
	expectedErr := errors.New("open failed")
	pipeline := NewAndroidInstallPatchPipeline(errorExecutor{err: expectedErr})
	manifest := &PatchManifest{PackageName: "tool"}

	err := pipeline.Run(context.Background(), manifest)
	require.ErrorIs(t, err, expectedErr)
	require.Equal(t, diagnostics.StatusFailed, manifest.Status)
}

type errorExecutor struct {
	err error
}

func (e errorExecutor) Execute(context.Context, StageRequest) (StageResult, error) {
	return StageResult{}, e.err
}
