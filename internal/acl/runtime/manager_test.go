package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRuntimeManagerInstallDiscoverValidateAndSelect(t *testing.T) {
	exe, err := os.Executable()
	require.NoError(t, err)

	machine := machineNameFromELF(t, exe)

	root := t.TempDir()
	mgr := NewManager(root)

	packageDir := filepath.Join(root, "package")
	require.NoError(t, os.MkdirAll(filepath.Join(packageDir, "rootfs", "loader"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(packageDir, "rootfs", "lib"), 0o755))

	copyFile(t, exe, filepath.Join(packageDir, "rootfs", "loader", "ld-linux-test.so"))
	copyFile(t, exe, filepath.Join(packageDir, "rootfs", "lib", "libc.so.6"))
	copyFile(t, exe, filepath.Join(packageDir, "rootfs", "lib", "libdl.so.2"))

	loaderHash := fileHash(t, filepath.Join(packageDir, "rootfs", "loader", "ld-linux-test.so"))
	libHash := fileHash(t, filepath.Join(packageDir, "rootfs", "lib", "libc.so.6"))

	manifest := Manifest{
		SchemaVersion:      "1.0",
		RuntimeID:          "acl-test-runtime",
		RuntimeVersion:     "0.1.0",
		Architecture:       machine,
		SupportedABIs:      []string{runtime.GOARCH},
		Loader:             RuntimeFile{Name: "loader", Path: "rootfs/loader/ld-linux-test.so", SHA256: loaderHash, Required: true, Kind: "loader"},
		Libraries:          []RuntimeFile{{Name: "libc.so.6", Path: "rootfs/lib/libc.so.6", SHA256: libHash, Required: true, Kind: "library"}},
		CompatibilityLevel: "experimental",
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
		Build: BuildInfo{
			Tool:      "acl-runtime-manager",
			Builder:   "test",
			GoVersion: runtime.Version(),
		},
		Hashes: map[string]string{"manifest": ""},
	}

	writeManifest(t, filepath.Join(packageDir, ManifestFileName), manifest)

	installed, err := mgr.InstallFromDir(packageDir)
	require.NoError(t, err)
	require.Equal(t, manifest.RuntimeID, installed.ID)
	requireExecutableMode(t, filepath.Join(installed.Path, "rootfs", "loader", "ld-linux-test.so"))
	requireExecutableMode(t, filepath.Join(installed.Path, "rootfs", "lib", "libc.so.6"))

	discovered, err := mgr.Discover()
	require.NoError(t, err)
	require.Len(t, discovered, 1)

	report, err := mgr.Validate(manifest.RuntimeID)
	require.NoError(t, err)
	require.Equal(t, StatusExperimental, report.Status)

	selected, err := mgr.SelectCompatible(SelectionRequest{
		Architecture: machine,
		ABI:          runtime.GOARCH,
	})
	require.NoError(t, err)
	require.Equal(t, manifest.RuntimeID, selected.ID)

	require.NoError(t, mgr.Activate(manifest.RuntimeID))
	active, err := mgr.ActiveRuntimeID()
	require.NoError(t, err)
	require.Equal(t, manifest.RuntimeID, active)

	status, err := mgr.Status()
	require.NoError(t, err)
	require.Equal(t, manifest.RuntimeID, status.ActiveRuntimeID)
	require.NotEmpty(t, FormatStatus(status))
	require.NotEmpty(t, FormatValidation(report))
}

func TestDefaultRootPriorityOrder(t *testing.T) {
	originalWD, err := os.Getwd()
	require.NoError(t, err)

	homeDir := t.TempDir()
	prefixDir := t.TempDir()
	cwdDir := t.TempDir()
	envRoot := t.TempDir()

	t.Setenv("HOME", homeDir)
	t.Setenv("PREFIX", prefixDir)
	t.Setenv("ACL_RUNTIME_ROOT", envRoot)

	require.NoError(t, os.Chdir(cwdDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWD))
	})

	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".arduino-cli-android", "acl-runtime"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(prefixDir, "opt", "arduino-cli-android", "acl-runtime"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(cwdDir, "acl-runtime"), 0o755))

	root, err := DefaultRoot()
	require.NoError(t, err)
	require.Equal(t, filepath.Clean(envRoot), filepath.Clean(root))

	t.Setenv("ACL_RUNTIME_ROOT", "")
	root, err = DefaultRoot()
	require.NoError(t, err)
	require.Equal(t, filepath.Clean(filepath.Join(homeDir, ".arduino-cli-android", "acl-runtime")), filepath.Clean(root))

	require.NoError(t, os.RemoveAll(filepath.Join(homeDir, ".arduino-cli-android", "acl-runtime")))
	root, err = DefaultRoot()
	require.NoError(t, err)
	require.Equal(t, filepath.Clean(filepath.Join(prefixDir, "opt", "arduino-cli-android", "acl-runtime")), filepath.Clean(root))

	require.NoError(t, os.RemoveAll(filepath.Join(prefixDir, "opt", "arduino-cli-android", "acl-runtime")))
	root, err = DefaultRoot()
	require.NoError(t, err)
	require.Equal(t, filepath.Clean(filepath.Join(cwdDir, "acl-runtime")), filepath.Clean(root))
}

func writeManifest(t *testing.T, path string, manifest Manifest) {
	t.Helper()
	data, err := json.MarshalIndent(manifest, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(dst, data, 0o755))
}

func fileHash(t *testing.T, path string) string {
	t.Helper()
	sum, err := sha256Hex(path)
	require.NoError(t, err)
	return sum
}

func requireExecutableMode(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&0o111, "expected executable bits for %s", path)
}

func machineNameFromELF(t *testing.T, path string) string {
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
		// Fall back to the current Go arch so the test remains useful even on
		// unusual builders.
		return runtime.GOARCH
	}
}
