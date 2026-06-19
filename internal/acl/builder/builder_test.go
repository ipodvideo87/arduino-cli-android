package builder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	aclruntime "github.com/arduino/arduino-cli/internal/acl/runtime"
	"github.com/stretchr/testify/require"
)

func TestGenerateManifestAndPackageAreDeterministic(t *testing.T) {
	b := NewBuilder()
	spec := minimalSpec(t)

	manifest1, err := b.GenerateManifest(spec)
	require.NoError(t, err)
	manifest2, err := b.GenerateManifest(spec)
	require.NoError(t, err)

	require.Equal(t, manifest1.RuntimeID, manifest2.RuntimeID)
	require.Equal(t, manifest1.Loader.SHA256, manifest2.Loader.SHA256)
	require.Equal(t, manifest1.Libraries[0].SHA256, manifest2.Libraries[0].SHA256)
	require.NoError(t, manifest1.ValidateBasic())

	out1 := filepath.Join(t.TempDir(), "pkg1")
	out2 := filepath.Join(t.TempDir(), "pkg2")

	result1, err := b.Package(out1, spec)
	require.NoError(t, err)
	result2, err := b.Package(out2, spec)
	require.NoError(t, err)

	require.Equal(t, result1.RuntimeID, result2.RuntimeID)
	require.Equal(t, result1.Checksums, result2.Checksums)

	verifyPackageLayout(t, out1)
	verifyPackageLayout(t, out2)
	require.NoError(t, b.Verify(out1))
	require.NoError(t, b.Verify(out2))
}

func TestPackageProducesRuntimeManagerCompatibleManifest(t *testing.T) {
	b := NewBuilder()
	spec := minimalSpec(t)
	out := filepath.Join(t.TempDir(), "pkg")

	result, err := b.Package(out, spec)
	require.NoError(t, err)

	manifestPath := filepath.Join(out, ManifestFileName)
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	var manifest struct {
		RuntimeID          string `json:"runtime_id"`
		RuntimeVersion     string `json:"runtime_version"`
		Architecture       string `json:"architecture"`
		CompatibilityLevel string `json:"compatibility_level"`
	}
	require.NoError(t, json.Unmarshal(data, &manifest))
	require.Equal(t, result.RuntimeID, manifest.RuntimeID)
	require.NotEmpty(t, manifest.RuntimeVersion)
	require.NotEmpty(t, manifest.Architecture)
	require.NotEmpty(t, manifest.CompatibilityLevel)
}

func TestRejectMissingLoader(t *testing.T) {
	b := NewBuilder()
	spec := minimalSpec(t)
	require.NoError(t, os.Remove(spec.Loader.SourcePath))

	_, err := b.GenerateManifest(spec)
	require.Error(t, err)
	require.Contains(t, err.Error(), "loader")
}

func TestRejectDuplicateLibrary(t *testing.T) {
	b := NewBuilder()
	spec := minimalSpec(t)
	spec.Libraries = append(spec.Libraries, spec.Libraries[0])

	err := b.Validate(spec)
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate")
}

func TestRejectUnsupportedABI(t *testing.T) {
	b := NewBuilder()
	spec := minimalSpec(t)
	spec.SupportedABIs = []string{"android-unsupported"}

	err := b.Validate(spec)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not compatible")
}

func TestRejectMalformedMetadata(t *testing.T) {
	b := NewBuilder()
	spec := minimalSpec(t)
	out := filepath.Join(t.TempDir(), "pkg")

	_, err := b.Package(out, spec)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(out, MetadataFileName), []byte("{"), 0o644))
	require.Error(t, b.Verify(out))
}

func TestComputeHashesAreDeterministic(t *testing.T) {
	b := NewBuilder()
	spec := minimalSpec(t)
	out1 := filepath.Join(t.TempDir(), "pkg1")
	out2 := filepath.Join(t.TempDir(), "pkg2")

	_, err := b.Package(out1, spec)
	require.NoError(t, err)
	_, err = b.Package(out2, spec)
	require.NoError(t, err)

	hashes1, err := b.ComputeHashes(out1)
	require.NoError(t, err)
	hashes2, err := b.ComputeHashes(out2)
	require.NoError(t, err)

	require.Equal(t, hashes1, hashes2)
}

func minimalSpec(t *testing.T) PackageSpec {
	t.Helper()

	exe, err := os.Executable()
	require.NoError(t, err)

	root := t.TempDir()
	loaderSrc := filepath.Join(root, "loader-src")
	libSrc := filepath.Join(root, "lib-src")
	require.NoError(t, os.MkdirAll(filepath.Dir(loaderSrc), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(libSrc), 0o755))
	copyFile(t, exe, loaderSrc)
	copyFile(t, exe, libSrc)

	return PackageSpec{
		RuntimeName:        "acl-test-runtime",
		RuntimeVersion:     "0.1.0",
		Architecture:       hostArchitecture(t, exe),
		SupportedABIs:      []string{hostABI(runtime.GOARCH)},
		CompatibilityLevel: "stable",
		CreatedAt:          fixedTime(),
		Loader: SourceAsset{
			Name:         "ld-linux-test.so",
			SourcePath:   loaderSrc,
			RelativePath: "loader/ld-linux-test.so",
			Kind:         "loader",
			Required:     true,
		},
		Libraries: []SourceAsset{
			{
				Name:         "libacl-test.so",
				SourcePath:   libSrc,
				RelativePath: "lib/libacl-test.so",
				Kind:         "library",
				Required:     true,
			},
		},
		Build: aclruntimeBuildInfo(),
		Extensions: map[string]json.RawMessage{
			"notes": json.RawMessage(`{"fixture":true}`),
		},
	}
}

func verifyPackageLayout(t *testing.T, packageDir string) {
	t.Helper()
	for _, rel := range []string{ManifestFileName, MetadataFileName, ChecksumsFileName, VersionFileName, "loader/ld-linux-test.so", "lib/libacl-test.so"} {
		_, err := os.Stat(filepath.Join(packageDir, rel))
		require.NoError(t, err, rel)
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(dst, data, 0o755))
}

func hostArchitecture(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	var hdr [20]byte
	_, err = f.ReadAt(hdr[:], 0)
	require.NoError(t, err)
	switch hdr[18] {
	case 0xb7:
		return "aarch64"
	case 0x3e:
		return "x86_64"
	case 0x28:
		return "arm"
	default:
		return runtime.GOARCH
	}
}

func hostABI(goarch string) string {
	switch goarch {
	case "arm64":
		return "arm64-v8a"
	case "amd64":
		return "x86_64"
	case "386":
		return "x86"
	case "arm":
		return "armeabi-v7a"
	default:
		return goarch
	}
}

func fixedTime() string {
	return time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
}

func aclruntimeBuildInfo() aclruntime.BuildInfo {
	return aclruntime.BuildInfo{
		Tool:      "acl-runtime-builder-test",
		Builder:   "test",
		GoVersion: runtime.Version(),
		BuiltAt:   fixedTime(),
		HostOS:    runtime.GOOS,
		HostArch:  runtime.GOARCH,
	}
}
