package bootstrap

import (
	"context"
	"testing"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	aclinstall "github.com/arduino/arduino-cli/internal/acl/install"
	"github.com/stretchr/testify/require"
)

type bootstrapExecutor struct{}

func (bootstrapExecutor) Execute(_ context.Context, req aclinstall.StageRequest) (aclinstall.StageResult, error) {
	switch req.Stage {
	case aclinstall.StagePermissionRuntimeFixes:
		return aclinstall.StageResult{Status: diagnostics.StatusPassed, Message: "repaired runtime permissions"}, nil
	default:
		return aclinstall.StageResult{Status: diagnostics.StatusSkipped, Message: "not applicable"}, nil
	}
}

func TestBootstrapPackageRunsPatchPipeline(t *testing.T) {
	pkg := New("/tmp/root", "esp32", "3.3.10")
	require.NoError(t, pkg.Run(context.Background(), bootstrapExecutor{}))
	require.True(t, pkg.Ready)
	require.NotEmpty(t, pkg.BeginnerSummary())
	require.NotEmpty(t, pkg.Detail)

	data, err := pkg.JSON()
	require.NoError(t, err)
	require.Contains(t, string(data), `"manifest"`)
}
