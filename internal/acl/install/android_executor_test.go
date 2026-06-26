package install

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/stretchr/testify/require"
)

func TestAndroidPatchExecutorRepairsRuntimePermissions(t *testing.T) {
	root := t.TempDir()
	manifest := &PatchManifest{
		PackageName: "builtin-tool",
		Source:      "tool install",
	}
	pipeline := NewAndroidInstallPatchPipeline(NewAndroidPatchExecutor(root, false))

	require.NoError(t, pipeline.Run(context.Background(), manifest))
	require.Equal(t, diagnostics.StatusPassed, manifest.Status)
	stage, ok := manifest.Stage(StagePermissionRuntimeFixes)
	require.True(t, ok)
	require.Equal(t, diagnostics.StatusPassed, stage.Status)

	info, err := os.Stat(filepath.Join(root, ".acl", "runtime", "ld-linux-aarch64.so.1"))
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&0o111)
}
