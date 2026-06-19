package runtime

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRuntimeManagerLifecycleWithMinimalFixture(t *testing.T) {
	root := t.TempDir()
	mgr := NewManager(root)

	experimentalID := "acl-test-runtime-exp"
	stableID := "acl-test-runtime-stable"
	hostArch := machineNameFromELF(t, mustExecutable(t))
	hostABI := hostABIForGOARCH(runtime.GOARCH)

	experimentalPkg := materializeMinimalRuntimePackage(t, experimentalID, "experimental", hostArch, hostABI)
	stablePkg := materializeMinimalRuntimePackage(t, stableID, "stable", hostArch, hostABI)

	installedExperimental, err := mgr.InstallFromDir(experimentalPkg)
	require.NoError(t, err)
	require.Equal(t, experimentalID, installedExperimental.ID)

	installedStable, err := mgr.InstallFromDir(stablePkg)
	require.NoError(t, err)
	require.Equal(t, stableID, installedStable.ID)

	discovered, err := mgr.Discover()
	require.NoError(t, err)
	require.Len(t, discovered, 2)

	validation, err := mgr.Validate(experimentalID)
	require.NoError(t, err)
	require.Equal(t, StatusExperimental, validation.Status)
	requireCheck(t, validation, "loader", StatusPass)
	requireCheck(t, validation, "loader:sha256", StatusPass)
	requireCheck(t, validation, "libacl-test.so", StatusPass)
	requireCheck(t, validation, "libacl-test.so:sha256", StatusPass)

	stableValidation, err := mgr.Validate(stableID)
	require.NoError(t, err)
	require.Equal(t, StatusPass, stableValidation.Status)

	selected, err := mgr.SelectCompatible(SelectionRequest{
		Architecture: hostArch,
		ABI:          hostABI,
	})
	require.NoError(t, err)
	require.Equal(t, stableID, selected.ID)

	selectedByABI, err := mgr.SelectCompatible(SelectionRequest{
		Architecture: hostArch,
		ABI:          hostABI,
	})
	require.NoError(t, err)
	require.Equal(t, stableID, selectedByABI.ID)

	require.NoError(t, mgr.Activate(stableID))
	activeID, err := mgr.ActiveRuntimeID()
	require.NoError(t, err)
	require.Equal(t, stableID, activeID)

	status, err := mgr.Status()
	require.NoError(t, err)
	require.Equal(t, stableID, status.ActiveRuntimeID)
	require.NotEmpty(t, FormatStatus(status))
	require.NotEmpty(t, FormatValidation(validation))

	require.NoError(t, mgr.Deactivate())
	activeID, err = mgr.ActiveRuntimeID()
	require.NoError(t, err)
	require.Empty(t, activeID)
}

