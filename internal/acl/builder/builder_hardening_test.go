package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateRejectsTraversalAndAbsolutePaths(t *testing.T) {
	b := NewBuilder()
	spec := minimalSpec(t)

	spec.Loader.RelativePath = "../escape.so"
	require.Error(t, b.Validate(spec))

	spec = minimalSpec(t)
	spec.Loader.RelativePath = filepath.Join(string(filepath.Separator), "abs", "loader.so")
	require.Error(t, b.Validate(spec))
}

func TestValidateRejectsInvalidCompatibilityArchitectureAndABI(t *testing.T) {
	b := NewBuilder()
	spec := minimalSpec(t)

	spec.CompatibilityLevel = "unsupported"
	require.Error(t, b.Validate(spec))

	spec = minimalSpec(t)
	spec.Architecture = "mystery-arch"
	require.Error(t, b.Validate(spec))

	spec = minimalSpec(t)
	spec.SupportedABIs = []string{"android-incompatible"}
	require.Error(t, b.Validate(spec))
}

func TestPackageRejectsOutputPathCollisions(t *testing.T) {
	b := NewBuilder()
	spec := minimalSpec(t)

	workspace := t.TempDir()
	output := filepath.Join(workspace, "package")
	require.NoError(t, os.MkdirAll(output, 0o755))
	_, err := b.Package(output, spec)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")

	outputInsideInput := filepath.Join(filepath.Dir(spec.Loader.SourcePath), "nested", "package")
	_, err = b.Package(outputInsideInput, spec)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must not be inside")
}

func TestPackageRejectsSymlinkSourceAssets(t *testing.T) {
	b := NewBuilder()
	spec := minimalSpec(t)

	linkDir := t.TempDir()
	symlinkPath := filepath.Join(linkDir, "loader-link")
	require.NoError(t, os.Symlink(spec.Loader.SourcePath, symlinkPath))
	spec.Loader.SourcePath = symlinkPath

	_, err := b.Package(filepath.Join(t.TempDir(), "package"), spec)
	require.Error(t, err)
	require.Contains(t, err.Error(), "symlink source")
}

func TestVerifyRejectsPackageCorruptionAndMissingFiles(t *testing.T) {
	b := NewBuilder()
	spec := minimalSpec(t)
	missingManifest := buildPackageForHardening(t, b, spec)
	require.NoError(t, os.Remove(filepath.Join(missingManifest, ManifestFileName)))
	require.Error(t, b.Verify(missingManifest))

	invalidMetadata := buildPackageForHardening(t, b, spec)
	require.NoError(t, os.WriteFile(filepath.Join(invalidMetadata, MetadataFileName), []byte("{"), 0o644))
	require.Error(t, b.Verify(invalidMetadata))

	missingChecksums := buildPackageForHardening(t, b, spec)
	require.NoError(t, os.Remove(filepath.Join(missingChecksums, ChecksumsFileName)))
	require.Error(t, b.Verify(missingChecksums))

	missingVersion := buildPackageForHardening(t, b, spec)
	require.NoError(t, os.Remove(filepath.Join(missingVersion, VersionFileName)))
	require.Error(t, b.Verify(missingVersion))

	unsupportedVersion := buildPackageForHardening(t, b, spec)
	require.NoError(t, os.WriteFile(filepath.Join(unsupportedVersion, VersionFileName), []byte("2.0\n"), 0o644))
	require.Error(t, b.Verify(unsupportedVersion))
}

func TestVerifyRejectsEmptyPackage(t *testing.T) {
	b := NewBuilder()
	require.Error(t, b.Verify(t.TempDir()))
}

func TestVerifyRejectsTamperedPackageFile(t *testing.T) {
	b := NewBuilder()
	spec := minimalSpec(t)
	packageDir := filepath.Join(t.TempDir(), "package")
	_, err := b.Package(packageDir, spec)
	require.NoError(t, err)

	f, err := os.OpenFile(filepath.Join(packageDir, "lib", "libacl-test.so"), os.O_APPEND|os.O_WRONLY, 0)
	require.NoError(t, err)
	_, err = f.Write([]byte("tampered"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	require.Error(t, b.Verify(packageDir))
}

func TestVerifyRejectsSymlinkEscapeAttempts(t *testing.T) {
	b := NewBuilder()
	spec := minimalSpec(t)
	packageDir := filepath.Join(t.TempDir(), "package")
	_, err := b.Package(packageDir, spec)
	require.NoError(t, err)

	real := filepath.Join(packageDir, "lib", "libacl-test.so")
	require.NoError(t, os.Remove(real))
	require.NoError(t, os.Symlink("/etc/passwd", real))

	require.Error(t, b.Verify(packageDir))
}

func buildPackageForHardening(t *testing.T, b *Builder, spec PackageSpec) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "package")
	_, err := b.Package(dir, spec)
	require.NoError(t, err)
	return dir
}

func TestPackageReproducibility(t *testing.T) {
	b := NewBuilder()
	spec := minimalSpec(t)

	left := filepath.Join(t.TempDir(), "left")
	right := filepath.Join(t.TempDir(), "right")

	leftResult, err := b.Package(left, spec)
	require.NoError(t, err)
	rightResult, err := b.Package(right, spec)
	require.NoError(t, err)

	require.Equal(t, leftResult.RuntimeID, rightResult.RuntimeID)
	require.Equal(t, leftResult.PackageFiles, rightResult.PackageFiles)
	require.Equal(t, leftResult.Checksums, rightResult.Checksums)

	leftManifest, err := os.ReadFile(filepath.Join(left, ManifestFileName))
	require.NoError(t, err)
	rightManifest, err := os.ReadFile(filepath.Join(right, ManifestFileName))
	require.NoError(t, err)
	require.Equal(t, leftManifest, rightManifest)

	leftMetadata, err := os.ReadFile(filepath.Join(left, MetadataFileName))
	require.NoError(t, err)
	rightMetadata, err := os.ReadFile(filepath.Join(right, MetadataFileName))
	require.NoError(t, err)
	require.Equal(t, leftMetadata, rightMetadata)
}
