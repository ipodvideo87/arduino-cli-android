package engine

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuilderResultSnapshotBuildInputPropagatesTargetChip(t *testing.T) {
	snapshot := BuilderResultSnapshot{
		BuildPath: filepath.Join(t.TempDir(), "build"),
		BuildProperties: map[string]string{
			"build.project_name": "sketch.ino",
			"build.mcu":          "esp32s3",
		},
	}

	input, err := snapshot.BuildInput(CompileRequest{
		SketchPath: filepath.Join(t.TempDir(), "sketch.ino"),
		FQBN:       "esp32:esp32:esp32s3",
		BuildPath:  filepath.Join(t.TempDir(), "build"),
	})
	require.NoError(t, err)
	require.Equal(t, "esp32s3", input.TargetChip)
}