func TestRuntimeManagerRejectsMissingLoader(t *testing.T) {
	root := t.TempDir()
	mgr := NewManager(root)

	hostArch := machineNameFromELF(t, mustExecutable(t))
	pkg := materializeMinimalRuntimePackage(t, "acl-test-runtime-missing-loader", "experimental", hostArch, hostABIForGOARCH(runtime.GOARCH))
	require.NoError(t, os.Remove(filepath.Join(pkg, "loader", "ld-linux-test.so")))

	var err error
	_, err = mgr.InstallFromDir(pkg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing")
}

func TestRuntimeManagerRejectsIncorrectHash(t *testing.T) {
	root := t.TempDir()
	mgr := NewManager(root)

	hostArch := machineNameFromELF(t, mustExecutable(t))
	pkg := materializeMinimalRuntimePackage(t, "acl-test-runtime-bad-hash", "experimental", hostArch, hostABIForGOARCH(runtime.GOARCH))
	f, err := os.OpenFile(filepath.Join(pkg, "lib", "libacl-test.so"), os.O_APPEND|os.O_WRONLY, 0)
	require.NoError(t, err)
	_, err = f.Write([]byte("tampered"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	_, err = mgr.InstallFromDir(pkg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sha256")
}

func TestRuntimeManagerRejectsIncompatibleABI(t *testing.T) {
	root := t.TempDir()
	mgr := NewManager(root)

	hostArch := machineNameFromELF(t, mustExecutable(t))
	hostABI := hostABIForGOARCH(runtime.GOARCH)
	pkg := materializeMinimalRuntimePackage(t, "acl-test-runtime-abi", "experimental", hostArch, hostABI)

	_, err := mgr.InstallFromDir(pkg)
	require.NoError(t, err)

	_, err = mgr.SelectCompatible(SelectionRequest{
		Architecture: hostArch,
		ABI:          "android-incompatible",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no compatible runtime")
}

func TestRuntimeManagerRejectsDuplicateRuntimeIDAndKeepsFirstInstall(t *testing.T) {
	root := t.TempDir()
	mgr := NewManager(root)

	hostArch := machineNameFromELF(t, mustExecutable(t))
	hostABI := hostABIForGOARCH(runtime.GOARCH)
	pkg := materializeMinimalRuntimePackage(t, "acl-test-runtime-duplicate", "stable", hostArch, hostABI)

	installed, err := mgr.InstallFromDir(pkg)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "runtimes", installed.ID), 0o755))

	_, err = mgr.InstallFromDir(pkg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")

	_, err = os.Stat(filepath.Join(root, "runtimes", installed.ID))
	require.NoError(t, err)
}

func TestRuntimeManagerRejectsSymlinkedPackageEntries(t *testing.T) {
	root := t.TempDir()
	mgr := NewManager(root)

	hostArch := machineNameFromELF(t, mustExecutable(t))
	hostABI := hostABIForGOARCH(runtime.GOARCH)
	pkg := materializeMinimalRuntimePackage(t, "acl-test-runtime-symlink", "stable", hostArch, hostABI)

	realLib := filepath.Join(pkg, "lib", "libacl-test.so")
	require.NoError(t, os.Remove(realLib))
	require.NoError(t, os.Symlink("/etc/passwd", realLib))

	_, err := mgr.InstallFromDir(pkg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "symlink package entry")
	_, err = os.Stat(filepath.Join(root, "runtimes", "acl-test-runtime-symlink"))
	require.Error(t, err)
}

func TestRuntimeManagerRejectsEmptyRuntimePackage(t *testing.T) {
	root := t.TempDir()
	mgr := NewManager(root)

	empty := t.TempDir()
	_, err := mgr.InstallFromDir(empty)
	require.Error(t, err)
	require.Contains(t, err.Error(), "manifest")
}

func TestActivationRejectsInvalidInstalledRuntime(t *testing.T) {
	root := t.TempDir()
	mgr := NewManager(root)

	hostArch := machineNameFromELF(t, mustExecutable(t))
	hostABI := hostABIForGOARCH(runtime.GOARCH)
	pkg := materializeMinimalRuntimePackage(t, "acl-test-runtime-activate", "stable", hostArch, hostABI)

	installed, err := mgr.InstallFromDir(pkg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(installed.Path, "lib", "libacl-test.so"), []byte("tampered"), 0o644))

	err = mgr.Activate(installed.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed validation")
	activeID, err := mgr.ActiveRuntimeID()
	require.NoError(t, err)
	require.Empty(t, activeID)
}

func TestManifestValidationRejectsAbsolutePaths(t *testing.T) {
	manifest := Manifest{
		SchemaVersion:  "1",
		RuntimeID:      "bad",
		RuntimeVersion: "1",
		Architecture:   "aarch64",
		SupportedABIs:  []string{"arm64-v8a"},
		Loader:         RuntimeFile{Name: "loader", Path: "/abs/path"},
	}
	require.Error(t, manifest.ValidateBasic())
}

func TestMinimalRuntimeFixtureManifestLoads(t *testing.T) {
	manifestPath := filepath.Join("testdata", "minimal-runtime", ManifestFileName)
	manifest, err := LoadManifest(manifestPath)
	require.NoError(t, err)
	require.NoError(t, manifest.ValidateBasic())
}

func materializeMinimalRuntimePackage(t *testing.T, runtimeID, compatibility, arch, abi string) string {
	t.Helper()

	fixDir := filepath.Join("testdata", "minimal-runtime")
	dst := filepath.Join(t.TempDir(), runtimeID)
	copyTreeForTest(t, fixDir, dst)

	exe := mustExecutable(t)
	loaderPath := filepath.Join(dst, "loader", "ld-linux-test.so")
	libraryPath := filepath.Join(dst, "lib", "libacl-test.so")
	copyFile(t, exe, loaderPath)
	copyFile(t, exe, libraryPath)

	loaderHash := fileHash(t, loaderPath)
	libraryHash := fileHash(t, libraryPath)

	manifestPath := filepath.Join(dst, ManifestFileName)
	manifest := loadManifestForTest(t, manifestPath)
	manifest.RuntimeID = runtimeID
	manifest.RuntimeVersion = "0.1.0"
	manifest.Architecture = arch
	manifest.SupportedABIs = []string{abi}
	manifest.CompatibilityLevel = compatibility
	manifest.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	manifest.Loader.Name = "loader"
	manifest.Loader.Path = "loader/ld-linux-test.so"
	manifest.Loader.SHA256 = loaderHash
	manifest.Loader.Kind = "loader"
	manifest.Loader.Required = true
	manifest.Libraries = []RuntimeFile{{
		Name:     "libacl-test.so",
		Path:     "lib/libacl-test.so",
		SONAME:   "libacl-test.so",
		SHA256:   libraryHash,
		Kind:     "library",
		Required: true,
	}}
	manifest.Hashes = map[string]string{
		"loader/ld-linux-test.so": loaderHash,
		"lib/libacl-test.so":      libraryHash,
	}
	manifest.Build.BuiltAt = manifest.CreatedAt
	manifest.Build.HostOS = runtime.GOOS
	manifest.Build.HostArch = runtime.GOARCH
	manifest.Build.Notes = fmt.Sprintf("materialized for %s", runtimeID)
	writeManifestForTest(t, manifestPath, manifest)
	return dst
}

func mustExecutable(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	require.NoError(t, err)
	return exe
}

func loadManifestForTest(t *testing.T, path string) Manifest {
	t.Helper()
	manifest, err := LoadManifest(path)
	require.NoError(t, err)
	return manifest
}

func writeManifestForTest(t *testing.T, path string, manifest Manifest) {
	t.Helper()
	data, err := json.MarshalIndent(manifest, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))
}

func copyTreeForTest(t *testing.T, src, dst string) {
	t.Helper()
	require.NoError(t, filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	}))
}

func requireCheck(t *testing.T, report ValidationReport, name, status string) {
	t.Helper()
	for _, check := range report.Checks {
		if check.Name == name {
			require.Equal(t, status, check.Status, "check %s", name)
			return
		}
	}
	t.Fatalf("missing check %q", name)
}

func hostABIForGOARCH(goarch string) string {
	switch goarch {
	case "arm64":
		return "android-aarch64"
	case "amd64":
		return "android-x86_64"
	case "386":
		return "android-i686"
	case "arm":
		return "android-arm"
	default:
		return "android-" + goarch
	}
}
