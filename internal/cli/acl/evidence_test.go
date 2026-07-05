package acl

import (
	"bytes"
	"context"
	"testing"

	"github.com/arduino/arduino-cli/internal/acl/evidence"
	"github.com/stretchr/testify/require"
)

type fakeEvidenceCollector struct {
	bundle evidence.EvidenceBundle
	err    error
}

func (f fakeEvidenceCollector) Collect(context.Context, evidence.CollectOptions) (evidence.EvidenceBundle, error) {
	return f.bundle, f.err
}

func TestEvidenceCollectCommandPrintsArtifactPath(t *testing.T) {
	oldFactory := newEvidenceCollector
	t.Cleanup(func() { newEvidenceCollector = oldFactory })
	newEvidenceCollector = func() evidenceCollector {
		return fakeEvidenceCollector{
			bundle: evidence.EvidenceBundle{
				SchemaVersion: evidence.SchemaVersion,
				OutputPath:    "/tmp/native-termux-evidence.json",
				Repository: evidence.RepositoryIdentity{
					Root:   "/root/arduino-cli-android",
					Branch: "android-runtime-v2",
					Commit: "deadbeef",
				},
				Binary:  evidence.BinaryIdentity{Path: "/root/arduino-cli-android/arduino-cli"},
				Status:  "warning",
				Summary: "native-termux evidence collection completed",
			},
		}
	}

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "evidence", "collect", "--device", "/dev/bus/usb/001/002"})

	require.NoError(t, root.Execute())
	output := buf.String()
	require.Contains(t, output, "ACL Evidence Collector")
	require.Contains(t, output, "/tmp/native-termux-evidence.json")
	require.Contains(t, output, "/root/arduino-cli-android")
	require.Contains(t, output, "android-runtime-v2")
}
