package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	aclruntime "github.com/arduino/arduino-cli/internal/acl/runtime"
	"github.com/stretchr/testify/require"
)

func TestRunRejectsMissingTarget(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := run([]string{"--runtime-root", t.TempDir()}, &stdout, &stderr)
	require.NotEqual(t, 0, rc)
	require.Contains(t, stderr.String(), "missing target executable")
}

func TestRunDryRunDefault(t *testing.T) {
	root := t.TempDir()
	installRuntimeFixture(t, root, "acl-exec-runtime")

	target := mustExecutable(t)
	var stdout, stderr bytes.Buffer
	rc := run([]string{"--runtime-root", root, "--target", target, "--", "--version"}, &stdout, &stderr)
	require.Equal(t, 0, rc)
	require.Contains(t, stdout.String(), "ACL Execution Planner")
	require.Contains(t, stdout.String(), "Apply mode: false")
	require.Empty(t, stderr.String())
}

func TestRunApplyIsExplicitAndExperimental(t *testing.T) {
	root := t.TempDir()
	installRuntimeFixture(t, root, "acl-exec-runtime")

	target := mustExecutable(t)
	var stdout, stderr bytes.Buffer
	rc := run([]string{"--runtime-root", root, "--target", target, "--apply", "--", "--version"}, &stdout, &stderr)
	require.NotEqual(t, 0, rc)
	require.Contains(t, stdout.String(), "Apply mode: true")
	require.Contains(t, stderr.String(), "execution backend not implemented")
}

func installRuntimeFixture(t *testing.T, root, runtimeID string) {
	t.Helper()

	exe := mustExecutable(t)
	packageDir := filepath.Join(t.TempDir(), runtimeID)
	require.NoError(t, os.MkdirAll(filepath.Join(packageDir, "loader"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(packageDir, "lib"), 0o755))

	loaderPath := filepath.Join(packageDir, "loader", "ld-linux-test.so")
	libPath := filepath.Join(packageDir, "lib", "libacl-test.so")
	copyFile(t, exe, loaderPath)
	copyFile(t, exe, libPath)

	loaderHash := fileHash(t, loaderPath)
	libHash := fileHash(t, libPath)
	manifest := aclruntime.Manifest{
		SchemaVersion:      "1.0",
		RuntimeID:          runtimeID,
		RuntimeVersion:     "0.1.0",
		Architecture:       hostArchitecture(t, exe),
		SupportedABIs:      []string{hostABI(runtime.GOARCH)},
		CompatibilityLevel: "stable",
		CreatedAt:          fixedTime(),
		Loader: aclruntime.RuntimeFile{
			Name:     "ld-linux-test.so",
			Path:     "loader/ld-linux-test.so",
			Kind:     "loader",
			Required: true,
			SHA256:   loaderHash,
		},
		Libraries: []aclruntime.RuntimeFile{{
			Name:     "libacl-test.so",
			Path:     "lib/libacl-test.so",
			Kind:     "library",
			Required: true,
			SHA256:   libHash,
		}},
		Hashes: map[string]string{
			"loader/ld-linux-test.so": loaderHash,
			"lib/libacl-test.so":      libHash,
		},
		Build: aclruntime.BuildInfo{
			Tool:      "acl-exec-cli-test",
			Builder:   "test",
			GoVersion: runtime.Version(),
			BuiltAt:   fixedTime(),
			HostOS:    runtime.GOOS,
			HostArch:  runtime.GOARCH,
		},
	}
	writeManifest(t, filepath.Join(packageDir, aclruntime.ManifestFileName), manifest)

	mgr := aclruntime.NewManager(root)
	installed, err := mgr.InstallFromDir(packageDir)
	require.NoError(t, err)
	require.NoError(t, mgr.Activate(installed.ID))
}

func writeManifest(t *testing.T, path string, manifest aclruntime.Manifest) {
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

func fixedTime() string {
	return time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
}

func mustExecutable(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	require.NoError(t, err)
	return exe
}

func sha256Hex(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}
