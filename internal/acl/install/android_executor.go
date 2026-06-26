package install

import (
	"context"
	"fmt"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/android"
)

type AndroidPatchExecutor struct {
	Root          string
	PatchPlatform bool
}

func NewAndroidPatchExecutor(root string, patchPlatform bool) *AndroidPatchExecutor {
	return &AndroidPatchExecutor{
		Root:          root,
		PatchPlatform: patchPlatform,
	}
}

func (e *AndroidPatchExecutor) Execute(_ context.Context, req StageRequest) (StageResult, error) {
	if e == nil {
		return StageResult{}, fmt.Errorf("android patch executor is nil")
	}
	switch req.Stage {
	case StageDownload, StageExtract, StageRegister, StageSelfTest, StageReady:
		return StageResult{Status: diagnostics.StatusSkipped, Message: "not applicable to Android patching"}, nil
	case StageAndroidPatch, StageExecutableValidation:
		if e.Root == "" {
			return StageResult{Status: diagnostics.StatusFailed, Message: "install root is required"}, nil
		}
		var err error
		if e.PatchPlatform {
			err = android.PatchPlatformForAndroid(e.Root)
		} else {
			err = android.PatchToolForAndroid(e.Root)
		}
		if err != nil {
			return StageResult{Status: diagnostics.StatusFailed, Message: err.Error()}, nil
		}
		message := "android compatibility patch applied"
		if req.Stage == StageExecutableValidation {
			message = "android executable validation completed"
		}
		return StageResult{
			Status:  diagnostics.StatusPassed,
			Message: message,
			Evidence: []string{
				".acl/runtime/ld-linux-aarch64.so.1",
			},
		}, nil
	case StagePermissionRuntimeFixes:
		if e.Root == "" {
			return StageResult{Status: diagnostics.StatusFailed, Message: "install root is required"}, nil
		}
		if err := android.RepairRuntimePermissionsForAndroid(e.Root); err != nil {
			return StageResult{Status: diagnostics.StatusFailed, Message: err.Error()}, nil
		}
		message := "runtime permissions repaired"
		return StageResult{
			Status:  diagnostics.StatusPassed,
			Message: message,
			Evidence: []string{
				".acl/runtime/ld-linux-aarch64.so.1",
			},
		}, nil
	default:
		return StageResult{Status: diagnostics.StatusSkipped, Message: "stage not used by Android patching"}, nil
	}
}
