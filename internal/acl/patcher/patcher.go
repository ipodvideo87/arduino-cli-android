package patcher

import "github.com/arduino/arduino-cli/internal/android"

type Report = android.PatchPreviewReport
type Entry = android.PatchPreviewEntry
type Summary = android.PatchPreviewSummary

func DryRun(root string) (Report, error) {
	return android.PreviewPatchTree(root)
}
